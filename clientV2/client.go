package clientV2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Yamashou/gqlgenc/graphqljson"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"golang.org/x/xerrors"
)

// HTTPRequestOption represents the options applicable to the http client
type HTTPRequestOption func(req *http.Request)

type RequestSet struct {
	HTTPRequest        *http.Request
	graphQLRequestBody *Request
}

func NewRequestSet(ctx context.Context, baseURL, operationName, query string, vars map[string]interface{}) (*RequestSet, error) {
	r := &Request{
		Query:         query,
		Variables:     vars,
		OperationName: operationName,
	}

	requestBody, err := json.Marshal(r)
	if err != nil {
		return nil, xerrors.Errorf("encode: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, xerrors.Errorf("create request struct failed: %w", err)
	}

	return &RequestSet{
		HTTPRequest:        req,
		graphQLRequestBody: r,
	}, nil
}

type Middleware func(next RequestInterceptorFunc) RequestInterceptorFunc

type RequestInterceptor func(ctx context.Context, requestSet *RequestSet, res interface{}, next RequestInterceptorFunc) error

type RequestInterceptorFunc func(ctx context.Context, requestSet *RequestSet, res interface{}) error

func ChainInterceptor(interceptors ...RequestInterceptor) RequestInterceptor {
	n := len(interceptors)

	return func(ctx context.Context, requestSet *RequestSet, res interface{}, next RequestInterceptorFunc) error {
		chainer := func(currentInter RequestInterceptor, currentFunc RequestInterceptorFunc) RequestInterceptorFunc {
			return func(currentCtx context.Context, currentRequestSet *RequestSet, currentRes interface{}) error {
				return currentInter(currentCtx, currentRequestSet, currentRes, currentFunc)
			}
		}

		chainedHandler := next
		for i := n - 1; i >= 0; i-- {
			chainedHandler = chainer(interceptors[i], chainedHandler)
		}

		return chainedHandler(ctx, requestSet, res)
	}
}

// Client is the http client wrapper
type Client struct {
	Client             *http.Client
	BaseURL            string
	Middlewares        []Middleware
	RequestInterceptor RequestInterceptor
}

// Request represents an outgoing GraphQL request
type Request struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

// NewClient creates a new http client wrapper
func NewClient(client *http.Client, baseURL string, middlewares ...Middleware) *Client {
	return &Client{
		Client:      client,
		BaseURL:     baseURL,
		Middlewares: middlewares,
	}
}

// NewClient creates a new http client wrapper
func NewClient2(client *http.Client, baseURL string, interceptors ...RequestInterceptor) *Client {
	return &Client{
		Client:  client,
		BaseURL: baseURL,
		RequestInterceptor: ChainInterceptor(append([]RequestInterceptor{func(ctx context.Context, requestSet *RequestSet, res interface{}, next RequestInterceptorFunc) error {
			return next(ctx, requestSet, res)
		}}, interceptors...)...),
	}
}

func (c *Client) Intercept(interceptor Middleware) {
	c.Middlewares = append(c.Middlewares, interceptor)
}

// GqlErrorList is the struct of a standard graphql error response
type GqlErrorList struct {
	Errors gqlerror.List `json:"errors"`
}

func (e *GqlErrorList) Error() string {
	return e.Errors.Error()
}

// HTTPError is the error when a GqlErrorList cannot be parsed
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse represent an handled error
type ErrorResponse struct {
	// populated when http status code is not OK
	NetworkError *HTTPError `json:"networkErrors"`
	// populated when http status code is OK but the server returned at least one graphql error
	GqlErrors *gqlerror.List `json:"graphqlErrors"`
}

// HasErrors returns true when at least one error is declared
func (er *ErrorResponse) HasErrors() bool {
	return er.NetworkError != nil || er.GqlErrors != nil
}

func (er *ErrorResponse) Error() string {
	content, err := json.Marshal(er)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

// Post sends a http POST request to the graphql endpoint with the given query then unpacks
// the response into the given object.
func (c *Client) Post(ctx context.Context, operationName, query string, respData interface{}, vars map[string]interface{}, middlewares ...Middleware) error {
	requestSet, err := NewRequestSet(ctx, c.BaseURL, operationName, query, vars)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	requestSet.HTTPRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	requestSet.HTTPRequest.Header.Set("Accept", "application/json; charset=utf-8")

	requestMiddlewares := append(c.Middlewares, middlewares...)

	r := c.do
	for i := len(requestMiddlewares) - 1; i >= 0; i-- {
		r = requestMiddlewares[i](r)
	}

	return r(ctx, requestSet, respData)
}

// the response into the given object.
func (c *Client) Post2(ctx context.Context, operationName, query string, respData interface{}, vars map[string]interface{}, interceptors ...RequestInterceptor) error {
	requestSet, err := NewRequestSet(ctx, c.BaseURL, operationName, query, vars)
	if err != nil {
		return xerrors.Errorf(": %w", err)
	}
	requestSet.HTTPRequest.Header.Set("Content-Type", "application/json; charset=utf-8")
	requestSet.HTTPRequest.Header.Set("Accept", "application/json; charset=utf-8")

	r := ChainInterceptor(append([]RequestInterceptor{c.RequestInterceptor}, interceptors...)...)

	return r(ctx, requestSet, respData, c.do)
}

func (c *Client) do(_ context.Context, requestSet *RequestSet, res interface{}) error {
	resp, err := c.Client.Do(requestSet.HTTPRequest)
	if err != nil {
		return xerrors.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return xerrors.Errorf("failed to read response body: %w", err)
	}

	return parseResponse(body, resp.StatusCode, res)
}

func parseResponse(body []byte, httpCode int, result interface{}) error {
	errResponse := &ErrorResponse{}
	isKOCode := httpCode < 200 || 299 < httpCode
	if isKOCode {
		errResponse.NetworkError = &HTTPError{
			Code:    httpCode,
			Message: fmt.Sprintf("Response body %s", string(body)),
		}
	}

	// some servers return a graphql error with a non OK http code, try anyway to parse the body
	if err := unmarshal(body, result); err != nil {
		if gqlErr, ok := err.(*GqlErrorList); ok {
			errResponse.GqlErrors = &gqlErr.Errors
		} else if !isKOCode { // if is KO code there is already the http error, this error should not be returned
			return err
		}
	}

	if errResponse.HasErrors() {
		return errResponse
	}

	return nil
}

// response is a GraphQL layer response from a handler.
type response struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

func unmarshal(data []byte, res interface{}) error {
	resp := response{}
	if err := json.Unmarshal(data, &resp); err != nil {
		return xerrors.Errorf("failed to decode data %s: %w", string(data), err)
	}

	if resp.Errors != nil && len(resp.Errors) > 0 {
		// try to parse standard graphql error
		errors := &GqlErrorList{}
		if e := json.Unmarshal(data, errors); e != nil {
			return xerrors.Errorf("faild to parse graphql errors. Response content %s - %w ", string(data), e)
		}

		return errors
	}

	if err := graphqljson.UnmarshalData(resp.Data, res); err != nil {
		return xerrors.Errorf("failed to decode data into response %s: %w", string(data), err)
	}

	return nil
}

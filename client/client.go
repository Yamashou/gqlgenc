package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/Yamashou/gqlgenc/v3/graphqljson"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

type Client struct {
	client   *http.Client
	endpoint string
}

// NewClient creates a new http client wrapper
func NewClient(endpoint string, options ...Option) *Client {
	client := &Client{
		client:   http.DefaultClient,
		endpoint: endpoint,
	}
	for _, option := range options {
		option(client)
	}

	return client
}

type Option func(*Client)

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.client = httpClient
	}
}

func (c *Client) Post(ctx context.Context, operationName, query string, variables map[string]any, out any, options ...Option) error {
	client := &Client{
		client:   c.client,
		endpoint: c.endpoint,
	}
	for _, option := range options {
		option(client)
	}

	req, err := newRequest(ctx, client.endpoint, operationName, query, variables)
	if err != nil {
		return fmt.Errorf("failed to create post request: %w", err)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return client.parseResponse(resp, out)
}

// PostMultipart send multipart form with files https://gqlgen.com/reference/file-upload/ https://github.com/jaydenseric/graphql-multipart-request-spec
func (c *Client) PostMultipart(ctx context.Context, operationName, query string, variables map[string]any, out any, options ...Option) error {
	client := &Client{
		client:   c.client,
		endpoint: c.endpoint,
	}
	for _, option := range options {
		option(client)
	}

	req, err := multipartRequest(ctx, client.endpoint, operationName, query, variables)
	if err != nil {
		return fmt.Errorf("failed to create post multipart request: %w", err)
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return client.parseResponse(resp, out)
}

/////////////////////////////////////////////////////////////////////
// Request

// Request represents an outgoing GraphQL request
type Request struct {
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables,omitempty"`
	OperationName string         `json:"operationName,omitempty"`
}

func newRequest(ctx context.Context, endpoint, operationName, query string, variables map[string]any) (*http.Request, error) {
	graphqlRequest := &Request{
		Query:         query,
		Variables:     variables,
		OperationName: operationName,
	}
	requestBody, err := json.Marshal(graphqlRequest)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create request struct failed: %w", err)
	}

	req.Header = http.Header{
		"Content-Type": []string{"application/graphql-response+json;charset=utf-8", "application/json; charset=utf-8"},
		"Accept":       []string{"application/graphql-response+json;charset=utf-8", "application/json; charset=utf-8"},
	}

	return req, nil
}

/////////////////////////////////////////////////////////////////////
// Response

// httpError is the error when a gqlErrors cannot be parsed
type httpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// gqlErrors is the struct of a standard graphql error response
type gqlErrors struct {
	Errors gqlerror.List `json:"errors"`
}

func (e *gqlErrors) Error() string {
	return e.Errors.Error()
}

// errorResponse represent an handled error
type errorResponse struct {
	// http status code is not OK
	NetworkError *httpError `json:"networkErrors"`
	// http status code is OK but the server returned at least one graphql error
	GqlErrors *gqlerror.List `json:"graphqlErrors"`
}

// HasErrors returns true when at least one error is declared
func (er *errorResponse) HasErrors() bool {
	return er.NetworkError != nil || er.GqlErrors != nil
}

func (er *errorResponse) Error() string {
	content, err := json.Marshal(er)
	if err != nil {
		return err.Error()
	}

	return string(content)
}

func (c *Client) parseResponse(resp *http.Response, out any) error {
	if resp.Header.Get("Content-Encoding") == "gzip" {
		respBody, err := gzip.NewReader(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to decode gzip: %w", err)
		}
		resp.Body = respBody
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	errResponse := &errorResponse{}
	isStatusCodeOK := 200 <= resp.StatusCode && resp.StatusCode <= 299
	if !isStatusCodeOK {
		errResponse.NetworkError = &httpError{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("Response body %s", string(body)),
		}
	}

	if err := c.unmarshalResponse(body, out); err != nil {
		var gqlErrs *gqlErrors
		if errors.As(err, &gqlErrs) {
			// success to parse graphql error response
			errResponse.GqlErrors = &gqlErrs.Errors
		} else if isStatusCodeOK {
			// status code is OK but the GraphQL response can't be parsed, it's an error.
			return fmt.Errorf("http status is OK but %w", err)
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

func (c *Client) unmarshalResponse(respBody []byte, out any) error {
	resp := response{}
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("failed to decode response %q: %w", respBody, err)
	}
	if err := graphqljson.UnmarshalData(resp.Data, out); err != nil {
		return fmt.Errorf("failed to decode response data %q: %w", resp.Data, err)
	}

	if len(resp.Errors) > 0 {
		gqlErrs := &gqlErrors{}
		if err := json.Unmarshal(respBody, gqlErrs); err != nil {
			return fmt.Errorf("faild to decode response error %q: %w", respBody, err)
		}
		return gqlErrs
	}

	return nil
}

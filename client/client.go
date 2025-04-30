package client

import (
	"context"
	"fmt"
	"net/http"
	"slices"
)

type Client struct {
	client   *http.Client
	endpoint string
}

// NewClient creates a new http client wrapper.
func NewClient(endpoint string, options ...Option) *Client {
	client := &Client{
		endpoint: endpoint,
		client:   http.DefaultClient,
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
	client := NewClient(c.endpoint, slices.Concat([]Option{WithHTTPClient(c.client)}, options)...)

	// PostMultipart send multipart form with files https://gqlgen.com/reference/file-upload/ https://github.com/jaydenseric/graphql-multipart-request-spec
	req, err := NewMultipartRequest(ctx, client.endpoint, operationName, query, variables)
	if err != nil {
		return fmt.Errorf("failed to create post multipart request: %w", err)
	}

	if req == nil {
		req, err = NewRequest(ctx, client.endpoint, operationName, query, variables)
		if err != nil {
			return fmt.Errorf("failed to create post request: %w", err)
		}
	}

	resp, err := client.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	return ParseResponse(resp, out)
}

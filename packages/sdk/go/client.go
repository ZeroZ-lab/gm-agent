package sdk

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gm-agent-org/gm-agent/packages/sdk/go/gen"
)

// Client wraps the generated OpenAPI client with stable helpers.
type Client struct {
	baseURL string
	apiKey  string
	client  *gen.ClientWithResponses
}

// NewClient creates a new SDK client.
func NewClient(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("server URL is empty")
	}
	if strings.HasPrefix(baseURL, ":") {
		baseURL = "http://localhost" + baseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	httpClient := &http.Client{Timeout: timeout}
	genClient, err := gen.NewClientWithResponses(baseURL, gen.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  genClient,
	}, nil
}

// Health returns the server health response.
func (c *Client) Health(ctx context.Context) (*gen.DtoHealthResponse, error) {
	resp, err := c.client.GetHealthWithResponse(ctx, c.requestEditor)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return nil, fmt.Errorf("server returned %d", resp.StatusCode())
	}
	return resp.JSON200, nil
}

func (c *Client) requestEditor(_ context.Context, req *http.Request) error {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	return nil
}

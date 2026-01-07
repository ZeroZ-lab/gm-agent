package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client wraps HTTP access to the gm-agent API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, apiKey string, timeout time.Duration) (*Client, error) {
	normalized, err := normalizeBaseURL(baseURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:    normalized,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: timeout},
	}, nil
}

// Get issues a GET request to the given path.
func (c *Client) Get(ctx context.Context, path string) (int, []byte, error) {
	req, err := c.newRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return 0, nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, body, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	target := strings.TrimRight(c.baseURL, "/")
	if path != "" {
		target = target + "/" + strings.TrimLeft(path, "/")
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	return req, nil
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("server URL is empty")
	}

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		raw = strings.TrimRight(raw, "/")
	} else if strings.HasPrefix(raw, ":") {
		raw = "http://localhost" + raw
	} else {
		raw = "http://" + raw
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("invalid server URL: %q", raw)
	}
	return strings.TrimRight(raw, "/"), nil
}

// Post issues a POST request to the given path.
func (c *Client) Post(ctx context.Context, path string, body interface{}) (int, []byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := c.newRequest(ctx, http.MethodPost, path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

type CreateSessionRequest struct {
	Prompt string `json:"prompt"`
}

type SessionResponse struct {
	ID string `json:"id"`
}

type MessageRequest struct {
	Content string `json:"content"`
}

func (c *Client) CreateSession(ctx context.Context, prompt string) (string, error) {
	status, body, err := c.Post(ctx, "/api/v1/session", CreateSessionRequest{Prompt: prompt})
	if err != nil {
		return "", err
	}
	// Allow 200/201
	if status != 201 && status != 200 {
		return "", fmt.Errorf("create session failed: status=%d body=%s", status, string(body))
	}
	var resp SessionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (c *Client) SendMessage(ctx context.Context, sessionID, content string) error {
	status, body, err := c.Post(ctx, fmt.Sprintf("/api/v1/session/%s/message", sessionID), MessageRequest{Content: content})
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("send message failed: status=%d body=%s", status, string(body))
	}
	return nil
}

type Event struct {
	Type string
	Data json.RawMessage
}

func (c *Client) StreamEvents(ctx context.Context, sessionID string) (<-chan Event, error) {
	req, err := c.newRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/session/%s/event", sessionID), nil)
	if err != nil {
		return nil, err
	}

	// Use a dedicated client without timeout for SSE (long-running connection)
	sseClient := &http.Client{} // No timeout

	resp, err := sseClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("stream events failed: status=%d", resp.StatusCode)
	}

	ch := make(chan Event)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "event:") {
				evtType := strings.TrimSpace(strings.TrimPrefix(line, "event:"))

				// Read data line
				line, err = reader.ReadString('\n')
				if err != nil {
					return
				}
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "data:") {
					data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
					ch <- Event{Type: evtType, Data: json.RawMessage(data)}
				}
			}
		}
	}()
	return ch, nil
}

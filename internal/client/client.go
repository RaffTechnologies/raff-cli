package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const Version = "0.1.0"

type Client struct {
	BaseURL    string
	APIKey     string
	AccountID  string
	HTTPClient *http.Client
}

// APIResponse is the standard envelope returned by the Raff API.
type APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
	Total   int             `json:"total,omitempty"`
}

type APIError struct {
	StatusCode int
	Body       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (%d): %s", e.StatusCode, e.Body)
}

func New(baseURL, apiKey, accountID string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		BaseURL:   baseURL,
		APIKey:    apiKey,
		AccountID: accountID,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Do(method, path string, body any) (*APIResponse, error) {
	url := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "raff-cli/"+Version)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}

	if c.AccountID != "" {
		req.Header.Set("X-Account-ID", c.AccountID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiResp APIResponse
		if json.Unmarshal(respBody, &apiResp) == nil {
			msg := apiResp.Message
			if msg == "" {
				msg = apiResp.Error
			}
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Body:       string(respBody),
				Message:    msg,
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Body:       string(respBody),
		}
	}

	var apiResp APIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return &apiResp, nil
}

func (c *Client) Get(path string) (*APIResponse, error) {
	return c.Do(http.MethodGet, path, nil)
}

func (c *Client) Post(path string, body any) (*APIResponse, error) {
	return c.Do(http.MethodPost, path, body)
}

func (c *Client) Put(path string, body any) (*APIResponse, error) {
	return c.Do(http.MethodPut, path, body)
}

func (c *Client) Delete(path string) (*APIResponse, error) {
	return c.Do(http.MethodDelete, path, nil)
}

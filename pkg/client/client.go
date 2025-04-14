package client

import (
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

const (
	defaultBaseURL = "https://pure.md"
	apiKeyHeader   = "x-puremd-api-token" // Header name remains the same per API spec
	contentType    = "application/json"
)

// Common errors
var (
	ErrUnauthorized     = errors.New("unauthorized: invalid or missing API token")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrPaymentRequired  = errors.New("payment required: this endpoint requires a paid subscription")
	ErrInvalidURL       = errors.New("invalid URL provided")
	ErrUnsupportedMedia = errors.New("unsupported media type: HTML responses only")
)

// IsInvalidURLError checks if the error is an invalid URL error
func IsInvalidURLError(err error) bool {
	return errors.Is(err, ErrInvalidURL) || 
		(err != nil && strings.Contains(err.Error(), "invalid URL provided"))
}

// Client represents a PureMD API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

// ClientOption defines a function that configures a Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the client
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets a timeout for the HTTP client
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// NewClient creates a new PureMD API client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	client := &Client{
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey: apiKey,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client
}

// FetchWebContent retrieves the content of a given URL in markdown format
func (c *Client) FetchWebContent(ctx context.Context, urlStr string) (string, error) {
	if urlStr == "" {
		return "", fmt.Errorf("fetch web content: %w", ErrInvalidURL)
	}

	// Encode URL if needed
	encodedURL := url.QueryEscape(urlStr)
	endpoint := fmt.Sprintf("/%s", encodedURL)

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	var response string
	if err := c.do(req, &response); err != nil {
		return "", fmt.Errorf("fetching content for %s: %w", urlStr, err)
	}

	return response, nil
}

// ExtractModel represents the available AI models for extraction
type ExtractModel string

const (
	ModelLlama31     ExtractModel = "meta/llama-3.1-8b"
	ModelLlama33     ExtractModel = "meta/llama-3.3-70b"
	ModelHermes2Pro  ExtractModel = "mistral/hermes-2-pro-7b"
	ModelDeepseekR1  ExtractModel = "deepseek/r1-distill-qwen-32b"
)

// ExtractRequest represents a request to extract data from a webpage
type ExtractRequest struct {
	Prompt string          `json:"prompt"`
	Model  ExtractModel    `json:"model,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
}

// ExtractData fetches a URL and extracts data based on the prompt
func (c *Client) ExtractData(ctx context.Context, urlStr string, request *ExtractRequest) ([]byte, error) {
	if urlStr == "" {
		return nil, fmt.Errorf("extract data: %w", ErrInvalidURL)
	}
	if request.Prompt == "" {
		return nil, fmt.Errorf("extract data: prompt is required")
	}

	// Encode URL if needed
	encodedURL := url.QueryEscape(urlStr)
	endpoint := fmt.Sprintf("/%s", encodedURL)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	var response []byte
	if err := c.do(req, &response); err != nil {
		return nil, fmt.Errorf("extracting data from %s: %w", urlStr, err)
	}

	return response, nil
}

// SearchWeb searches the web for a given query and returns results in markdown
func (c *Client) SearchWeb(ctx context.Context, query string) (string, error) {
	if query == "" {
		return "", fmt.Errorf("search web: query is required")
	}

	// Encode query if needed
	encodedQuery := url.QueryEscape(query)
	endpoint := fmt.Sprintf("/search?q=%s", encodedQuery)

	req, err := c.newRequest(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	var response string
	if err := c.do(req, &response); err != nil {
		return "", fmt.Errorf("searching web for %s: %w", query, err)
	}

	return response, nil
}

// SearchExtractRequest represents a request to search and extract data
type SearchExtractRequest struct {
	Prompt string          `json:"prompt"`
	Model  ExtractModel    `json:"model,omitempty"`
	Schema json.RawMessage `json:"schema,omitempty"`
}

// SearchAndExtract searches the web and extracts data based on the prompt
func (c *Client) SearchAndExtract(ctx context.Context, query string, request *SearchExtractRequest) ([]byte, error) {
	if query == "" {
		return nil, fmt.Errorf("search and extract: query is required")
	}
	if request.Prompt == "" {
		return nil, fmt.Errorf("search and extract: prompt is required")
	}

	// Encode query if needed
	encodedQuery := url.QueryEscape(query)
	endpoint := fmt.Sprintf("/search?q=%s", encodedQuery)

	reqBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("encoding request: %w", err)
	}

	req, err := c.newRequest(ctx, http.MethodPost, endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	var response []byte
	if err := c.do(req, &response); err != nil {
		return nil, fmt.Errorf("searching and extracting for %s: %w", query, err)
	}

	return response, nil
}

// newRequest creates a new HTTP request with appropriate headers
func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}

	// Handle path
	if path[0] == '/' {
		path = path[1:]
	}
	
	u.Path = path

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("creating HTTP request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/json, text/plain")
	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}
	if c.apiKey != "" {
		req.Header.Set(apiKeyHeader, c.apiKey)
	}

	return req, nil
}

// do performs the HTTP request and processes the response
func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	switch resp.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidURL, readErrorBody(resp))
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusPaymentRequired:
		return ErrPaymentRequired
	case http.StatusUnsupportedMediaType:
		return ErrUnsupportedMedia
	case http.StatusTooManyRequests:
		return ErrRateLimitExceeded
	default:
		return fmt.Errorf("unexpected status code: %d - %s", resp.StatusCode, readErrorBody(resp))
	}

	// Read and parse the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	// Handle different response types
	switch v := v.(type) {
	case *string:
		*v = string(bodyBytes)
	case *[]byte:
		*v = bodyBytes
	default:
		if err := json.Unmarshal(bodyBytes, v); err != nil {
			return fmt.Errorf("unmarshaling response: %w", err)
		}
	}

	return nil
}

// readErrorBody reads the response body for error messages
func readErrorBody(resp *http.Response) string {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "unable to read error message"
	}
	return string(body)
}
package http_client

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

const defaultMaxResponseBodyBytes = 32 << 20 // 32 MB

// Config defines connection-pool and timeout parameters for the HTTP client.
// All fields have sensible defaults; zero values are filled in by normalized().
type Config struct {
	Timeout             time.Duration `yaml:"timeout"                  json:"timeout"`                   // request timeout, default 30s
	MaxIdleConns        int           `yaml:"max_idle_conns"           json:"max_idle_conns"`            // total max idle connections, default 100
	MaxIdleConnsPerHost int           `yaml:"max_idle_conns_per_host"  json:"max_idle_conns_per_host"`  // per-host max idle connections, default 10
	MaxConnsPerHost     int           `yaml:"max_conns_per_host"       json:"max_conns_per_host"`       // per-host max connections, 0 = unlimited
	IdleConnTimeout     time.Duration `yaml:"idle_conn_timeout"        json:"idle_conn_timeout"`        // idle connection reclaim timeout, default 90s
	TLSHandshakeTimeout time.Duration `yaml:"tls_handshake_timeout"    json:"tls_handshake_timeout"`   // TLS handshake timeout, default 10s
	MaxResponseBodyBytes int64        `yaml:"max_response_body_bytes"  json:"max_response_body_bytes"` // response body read limit, default 32 MB; 0 = use default
}

func (c Config) normalized() Config {
	if c.Timeout <= 0 {
		c.Timeout = 30 * time.Second
	}
	if c.MaxIdleConns <= 0 {
		c.MaxIdleConns = 100
	}
	if c.MaxIdleConnsPerHost <= 0 {
		c.MaxIdleConnsPerHost = 10
	}
	if c.IdleConnTimeout <= 0 {
		c.IdleConnTimeout = 90 * time.Second
	}
	if c.TLSHandshakeTimeout <= 0 {
		c.TLSHandshakeTimeout = 10 * time.Second
	}
	if c.MaxResponseBodyBytes <= 0 {
		c.MaxResponseBodyBytes = defaultMaxResponseBodyBytes
	}
	return c
}

// Client is an HTTP client with a shared connection pool.
type Client struct {
	httpClient           *http.Client
	maxResponseBodyBytes int64
}

// NewClient creates an HTTP client with a connection pool. Zero-value Config fields
// use built-in defaults.
func NewClient(cfg Config) *Client {
	cfg = cfg.normalized()

	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		maxResponseBodyBytes: cfg.MaxResponseBodyBytes,
	}
}

// HTTPClient returns the underlying *http.Client, useful for injecting into
// third-party modules that require a raw client (e.g. pkg/pay).
func (c *Client) HTTPClient() *http.Client {
	if c == nil {
		return nil
	}
	return c.httpClient
}

// Transport returns the underlying http.RoundTripper (connection pool).
// This is useful when a caller needs to share the connection pool but apply a
// different timeout policy — for example, a long-lived SSE or WebSocket
// connection that cannot use the global Timeout set on the Client.
//
//	base := http_client.GetHTTPClient()          // or NewClient(cfg)
//	streamCl := &http.Client{
//	    Transport: base.Transport(),             // shared pool
//	    Timeout:   0,                            // no deadline for streaming
//	}
//
// Returns nil if the receiver is nil.
func (c *Client) Transport() http.RoundTripper {
	if c == nil {
		return nil
	}
	return c.httpClient.Transport
}

// Get sends a GET request. ctx controls timeout and cancellation.
func (c *Client) Get(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req, headers)
	return c.do(req)
}

// Post sends a POST request. ctx controls timeout and cancellation.
func (c *Client) Post(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req, headers)
	return c.do(req)
}

// Put sends a PUT request. ctx controls timeout and cancellation.
func (c *Client) Put(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req, headers)
	return c.do(req)
}

// Patch sends a PATCH request. ctx controls timeout and cancellation.
func (c *Client) Patch(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req, headers)
	return c.do(req)
}

// Delete sends a DELETE request. ctx controls timeout and cancellation.
func (c *Client) Delete(ctx context.Context, url string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, 0, err
	}
	setHeaders(req, headers)
	return c.do(req)
}

// Do executes an arbitrary *http.Request, reading the response body up to the
// configured limit. The caller is responsible for setting ctx on the request.
func (c *Client) Do(req *http.Request) ([]byte, int, error) {
	return c.do(req)
}

func (c *Client) do(req *http.Request) ([]byte, int, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, c.maxResponseBodyBytes))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return respBody, resp.StatusCode, nil
}

func setHeaders(req *http.Request, headers map[string]string) {
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}

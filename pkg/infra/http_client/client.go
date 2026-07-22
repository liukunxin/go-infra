package http_client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/liukunxin/go-infra/pkg/base/trace"
)

// ErrNilClient is returned when a Client method is invoked on a nil receiver.
var ErrNilClient = errors.New("http_client: nil Client")

// ErrResponseBodyTooLarge is returned when the response body exceeds MaxResponseBodyBytes.
var ErrResponseBodyTooLarge = errors.New("http_client: response body too large")

const defaultMaxResponseBodyBytes = 32 << 20 // 32 MB

// Config defines connection-pool and timeout parameters for the HTTP client.
// All fields have sensible defaults; zero values are filled in by normalized().
type Config struct {
	Timeout              time.Duration    `yaml:"timeout"                 json:"timeout"`                   // request timeout, default 30s
	MaxIdleConns         int              `yaml:"max_idle_conns"          json:"max_idle_conns"`            // total max idle connections, default 100
	MaxIdleConnsPerHost  int              `yaml:"max_idle_conns_per_host" json:"max_idle_conns_per_host"`  // per-host max idle connections, default 10
	MaxConnsPerHost      int              `yaml:"max_conns_per_host"      json:"max_conns_per_host"`       // per-host max connections, 0 = unlimited
	IdleConnTimeout      time.Duration    `yaml:"idle_conn_timeout"       json:"idle_conn_timeout"`        // idle connection reclaim timeout, default 90s
	TLSHandshakeTimeout  time.Duration    `yaml:"tls_handshake_timeout"   json:"tls_handshake_timeout"`    // TLS handshake timeout, default 10s
	MaxResponseBodyBytes int64             `yaml:"max_response_body_bytes" json:"max_response_body_bytes"` // response body read limit, default 32 MB; 0 = use default
	DefaultHeaders       map[string]string `yaml:"default_headers"         json:"default_headers"`         // headers applied to every request; per-request options override
	MetricsEnabled       bool              `yaml:"metrics_enabled"         json:"metrics_enabled"`         // record outbound request metrics when true
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
	defaultHeaders       http.Header // immutable after construction; never mutated at request time
	metricsEnabled       bool
}

// NewClient creates an HTTP client with a connection pool. Zero-value Config fields
// use built-in defaults.
func NewClient(cfg Config) *Client {
	cfg = cfg.normalized()

	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     cfg.MaxConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		TLSHandshakeTimeout: cfg.TLSHandshakeTimeout,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	var defaultHeaders http.Header
	if len(cfg.DefaultHeaders) > 0 {
		defaultHeaders = make(http.Header, len(cfg.DefaultHeaders))
		for k, v := range cfg.DefaultHeaders {
			defaultHeaders.Set(k, v)
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		maxResponseBodyBytes: cfg.MaxResponseBodyBytes,
		defaultHeaders:       defaultHeaders,
		metricsEnabled:       cfg.MetricsEnabled,
	}
}

// HTTPClient returns the underlying *http.Client, useful for injecting into
// third-party modules that require a raw client (e.g. pkg/pay).
// Note: DefaultHeaders / X-Request-ID auto-bind / RequestOption / metrics are only applied by
// Client methods (Get/Post/.../Do), not by the raw *http.Client.
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

// CloseIdleConnections closes any idle connections in the underlying transport.
func (c *Client) CloseIdleConnections() {
	if c == nil || c.httpClient == nil {
		return
	}
	c.httpClient.CloseIdleConnections()
}

// Get sends a GET request. ctx controls timeout and cancellation.
func (c *Client) Get(ctx context.Context, url string, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, http.MethodGet, url, nil, opts...)
}

// Head sends a HEAD request. ctx controls timeout and cancellation.
func (c *Client) Head(ctx context.Context, url string, opts ...RequestOption) (int, error) {
	_, status, err := c.doMethod(ctx, http.MethodHead, url, nil, opts...)
	return status, err
}

// Post sends a POST request. ctx controls timeout and cancellation.
func (c *Client) Post(ctx context.Context, url string, body []byte, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, http.MethodPost, url, bodyReader(body), opts...)
}

// Put sends a PUT request. ctx controls timeout and cancellation.
func (c *Client) Put(ctx context.Context, url string, body []byte, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, http.MethodPut, url, bodyReader(body), opts...)
}

// Patch sends a PATCH request. ctx controls timeout and cancellation.
func (c *Client) Patch(ctx context.Context, url string, body []byte, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, http.MethodPatch, url, bodyReader(body), opts...)
}

// Delete sends a DELETE request. ctx controls timeout and cancellation.
func (c *Client) Delete(ctx context.Context, url string, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, http.MethodDelete, url, nil, opts...)
}

// DoRequest builds and executes a request with the given method, URL and optional body.
// Prefer this when you need a custom method or an io.Reader body without buffering into []byte.
func (c *Client) DoRequest(ctx context.Context, method, url string, body io.Reader, opts ...RequestOption) ([]byte, int, error) {
	return c.doMethod(ctx, method, url, body, opts...)
}

// Do executes an arbitrary *http.Request, applying client DefaultHeaders,
// RequestOption, and auto X-Request-ID (from ctx TraceID when unset), then reading
// the response body up to the configured limit.
// The caller is responsible for setting ctx on the request (via http.NewRequestWithContext).
func (c *Client) Do(req *http.Request, opts ...RequestOption) ([]byte, int, error) {
	if c == nil {
		return nil, 0, ErrNilClient
	}
	cfg := applyOptions(opts)
	c.applyRequestMeta(req, cfg)
	return c.do(req, cfg.metricPath)
}

func (c *Client) doMethod(ctx context.Context, method, url string, body io.Reader, opts ...RequestOption) ([]byte, int, error) {
	if c == nil {
		return nil, 0, ErrNilClient
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, 0, err
	}
	cfg := applyOptions(opts)
	c.applyRequestMeta(req, cfg)
	return c.do(req, cfg.metricPath)
}

// applyRequestMeta merges default headers and per-request options, then binds
// TraceID → X-Request-ID when available. Order: defaults < options < auto trace
// (auto trace only fills X-Request-ID when still empty).
func (c *Client) applyRequestMeta(req *http.Request, cfg requestConfig) {
	for k, vs := range c.defaultHeaders {
		if req.Header.Get(k) == "" {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	for k, vs := range cfg.header {
		for i, v := range vs {
			if i == 0 {
				req.Header.Set(k, v)
			} else {
				req.Header.Add(k, v)
			}
		}
	}
	// Prefer caller-defined X-Request-ID; otherwise fill from ctx TraceID when present.
	if req.Header.Get(HeaderRequestID) == "" {
		if id := trace.GetTraceID(req.Context()); id != "" {
			req.Header.Set(HeaderRequestID, id)
		}
	}
}

func (c *Client) do(req *http.Request, metricPath string) ([]byte, int, error) {
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.observe(req, metricPath, 0, start)
		return nil, 0, err
	}
	defer resp.Body.Close()

	// HEAD responses have no body — drain without retaining bytes.
	if req.Method == http.MethodHead {
		_, _ = io.Copy(io.Discard, resp.Body)
		c.observe(req, metricPath, resp.StatusCode, start)
		return nil, resp.StatusCode, nil
	}

	// Read one byte past the limit so we can detect truncation without a second pass.
	limit := c.maxResponseBodyBytes
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		c.observe(req, metricPath, resp.StatusCode, start)
		return nil, resp.StatusCode, err
	}
	if int64(len(respBody)) > limit {
		// Do not drain the remainder: a huge/malicious body would still be fully
		// downloaded. Closing without reading drops the connection from the pool,
		// which is the safer trade-off here.
		c.observe(req, metricPath, resp.StatusCode, start)
		return nil, resp.StatusCode, ErrResponseBodyTooLarge
	}
	c.observe(req, metricPath, resp.StatusCode, start)
	return respBody, resp.StatusCode, nil
}

func (c *Client) observe(req *http.Request, metricPath string, status int, start time.Time) {
	if !c.metricsEnabled {
		return
	}
	ctx := context.Background()
	if req != nil {
		ctx = req.Context()
	}
	recordClientMetrics(ctx, req, metricPath, status, time.Since(start))
}

func bodyReader(body []byte) io.Reader {
	if len(body) == 0 {
		return nil
	}
	return bytes.NewReader(body)
}

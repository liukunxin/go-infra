package http_client

import (
	"net/http"
)

// HeaderRequestID is the outbound header used to carry the request/trace identifier.
const HeaderRequestID = "X-Request-ID"

// RequestOption customizes a single request. Options are applied in order;
// later options override earlier ones for the same header key.
type RequestOption func(*requestConfig)

type requestConfig struct {
	header http.Header
}

// WithHeader sets a single request header.
func WithHeader(key, value string) RequestOption {
	return func(cfg *requestConfig) {
		cfg.ensureHeader().Set(key, value)
	}
}

// WithHeaders sets multiple request headers. Nil or empty maps are ignored.
func WithHeaders(headers map[string]string) RequestOption {
	return func(cfg *requestConfig) {
		if len(headers) == 0 {
			return
		}
		h := cfg.ensureHeader()
		for k, v := range headers {
			h.Set(k, v)
		}
	}
}

// WithContentType sets the Content-Type header.
func WithContentType(contentType string) RequestOption {
	return WithHeader("Content-Type", contentType)
}

// WithJSON sets Content-Type to application/json.
func WithJSON() RequestOption {
	return WithContentType("application/json")
}

func (cfg *requestConfig) ensureHeader() http.Header {
	if cfg.header == nil {
		cfg.header = make(http.Header, 4)
	}
	return cfg.header
}

func applyOptions(opts []RequestOption) requestConfig {
	var cfg requestConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

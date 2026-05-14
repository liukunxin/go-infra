package llm

import (
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAICompatibleClientConfig is a one-shot config for quickest setup.
type OpenAICompatibleClientConfig struct {
	Provider    string
	Model       string
	BaseURL     string
	APIKey      string
	APIKeyEnv   string
	HTTPClient  *http.Client
	HTTPTimeout time.Duration
	Headers     map[string]string
	Retry       RetryConfig
}

// NewOpenAICompatibleClient creates a ready-to-use Client with one provider.
// This is the shortest setup path for most projects.
func NewOpenAICompatibleClient(cfg OpenAICompatibleClientConfig) (*Client, error) {
	opts, err := buildSimpleOptions(cfg)
	if err != nil {
		return nil, err
	}
	return New(opts...)
}

// InitOpenAICompatibleClient initializes global singleton client via shortest path.
func InitOpenAICompatibleClient(cfg OpenAICompatibleClientConfig) error {
	opts, err := buildSimpleOptions(cfg)
	if err != nil {
		return err
	}
	return Init(opts...)
}

func buildSimpleOptions(cfg OpenAICompatibleClientConfig) ([]Option, error) {
	providerName := strings.TrimSpace(cfg.Provider)
	if providerName == "" {
		providerName = "default"
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" && strings.TrimSpace(cfg.APIKeyEnv) != "" {
		apiKey = os.Getenv(strings.TrimSpace(cfg.APIKeyEnv))
	}
	provider, err := NewOpenAICompatibleProvider(providerName, OpenAICompatibleConfig{
		BaseURL:      cfg.BaseURL,
		APIKey:       apiKey,
		HTTPClient:   cfg.HTTPClient,
		HTTPTimeout:  cfg.HTTPTimeout,
		Headers:      cfg.Headers,
		DefaultModel: cfg.Model,
		Retry:        cfg.Retry,
	})
	if err != nil {
		return nil, err
	}
	return []Option{
		WithProvider(providerName, provider),
		WithDefaultProvider(providerName),
		WithDefaultModel(cfg.Model),
	}, nil
}

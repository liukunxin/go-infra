package llm

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	ProviderTypeOpenAICompatible = "openai_compatible"
)

// Config defines YAML-friendly llm settings for app configuration files.
type Config struct {
	DefaultProvider string                    `yaml:"default_provider" json:"default_provider"`
	DefaultModel    string                    `yaml:"default_model" json:"default_model"`
	Providers       map[string]ProviderConfig `yaml:"providers" json:"providers"`
	Fallbacks       []FallbackConfig          `yaml:"fallbacks" json:"fallbacks"`
}

// ProviderConfig defines one provider entry under config.providers.
type ProviderConfig struct {
	Type         string            `yaml:"type" json:"type"`
	BaseURL      string            `yaml:"base_url" json:"base_url"`
	APIKey       string            `yaml:"api_key" json:"api_key"`
	APIKeyEnv    string            `yaml:"api_key_env" json:"api_key_env"`
	DefaultModel string            `yaml:"default_model" json:"default_model"`
	HTTPTimeout  time.Duration     `yaml:"http_timeout" json:"http_timeout"`
	Retry        RetryConfig       `yaml:"retry" json:"retry"`
	Headers      map[string]string `yaml:"headers" json:"headers"`

	// AI Gateway 专属字段
	GatewayProvider string `yaml:"gateway_provider" json:"gateway_provider"`
	GatewayVersion  string `yaml:"gateway_version" json:"gateway_version"`
	Path            string `yaml:"path" json:"path"`
	ProductName     string `yaml:"product_name" json:"product_name"`
	IntentionCode   string `yaml:"intention_code" json:"intention_code"`
	DefaultUID      string `yaml:"default_uid" json:"default_uid"`
}

// FallbackConfig defines primary route and backup targets.
type FallbackConfig struct {
	PrimaryProvider string           `yaml:"primary_provider" json:"primary_provider"`
	PrimaryModel    string           `yaml:"primary_model" json:"primary_model"`
	Targets         []FallbackTarget `yaml:"targets" json:"targets"`
}

// NewFromConfig creates an LLM client from Config, then applies extra opts.
func NewFromConfig(cfg Config, opts ...Option) (*Client, error) {
	built, err := buildOptionsFromConfig(cfg)
	if err != nil {
		return nil, err
	}
	built = append(built, opts...)
	return New(built...)
}

// InitFromConfig initializes global LLM client from Config.
func InitFromConfig(cfg Config, opts ...Option) error {
	built, err := buildOptionsFromConfig(cfg)
	if err != nil {
		return err
	}
	built = append(built, opts...)
	return Init(built...)
}

func buildOptionsFromConfig(cfg Config) ([]Option, error) {
	opts := make([]Option, 0, len(cfg.Providers)+2+len(cfg.Fallbacks))
	for name, providerCfg := range cfg.Providers {
		provider, err := buildProvider(name, providerCfg)
		if err != nil {
			return nil, err
		}
		opts = append(opts, WithProvider(name, provider))
	}
	if cfg.DefaultProvider != "" {
		opts = append(opts, WithDefaultProvider(cfg.DefaultProvider))
	}
	if cfg.DefaultModel != "" {
		opts = append(opts, WithDefaultModel(cfg.DefaultModel))
	}
	for _, fb := range cfg.Fallbacks {
		if strings.TrimSpace(fb.PrimaryModel) == "" {
			opts = append(opts, WithFallbackAnyModel(fb.PrimaryProvider, fb.Targets...))
			continue
		}
		opts = append(opts, WithFallback(fb.PrimaryProvider, fb.PrimaryModel, fb.Targets...))
	}
	return opts, nil
}

func buildProvider(name string, cfg ProviderConfig) (Provider, error) {
	providerType := strings.TrimSpace(cfg.Type)
	if providerType == "" {
		providerType = ProviderTypeOpenAICompatible
	}
	apiKey := strings.TrimSpace(cfg.APIKey)
	if apiKey == "" && strings.TrimSpace(cfg.APIKeyEnv) != "" {
		apiKey = os.Getenv(strings.TrimSpace(cfg.APIKeyEnv))
	}

	switch providerType {
	case ProviderTypeOpenAICompatible:
		return NewOpenAICompatibleProvider(name, OpenAICompatibleConfig{
			BaseURL:      cfg.BaseURL,
			APIKey:       apiKey,
			HTTPTimeout:  cfg.HTTPTimeout,
			Retry:        cfg.Retry,
			Headers:      cfg.Headers,
			DefaultModel: cfg.DefaultModel,
		})
	case ProviderTypeAIGateway:
		return NewAIGatewayProvider(name, AIGatewayConfig{
			BaseURL:         cfg.BaseURL,
			APIKey:          apiKey,
			HTTPTimeout:     cfg.HTTPTimeout,
			Retry:           cfg.Retry,
			Headers:         cfg.Headers,
			DefaultModel:    cfg.DefaultModel,
			GatewayProvider: cfg.GatewayProvider,
			GatewayVersion:  cfg.GatewayVersion,
			Path:            cfg.Path,
			ProductName:     cfg.ProductName,
			IntentionCode:   cfg.IntentionCode,
			DefaultUID:      cfg.DefaultUID,
		})
	default:
		return nil, fmt.Errorf("%w: unsupported provider type %q", ErrInvalidConfig, providerType)
	}
}

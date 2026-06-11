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
// APIKey 支持两种写法：直接填值 "sk-xxx" 或环境变量引用 "${ENV_NAME}"。
type ProviderConfig struct {
	Type         string            `yaml:"type" json:"type"`
	BaseURL      string            `yaml:"base_url" json:"base_url"`
	APIKey       string            `yaml:"api_key" json:"api_key"`
	DefaultModel string            `yaml:"default_model" json:"default_model"`
	HTTPTimeout  time.Duration     `yaml:"http_timeout" json:"http_timeout"`
	Retry        RetryConfig       `yaml:"retry" json:"retry"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
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
	apiKey := resolveEnvValue(cfg.APIKey)

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
	default:
		return nil, fmt.Errorf("%w: unsupported provider type %q", ErrInvalidConfig, providerType)
	}
}

// resolveEnvValue 解析配置值：支持 "${ENV_NAME}" 语法从环境变量读取，否则直接返回原值。
func resolveEnvValue(val string) string {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "${") && strings.HasSuffix(val, "}") {
		return os.Getenv(val[2 : len(val)-1])
	}
	return val
}

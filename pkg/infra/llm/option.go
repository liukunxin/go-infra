package llm

import "github.com/liukunxin/go-infra/internal/option"

type optionConfig struct {
	defaultProvider string
	defaultModel    string
	providers       map[string]Provider
	fallbacks       map[string][]FallbackTarget
}

func defaultOptionConfig() *optionConfig {
	return &optionConfig{
		providers: make(map[string]Provider),
		fallbacks: make(map[string][]FallbackTarget),
	}
}

// Option configures llm.Client.
type Option = option.Option[optionConfig]

// WithProvider registers one named provider.
func WithProvider(name string, provider Provider) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		if name == "" || provider == nil {
			return ErrInvalidConfig
		}
		c.providers[name] = provider
		return nil
	})
}

// WithDefaultProvider sets the fallback provider name for requests.
func WithDefaultProvider(name string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.defaultProvider = name
		return nil
	})
}

// WithDefaultModel sets the fallback model for requests.
func WithDefaultModel(model string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.defaultModel = model
		return nil
	})
}

// WithFallback registers fallback route for a primary provider/model.
// If primaryModel is empty, fallback applies to all models of primaryProvider.
func WithFallback(primaryProvider, primaryModel string, targets ...FallbackTarget) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		if primaryProvider == "" {
			return ErrInvalidConfig
		}
		validTargets := make([]FallbackTarget, 0, len(targets))
		for _, t := range targets {
			if t.Provider == "" || t.Model == "" {
				return ErrInvalidConfig
			}
			validTargets = append(validTargets, t)
		}
		if len(validTargets) == 0 {
			return nil
		}
		key := fallbackKey(primaryProvider, primaryModel)
		c.fallbacks[key] = append(c.fallbacks[key], validTargets...)
		return nil
	})
}

// WithFallbackAnyModel registers fallback route for all models of primaryProvider.
func WithFallbackAnyModel(primaryProvider string, targets ...FallbackTarget) Option {
	return WithFallback(primaryProvider, "", targets...)
}

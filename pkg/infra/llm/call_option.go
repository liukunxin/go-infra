package llm

type callOptionConfig struct {
	provider string
	model    string
}

// CallOption configures one Generate/GenerateStream call.
type CallOption interface {
	Apply(*callOptionConfig) error
}

type callOptionFunc func(*callOptionConfig) error

func (f callOptionFunc) Apply(c *callOptionConfig) error {
	return f(c)
}

// WithCallProvider overrides provider for one call.
func WithCallProvider(provider string) CallOption {
	return callOptionFunc(func(c *callOptionConfig) error {
		c.provider = provider
		return nil
	})
}

// WithCallModel overrides model for one call.
func WithCallModel(model string) CallOption {
	return callOptionFunc(func(c *callOptionConfig) error {
		c.model = model
		return nil
	})
}

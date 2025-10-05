package traffic

type optionConfig struct {
	controller Controller
}

type Option interface {
	apply(*optionConfig) error
}

type optionFunc func(*optionConfig) error

func (f optionFunc) apply(c *optionConfig) error {
	return f(c)
}

func WithController(controller Controller) Option {
	return optionFunc(func(c *optionConfig) error {
		c.controller = controller
		return nil
	})
}

type TryPassOptionConfig struct {
	TrafficType TrafficType
}

type TryPassOption interface {
	Apply(*TryPassOptionConfig) error
}

type tryPassOptionFunc func(*TryPassOptionConfig) error

func (f tryPassOptionFunc) Apply(c *TryPassOptionConfig) error {
	return f(c)
}

type TryPassOptions struct {
	opts []TryPassOption
}

func NewTryPassOptions() *TryPassOptions {
	return &TryPassOptions{}
}

func (o *TryPassOptions) Apply(oc *TryPassOptionConfig) error {
	for _, opt := range o.opts {
		if err := opt.Apply(oc); err != nil {
			return err
		}
	}

	return nil
}

func (o *TryPassOptions) with(opts ...TryPassOption) *TryPassOptions {
	o.opts = append(o.opts, opts...)
	return o
}

func (o *TryPassOptions) WithTrafficType(trafficType TrafficType) *TryPassOptions {
	return o.with(tryPassOptionFunc(func(c *TryPassOptionConfig) error {
		c.TrafficType = trafficType
		return nil
	}))
}

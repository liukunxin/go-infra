package traffic

import "github.com/liukunxin/go-infra/internal/option"

type optionConfig struct {
	controller Controller
}

// Option 是 traffic 模块 Init 函数的函数式选项类型。
type Option = option.Option[optionConfig]

func WithController(controller Controller) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
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

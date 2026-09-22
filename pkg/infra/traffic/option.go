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

// Package option 提供泛型函数式选项（Functional Options）基础类型，
// 供 SDK 内各包的 Init/New 函数统一使用，避免每个包重复定义相同的 Option 接口和 optionFunc 类型。
//
// 使用方式：
//
//	// 1. 在目标包定义 optionConfig（私有）
//	type optionConfig struct { timeout time.Duration }
//
//	// 2. 用类型别名暴露 Option，无需本包出现在公开 API 中
//	type Option = option.Option[optionConfig]
//
//	// 3. 构造函数直接返回 option.Func
//	func WithTimeout(d time.Duration) Option {
//	    return option.Func[optionConfig](func(c *optionConfig) error {
//	        c.timeout = d
//	        return nil
//	    })
//	}
//
//	// 4. Apply 选项
//	func Init(opts ...Option) error {
//	    c := &optionConfig{}
//	    for _, opt := range opts {
//	        if err := opt.Apply(c); err != nil { return err }
//	    }
//	    ...
//	}
package option

// Option 是泛型函数式选项接口，用于配置类型 C 的实例。
type Option[C any] interface {
	Apply(*C) error
}

// Func 是函数类型，实现了 Option[C] 接口。
// 可直接将 func(*C) error 类型转换为 Func[C] 使用。
type Func[C any] func(*C) error

func (f Func[C]) Apply(c *C) error { return f(c) }

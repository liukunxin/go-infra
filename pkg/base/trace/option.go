package trace

import (
	"fmt"

	"github.com/liukunxin/go-infra/internal/option"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type optionConfig struct {
	serviceName  *string
	sampleRatio  *float64
	spanExporter tracesdk.SpanExporter
}

// Option 是 trace 模块的函数式选项类型。
type Option = option.Option[optionConfig]

func WithServiceName(serviceName string) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.serviceName = &serviceName
		return nil
	})
}

// WithSampleRatio sets the trace sampling ratio. Must be in [0.0, 1.0].
// 0.0 means no traces are sampled; 1.0 means all traces are sampled.
func WithSampleRatio(sampleRatio float64) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		if sampleRatio < 0.0 || sampleRatio > 1.0 {
			return fmt.Errorf("trace: sample ratio must be between 0.0 and 1.0, got %g", sampleRatio)
		}
		c.sampleRatio = &sampleRatio
		return nil
	})
}

func WithSpanExporter(spanExporter tracesdk.SpanExporter) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		c.spanExporter = spanExporter
		return nil
	})
}

func WithConfig(cfg *Config) Option {
	return option.Func[optionConfig](func(c *optionConfig) error {
		if cfg == nil {
			return nil
		}
		if cfg.ServiceName != nil {
			c.serviceName = cfg.ServiceName
		}
		if cfg.SampleRatio != nil {
			c.sampleRatio = cfg.SampleRatio
		}
		return nil
	})
}

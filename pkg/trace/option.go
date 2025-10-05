package trace

import (
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
)

type optionConfig struct {
	serviceName  *string
	sampleRatio  *float64
	spanExporter tracesdk.SpanExporter
}

type Option interface {
	apply(*optionConfig) error
}

type optionFunc func(*optionConfig) error

func (f optionFunc) apply(c *optionConfig) error {
	return f(c)
}

func WithServiceName(serviceName string) Option {
	return optionFunc(func(c *optionConfig) error {
		c.serviceName = &serviceName
		return nil
	})
}

func WithSampleRatio(sampleRatio float64) Option {
	return optionFunc(func(c *optionConfig) error {
		c.sampleRatio = &sampleRatio
		return nil
	})
}
func WithSpanExporter(spanExporter tracesdk.SpanExporter) Option {
	return optionFunc(func(c *optionConfig) error {
		c.spanExporter = spanExporter
		return nil
	})
}

func WithConfig(cfg *Config) Option {
	return optionFunc(func(c *optionConfig) error {
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

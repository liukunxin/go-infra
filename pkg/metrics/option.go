package metrics

import (
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type optionConfig struct {
	reader     sdkmetric.Reader
	attributes []attribute.KeyValue
}

type Option interface {
	apply(*optionConfig) error
}

type optionFunc func(*optionConfig) error

func (f optionFunc) apply(c *optionConfig) error {
	return f(c)
}

func WithReader(reader sdkmetric.Reader) Option {
	return optionFunc(func(oc *optionConfig) error {
		oc.reader = reader
		return nil
	})
}

func WithAttributes(attributes ...attribute.KeyValue) Option {
	return optionFunc(func(oc *optionConfig) error {
		oc.attributes = attributes
		return nil
	})
}

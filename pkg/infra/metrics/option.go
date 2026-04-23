package metrics

import (
	"github.com/liukunxin/go-infra/internal/option"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type optionConfig struct {
	reader     sdkmetric.Reader
	attributes []attribute.KeyValue
}

// Option 是 metrics 模块的函数式选项类型。
type Option = option.Option[optionConfig]

func WithReader(reader sdkmetric.Reader) Option {
	return option.Func[optionConfig](func(oc *optionConfig) error {
		oc.reader = reader
		return nil
	})
}

func WithAttributes(attributes ...attribute.KeyValue) Option {
	return option.Func[optionConfig](func(oc *optionConfig) error {
		oc.attributes = attributes
		return nil
	})
}

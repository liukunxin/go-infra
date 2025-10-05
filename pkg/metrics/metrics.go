package metrics

import (
	"context"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func Must(ctx context.Context, opts ...Option) {
	err := Init(ctx, opts...)
	if err != nil {
		panic(err)
	}
}

func Init(ctx context.Context, opts ...Option) error {
	c := &optionConfig{}

	for _, opt := range opts {
		if err := opt.apply(c); err != nil {
			return err
		}
	}

	metricOpts := make([]sdkmetric.Option, 0, 4)

	if c.reader != nil {
		metricOpts = append(metricOpts, sdkmetric.WithReader(c.reader))
	}

	metricOpts = append(metricOpts,
		sdkmetric.WithResource(resource.NewWithAttributes(semconv.SchemaURL, c.attributes...)),
	)

	meterProvider := sdkmetric.NewMeterProvider(metricOpts...)

	otel.SetMeterProvider(meterProvider)

	return nil
}

package trace

import (
	"context"
	"fmt"
	"github.com/liukunxin/go-infra/pkg/env"
	"sync/atomic"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var (
	globalTracerProvider atomic.Pointer[trace.TracerProvider]
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

	var serviceName string
	if c.serviceName != nil {
		serviceName = *c.serviceName
	}
	if serviceName == "" {
		serviceName = env.GetName()
	}
	if serviceName == "" {
		return fmt.Errorf("service name is required")
	}

	tpOpts := make([]trace.TracerProviderOption, 0, 8)

	if c.sampleRatio != nil {
		tpOpts = append(tpOpts, trace.WithSampler(
			trace.ParentBased(trace.TraceIDRatioBased(*c.sampleRatio)),
		))
	}

	if c.spanExporter != nil {
		spanExporter := c.spanExporter
		tpOpts = append(tpOpts, trace.WithBatcher(spanExporter))
	}

	attrs := make([]attribute.KeyValue, 0)
	attrs = append(attrs, semconv.ServiceNameKey.String(serviceName))

	tpOpts = append(tpOpts,
		trace.WithResource(resource.NewWithAttributes(semconv.SchemaURL, attrs...)),
	)

	tracerProvider := trace.NewTracerProvider(tpOpts...)

	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(b3.New()),
	)

	globalTracerProvider.Store(tracerProvider)

	return nil
}

func Flush() error {
	tracerProvider := globalTracerProvider.Load()
	if tracerProvider != nil {
		return tracerProvider.ForceFlush(context.Background())
	}

	return nil
}

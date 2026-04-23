package trace

import (
	"context"
	"fmt"
	"github.com/liukunxin/go-infra/pkg/base/env"
	"log"
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

func Init(opts ...Option) {
	tp, err := newTracerProvider(opts...)
	if err != nil {
		log.Fatal(err.Error())
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(b3.New()),
	)

	globalTracerProvider.Store(tp)
}

func newTracerProvider(opts ...Option) (*trace.TracerProvider, error) {
	c := &optionConfig{}

	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
			return nil, err
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
		return nil, fmt.Errorf("service name is required")
	}

	tpOpts := make([]trace.TracerProviderOption, 0, 8)

	// 配置采样器（默认使用ParentBased，确保trace链路完整）
	if c.sampleRatio != nil {
		tpOpts = append(tpOpts, trace.WithSampler(
			trace.ParentBased(trace.TraceIDRatioBased(*c.sampleRatio)),
		))
	} else {
		// 默认使用 ParentBased(AlwaysSample)，确保每个请求都有trace
		tpOpts = append(tpOpts, trace.WithSampler(
			trace.ParentBased(trace.AlwaysSample()),
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
	return tracerProvider, nil
}

func Flush() {
	tracerProvider := globalTracerProvider.Load()
	if tracerProvider == nil {
		return
	}
	if err := tracerProvider.ForceFlush(context.Background()); err != nil {
		log.Fatal(err.Error())
	}
}

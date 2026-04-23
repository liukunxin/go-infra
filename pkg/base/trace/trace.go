package trace

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"

	"github.com/liukunxin/go-infra/pkg/base/env"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

var globalTracerProvider atomic.Pointer[trace.TracerProvider]

// Init initializes the global TracerProvider and text-map propagator.
// Returns an error if the configuration is invalid (e.g. missing service name).
func Init(opts ...Option) error {
	tp, err := newTracerProvider(opts...)
	if err != nil {
		return err
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(b3.New()),
	)
	globalTracerProvider.Store(tp)
	return nil
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

// Shutdown flushes all pending spans, closes the exporter connection, and stops
// the TracerProvider. Call this on program exit with a deadline context:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	_ = trace.Shutdown(ctx)
func Shutdown(ctx context.Context) error {
	tp := globalTracerProvider.Load()
	if tp == nil {
		return nil
	}
	return tp.Shutdown(ctx)
}

// Flush forces an immediate export of all buffered spans without closing the provider.
// Prefer Shutdown for graceful termination; use Flush only when you need to ensure
// spans are exported while the provider remains active.
func Flush() {
	tp := globalTracerProvider.Load()
	if tp == nil {
		return
	}
	if err := tp.ForceFlush(context.Background()); err != nil {
		log.Printf("trace: force flush error: %v", err)
	}
}

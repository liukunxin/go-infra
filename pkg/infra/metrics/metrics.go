package metrics

import (
	"context"
	"log"
	"os"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"github.com/liukunxin/go-infra/internal/consts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// globalMeterProvider holds the concrete SDK provider so Shutdown can be called on it.
// The otel.MeterProvider interface does not expose Shutdown.
var globalMeterProvider atomic.Pointer[sdkmetric.MeterProvider]

// Init initializes the OTel MeterProvider with a Prometheus exporter.
// appName is used as the "service.name" resource attribute; falls back to the
// APP_NAME environment variable when empty.
//
// Call RegisterGinRoutes separately to expose /metrics and attach the middleware.
func Init(appName string, opts ...Option) error {
	exporter, err := prometheus.New()
	if err != nil {
		return err
	}
	if appName == "" {
		appName = os.Getenv(consts.AppName)
	}

	// Built-in attributes come first; user-supplied WithAttributes calls append to them.
	allOpts := append([]Option{
		WithReader(exporter),
		WithAttributes(attribute.String(string(semconv.ServiceNameKey), appName)),
	}, opts...)

	return initProvider(allOpts...)
}

// RegisterGinRoutes registers the /metrics scrape endpoint and attaches the
// per-request metrics middleware to router. Must be called after Init.
func RegisterGinRoutes(router *gin.Engine) {
	router.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})
	router.Use(ginMetricsMiddleware())
}

// InitMetrics is a convenience wrapper that calls Init and RegisterGinRoutes in one step.
// It calls log.Fatal on any initialization error to preserve the original fail-fast behaviour.
func InitMetrics(appName string, router *gin.Engine, opts ...Option) {
	if err := Init(appName, opts...); err != nil {
		log.Fatalf("metrics: init provider: %v", err)
	}
	if err := initHTTPMetrics(); err != nil {
		log.Fatalf("metrics: init http metrics: %v", err)
	}
	RegisterGinRoutes(router)
}

// Shutdown flushes pending metrics and releases provider resources.
// Call on program exit:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//	_ = metrics.Shutdown(ctx)
func Shutdown(ctx context.Context) error {
	mp := globalMeterProvider.Load()
	if mp == nil {
		return nil
	}
	return mp.Shutdown(ctx)
}

func initProvider(opts ...Option) error {
	c := &optionConfig{}

	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
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

	mp := sdkmetric.NewMeterProvider(metricOpts...)
	otel.SetMeterProvider(mp)
	globalMeterProvider.Store(mp)

	return nil
}

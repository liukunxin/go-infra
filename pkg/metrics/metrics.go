package metrics

import (
	"github.com/gin-gonic/gin"
	"github.com/liukunxin/go-infra/internal/consts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"log"
	"os"
)

// InitMetrics 对外暴露的初始化入口方法
func InitMetrics(appName string, router *gin.Engine, opts ...Option) {
	// 1️⃣ 创建 Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err.Error())
	}
	if appName == "" {
		appName = os.Getenv(consts.AppName)
	}
	// 2️⃣ 初始化 metrics Provider
	registerOpts := append([]Option{
		WithReader(exporter),
		WithAttributes(
			attribute.String("service.name", appName),
		),
	}, opts...) // 将可选参数追加进来

	if err = initProvider(registerOpts...); err != nil {
		log.Fatal(err.Error())
	}

	// 3️⃣ 暴露 /metrics 路由
	router.GET("/metrics", func(c *gin.Context) {
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	})

	// 4️⃣ 初始化默认 HTTP 指标
	initHTTPMetrics()

	// 5️⃣ 注册 Gin 中间件收集指标
	router.Use(ginMetricsMiddleware())
}

// OpenTelemetry Metrics 初始化核心
// 内部初始化全局 MeterProvider
func initProvider(opts ...Option) error {
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

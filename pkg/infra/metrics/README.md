# Metrics - Prometheus 监控指标

基于 OpenTelemetry 和 Prometheus 的监控指标采集，自动暴露 `/metrics` 端点。

## 📖 功能特性

- ✅ 基于 OpenTelemetry Metrics SDK
- ✅ Prometheus 格式导出
- ✅ 自动 HTTP 指标采集
- ✅ 自动注册 Gin 中间件
- ✅ 支持自定义指标
- ✅ 开箱即用

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/metrics"
)

func main() {
    router := gin.Default()

    // 初始化 Metrics（自动注册 /metrics 路由和中间件）
    metrics.InitMetrics("my-service", router)

    // 业务路由
    router.GET("/api/user", func(c *gin.Context) {
        c.JSON(200, gin.H{"message": "ok"})
    })

    router.Run(":8080")
}
```

访问 `http://localhost:8080/metrics` 即可看到指标数据。

## 📋 默认指标

### HTTP 请求指标

#### 1. http_requests_total

请求总数（Counter）

```
# 标签
method: GET, POST, PUT, DELETE, etc.
path: /api/user, /api/order, etc.
status: 200, 404, 500, etc.

# 示例
http_requests_total{method="GET",path="/api/user",status="200"} 1234
http_requests_total{method="POST",path="/api/order",status="500"} 5
```

#### 2. http_request_duration_ms

请求延迟（Histogram）

```
# 标签
method: GET, POST, etc.
path: /api/user, /api/order, etc.
status: 200, 404, 500, etc.

# 示例
http_request_duration_ms_bucket{method="GET",path="/api/user",status="200",le="50"} 800
http_request_duration_ms_bucket{method="GET",path="/api/user",status="200",le="100"} 950
http_request_duration_ms_bucket{method="GET",path="/api/user",status="200",le="+Inf"} 1000
```

## 💡 使用示例

### 示例1：完整的监控设置

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/metrics"
    "github.com/liukunxin/go-infra/pkg/trace"
    "github.com/liukunxin/go-infra/pkg/log"
    "github.com/liukunxin/go-infra/pkg/middlewares"
)

func main() {
    // 初始化日志和追踪
    log.Init(log.Config{Level: "info"})
    trace.Init(trace.WithServiceName("my-service"))

    router := gin.Default()

    // 初始化 Metrics（必须在业务路由之前）
    metrics.InitMetrics("my-service", router)

    // 注册中间件
    router.Use(middlewares.GinTraceMiddleware())
    router.Use(middlewares.HttpLogRecord())

    // 业务路由
    router.GET("/api/user/:id", getUserHandler)
    router.POST("/api/order", createOrderHandler)

    router.Run(":8080")
}
```

### 示例2：自定义 Counter

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/metric"
)

var (
    orderCounter metric.Int64Counter
)

func init() {
    meter := otel.Meter("business")
    
    // 创建自定义 Counter
    var err error
    orderCounter, err = meter.Int64Counter(
        "order_created_total",
        metric.WithDescription("订单创建总数"),
    )
    if err != nil {
        log.Fatal(err)
    }
}

func createOrder(ctx context.Context, order *Order) error {
    // 业务逻辑...
    
    // 增加计数
    orderCounter.Add(ctx, 1, 
        metric.WithAttributes(
            attribute.String("payment_type", order.PaymentType),
            attribute.String("status", "success"),
        ),
    )
    
    return nil
}
```

### 示例3：自定义 Histogram

```go
var (
    paymentDuration metric.Float64Histogram
)

func init() {
    meter := otel.Meter("business")
    
    // 创建自定义 Histogram
    var err error
    paymentDuration, err = meter.Float64Histogram(
        "payment_duration_seconds",
        metric.WithDescription("支付处理时长"),
    )
    if err != nil {
        log.Fatal(err)
    }
}

func processPayment(ctx context.Context, order *Order) error {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        paymentDuration.Record(ctx, duration,
            metric.WithAttributes(
                attribute.String("payment_type", order.PaymentType),
            ),
        )
    }()
    
    // 支付逻辑...
    return nil
}
```

### 示例4：自定义 Gauge

```go
var (
    activeConnections metric.Int64UpDownCounter
)

func init() {
    meter := otel.Meter("system")
    
    // 创建 Gauge（使用 UpDownCounter）
    var err error
    activeConnections, err = meter.Int64UpDownCounter(
        "active_connections",
        metric.WithDescription("当前活跃连接数"),
    )
    if err != nil {
        log.Fatal(err)
    }
}

func handleConnection(conn net.Conn) {
    ctx := context.Background()
    
    // 连接建立
    activeConnections.Add(ctx, 1)
    defer activeConnections.Add(ctx, -1)  // 连接关闭
    
    // 处理连接...
}
```

### 示例5：业务指标监控

```go
type BusinessMetrics struct {
    orderTotal       metric.Int64Counter
    orderAmount      metric.Float64Histogram
    userActive       metric.Int64UpDownCounter
    cacheHitRate     metric.Float64Gauge
}

func NewBusinessMetrics() *BusinessMetrics {
    meter := otel.Meter("business")
    
    orderTotal, _ := meter.Int64Counter("business_order_total")
    orderAmount, _ := meter.Float64Histogram("business_order_amount")
    userActive, _ := meter.Int64UpDownCounter("business_user_active")
    
    return &BusinessMetrics{
        orderTotal:   orderTotal,
        orderAmount:  orderAmount,
        userActive:   userActive,
    }
}

// 使用示例
func (m *BusinessMetrics) RecordOrder(ctx context.Context, order *Order) {
    // 订单数量
    m.orderTotal.Add(ctx, 1,
        metric.WithAttributes(
            attribute.String("status", order.Status),
            attribute.String("channel", order.Channel),
        ),
    )
    
    // 订单金额
    m.orderAmount.Record(ctx, order.Amount,
        metric.WithAttributes(
            attribute.String("channel", order.Channel),
        ),
    )
}
```

## 🔧 Prometheus 配置

### prometheus.yml

```yaml
global:
  scrape_interval: 15s  # 每15秒抓取一次

scrape_configs:
  - job_name: 'my-service'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
```

### Grafana Dashboard 查询

```promql
# QPS（每秒请求数）
rate(http_requests_total[1m])

# 平均响应时间
rate(http_request_duration_ms_sum[1m]) / rate(http_request_duration_ms_count[1m])

# P95 响应时间
histogram_quantile(0.95, rate(http_request_duration_ms_bucket[1m]))

# P99 响应时间
histogram_quantile(0.99, rate(http_request_duration_ms_bucket[1m]))

# 错误率
rate(http_requests_total{status=~"5.."}[1m]) / rate(http_requests_total[1m])

# 按路径分组的 QPS
sum by (path) (rate(http_requests_total[1m]))
```

## 📊 常用监控场景

### 1. 服务健康监控

```promql
# 服务是否存活
up{job="my-service"}

# 总 QPS
sum(rate(http_requests_total[1m]))

# 错误率
sum(rate(http_requests_total{status=~"5.."}[1m])) / sum(rate(http_requests_total[1m]))
```

### 2. 性能监控

```promql
# P50 响应时间
histogram_quantile(0.50, rate(http_request_duration_ms_bucket[1m]))

# P95 响应时间
histogram_quantile(0.95, rate(http_request_duration_ms_bucket[1m]))

# P99 响应时间
histogram_quantile(0.99, rate(http_request_duration_ms_bucket[1m]))
```

### 3. 业务监控

```promql
# 订单创建速率
rate(order_created_total[1m])

# 支付成功率
sum(rate(order_created_total{status="paid"}[1m])) / sum(rate(order_created_total[1m]))

# 平均订单金额
rate(business_order_amount_sum[1m]) / rate(business_order_amount_count[1m])
```

### 4. 告警规则

```yaml
groups:
  - name: service_alerts
    rules:
      # 错误率告警
      - alert: HighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) > 0.05
        for: 5m
        annotations:
          summary: "错误率超过5%"
          description: "服务 {{ $labels.job }} 错误率为 {{ $value }}"

      # 响应时间告警
      - alert: HighLatency
        expr: |
          histogram_quantile(0.99, rate(http_request_duration_ms_bucket[5m])) > 1000
        for: 5m
        annotations:
          summary: "P99延迟超过1秒"
          description: "服务 {{ $labels.job }} P99延迟为 {{ $value }}ms"

      # QPS 异常告警
      - alert: LowTraffic
        expr: |
          sum(rate(http_requests_total[5m])) < 10
        for: 10m
        annotations:
          summary: "QPS异常低"
          description: "服务 {{ $labels.job }} QPS为 {{ $value }}"
```

## 🎯 最佳实践

### 1. 指标命名规范

```go
// ✅ 好的命名
"http_requests_total"           // 清晰的含义
"order_processing_duration_ms"  // 描述性强
"cache_hit_rate"                // 易于理解

// ❌ 不好的命名
"req"                          // 不清晰
"time"                         // 太通用
"my_metric"                    // 无意义
```

### 2. 合理使用标签

```go
// ✅ 好的做法 - 适量标签
metric.WithAttributes(
    attribute.String("method", "GET"),
    attribute.String("status", "200"),
    attribute.String("path", "/api/user"),
)

// ❌ 不好的做法 - 过多标签
metric.WithAttributes(
    attribute.String("user_id", "123"),      // 高基数！
    attribute.String("request_id", "abc"),   // 高基数！
    attribute.String("timestamp", "..."),    // 不应该作为标签
)
```

### 3. 选择正确的指标类型

```go
// Counter - 只增不减的计数
orderCounter.Add(ctx, 1)  // ✅ 订单总数

// Histogram - 分布统计
duration.Record(ctx, latency)  // ✅ 响应时间

// UpDownCounter - 可增可减
activeUsers.Add(ctx, 1)   // ✅ 在线用户数
activeUsers.Add(ctx, -1)  // 用户下线
```

### 4. 避免高基数标签

```go
// ❌ 危险 - user_id 有数百万个值
metric.WithAttributes(
    attribute.String("user_id", userID),  // 造成指标爆炸！
)

// ✅ 安全 - user_type 只有几个值
metric.WithAttributes(
    attribute.String("user_type", userType),  // VIP, Normal, etc.
)
```

## ⚠️ 注意事项

1. **在路由注册之前初始化** - InitMetrics 必须最先调用
2. **避免高基数标签** - 如 user_id、request_id 等
3. **标签值不要动态变化** - 会导致时间序列爆炸
4. **合理设置抓取间隔** - 一般 15s 或 30s
5. **监控 metrics 端点性能** - 避免采集耗时过长

## 🔗 相关模块

- [Trace](trace.md) - 链路追踪
- [Log](log.md) - 日志记录
- [Middlewares](middlewares.md) - HTTP 中间件

## 📖 推荐阅读

- [Prometheus 官方文档](https://prometheus.io/docs/)
- [OpenTelemetry Metrics](https://opentelemetry.io/docs/specs/otel/metrics/)
- [Prometheus 最佳实践](https://prometheus.io/docs/practices/)

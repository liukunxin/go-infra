# Trace - 分布式链路追踪

基于 OpenTelemetry 的分布式链路追踪，支持 Jaeger、Zipkin 等多种后端。

## 📖 功能特性

- ✅ 基于 OpenTelemetry 标准
- ✅ 自动生成唯一 TraceID
- ✅ 支持上下游链路传播
- ✅ Gin中间件自动埋点
- ✅ 支持采样率配置
- ✅ 可扩展的Exporter
- ✅ 每个请求唯一 TraceID

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/trace"
    "github.com/liukunxin/go-infra/pkg/middlewares"
)

func main() {
    // 1. 初始化 Trace（使用默认配置）
    trace.Init()

    // 2. 创建 Gin 路由
    router := gin.Default()

    // 3. 注册 Trace 中间件（每个请求自动生成 TraceID）
    router.Use(middlewares.GinTraceMiddleware())

    // 4. 业务路由
    router.GET("/api/user/:id", func(c *gin.Context) {
        ctx := c.Request.Context()
        
        // 获取当前请求的 TraceID
        traceID := trace.GetTraceID(ctx)
        
        c.JSON(200, gin.H{
            "trace_id": traceID,
            "message":  "success",
        })
    })

    router.Run(":8080")
}
```

## 📋 配置选项

### 基础配置

```go
// 使用默认配置
trace.Init()

// 自定义服务名
trace.Init(
    trace.WithServiceName("my-service"),
)

// 配置采样率（10%采样）
trace.Init(
    trace.WithServiceName("my-service"),
    trace.WithSampleRatio(0.1),
)
```

### 配置 Exporter（导出到 Jaeger）

```go
import (
    "go.opentelemetry.io/otel/exporters/jaeger"
    "github.com/liukunxin/go-infra/pkg/trace"
)

func main() {
    // 创建 Jaeger Exporter
    exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
        jaeger.WithEndpoint("http://localhost:14268/api/traces"),
    ))
    if err != nil {
        log.Fatal(err)
    }

    // 初始化 Trace
    trace.Init(
        trace.WithServiceName("my-service"),
        trace.WithSpanExporter(exporter),
        trace.WithSampleRatio(1.0),  // 100%采样
    )
    
    defer trace.Flush()  // 程序退出前刷新数据
}
```

## 💡 使用示例

### 示例1：基础 HTTP 服务

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/trace"
    "github.com/liukunxin/go-infra/pkg/log"
    "github.com/liukunxin/go-infra/pkg/middlewares"
)

func main() {
    // 初始化
    trace.Init(trace.WithServiceName("user-service"))
    log.Init(log.Config{Level: "info"})

    router := gin.Default()
    router.Use(middlewares.GinTraceMiddleware())
    router.Use(middlewares.HttpLogRecord())

    router.GET("/user/:id", GetUserHandler)
    router.Run(":8080")
}

func GetUserHandler(c *gin.Context) {
    ctx := c.Request.Context()
    
    // 日志自动包含 TraceID
    log.WithContext(ctx).Info("查询用户信息")
    
    // 业务逻辑...
    user, err := getUserByID(ctx, c.Param("id"))
    if err != nil {
        log.WithContext(ctx).Error("查询失败: %v", err)
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // 响应包含 TraceID
    c.JSON(200, gin.H{
        "trace_id": trace.GetTraceID(ctx),
        "data":     user,
    })
}
```

### 示例2：创建子 Span（追踪内部操作）

```go
import (
    "context"
    "go.opentelemetry.io/otel"
)

func processOrder(ctx context.Context, orderID string) error {
    // 创建子 Span 追踪数据库操作
    ctx, span := otel.Tracer("order-service").Start(ctx, "查询订单")
    defer span.End()
    
    // 数据库操作
    order, err := db.GetOrder(ctx, orderID)
    if err != nil {
        span.RecordError(err)  // 记录错误
        return err
    }
    
    // 添加属性
    span.SetAttributes(
        attribute.String("order_id", order.ID),
        attribute.Float64("amount", order.Amount),
    )
    
    return nil
}
```

### 示例3：完整的链路追踪

```go
func handlePayment(c *gin.Context) {
    ctx := c.Request.Context()
    tracer := otel.Tracer("payment-service")
    
    // 1. 验证订单
    ctx, span1 := tracer.Start(ctx, "验证订单")
    if err := validateOrder(ctx, orderID); err != nil {
        span1.RecordError(err)
        span1.End()
        return
    }
    span1.End()
    
    // 2. 调用支付API
    ctx, span2 := tracer.Start(ctx, "调用支付API")
    result, err := callPaymentAPI(ctx, order)
    if err != nil {
        span2.RecordError(err)
        span2.End()
        return
    }
    span2.SetAttributes(
        attribute.String("transaction_id", result.TransactionID),
    )
    span2.End()
    
    // 3. 更新订单状态
    ctx, span3 := tracer.Start(ctx, "更新订单")
    updateOrder(ctx, orderID, "paid")
    span3.End()
}
```

### 示例4：跨服务调用（HTTP传播 TraceID）

```go
import (
    "net/http"
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)

// 客户端：调用下游服务
func callDownstream(ctx context.Context) error {
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://downstream/api", nil)
    
    // 注入 TraceID 到 HTTP Header
    otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}

// 服务端：自动提取 TraceID
// GinTraceMiddleware 会自动从 HTTP Header 提取 TraceID
router.Use(middlewares.GinTraceMiddleware())
```

## 🎯 TraceID 使用说明

### TraceID vs SpanID

```
TraceID (推荐使用):
├─ 作用: 标识整个请求链路
├─ 特点: 一个请求只有一个 TraceID
└─ 用途: 日志关联、问题排查

SpanID (可选):
├─ 作用: 标识链路中的每个操作
├─ 特点: 一个请求有多个 SpanID
└─ 用途: 性能分析、调用树构建
```

### 获取 TraceID

```go
// 方法1：使用辅助函数
traceID := trace.GetTraceID(ctx)

// 方法2：使用 OpenTelemetry 原生API
span := trace.SpanFromContext(ctx)
traceID := span.SpanContext().TraceID().String()

// 响应中返回 TraceID
c.JSON(200, gin.H{
    "trace_id": traceID,
    "data":     result,
})
```

### TraceID 传播

```
请求流程:
1. 客户端请求 → 网关
2. 网关提取/生成 TraceID → 服务A
3. 服务A携带 TraceID → 服务B
4. 服务B携带 TraceID → 数据库/缓存
5. 所有日志都包含同一个 TraceID
```

## 🔧 高级用法

### 自定义 Span 属性

```go
import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

func processData(ctx context.Context) error {
    ctx, span := otel.Tracer("my-service").Start(ctx, "数据处理")
    defer span.End()
    
    // 添加属性
    span.SetAttributes(
        attribute.String("user_id", "123"),
        attribute.Int("batch_size", 100),
        attribute.Bool("is_retry", false),
    )
    
    // 处理逻辑...
    if err != nil {
        // 标记为错误
        span.SetStatus(codes.Error, "处理失败")
        span.RecordError(err)
        return err
    }
    
    // 标记为成功
    span.SetStatus(codes.Ok, "")
    return nil
}
```

### 采样策略

```go
// 场景1：开发环境 - 100%采样
trace.Init(
    trace.WithServiceName("my-service"),
    trace.WithSampleRatio(1.0),
)

// 场景2：生产环境 - 10%采样（减少开销）
trace.Init(
    trace.WithServiceName("my-service"),
    trace.WithSampleRatio(0.1),
)

// 场景3：不采样（只生成TraceID用于日志）
trace.Init(
    trace.WithServiceName("my-service"),
    // 不配置 Exporter，只生成 TraceID
)
```

## 🎯 最佳实践

### 1. 确保上下文传递

```go
// ✅ 好的做法 - 传递 context
func handleRequest(c *gin.Context) {
    ctx := c.Request.Context()
    processData(ctx)  // 传递 ctx
}

func processData(ctx context.Context) {
    queryDatabase(ctx)  // 继续传递
}

// ❌ 不好的做法 - 不传递 context
func processData() {
    queryDatabase()  // 丢失 TraceID
}
```

### 2. Span 命名规范

```go
// ✅ 好的命名 - 清晰描述操作
tracer.Start(ctx, "查询用户信息")
tracer.Start(ctx, "MySQL: SELECT * FROM users")
tracer.Start(ctx, "调用支付API")

// ❌ 不好的命名 - 模糊不清
tracer.Start(ctx, "process")
tracer.Start(ctx, "step1")
```

### 3. 合理使用 Span

```go
// ✅ 好的做法 - 重要操作创建 Span
func processOrder(ctx context.Context) {
    // 数据库操作
    ctx, span := tracer.Start(ctx, "查询订单")
    order := db.Query(ctx)
    span.End()
    
    // RPC调用
    ctx, span = tracer.Start(ctx, "调用库存服务")
    inventory := rpc.Call(ctx)
    span.End()
}

// ❌ 不好的做法 - 过度使用 Span
for i := 0; i < 1000; i++ {
    ctx, span := tracer.Start(ctx, "处理单条记录")  // 太多了！
    process(item)
    span.End()
}
```

### 4. 错误处理

```go
func callAPI(ctx context.Context) error {
    ctx, span := tracer.Start(ctx, "调用外部API")
    defer span.End()
    
    resp, err := http.Get(url)
    if err != nil {
        // 记录错误
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    if resp.StatusCode != 200 {
        err := fmt.Errorf("HTTP %d", resp.StatusCode)
        span.RecordError(err)
        span.SetStatus(codes.Error, err.Error())
        return err
    }
    
    span.SetStatus(codes.Ok, "")
    return nil
}
```

## 📊 Trace 数据可视化

### Jaeger UI 示例

```
TraceID: abc123def456
Duration: 234ms

Span 树:
├─ [50ms] HTTP GET /api/order
   ├─ [20ms] 验证用户权限
   ├─ [100ms] 查询订单信息
   │  └─ [80ms] MySQL: SELECT * FROM orders
   ├─ [50ms] 调用库存服务
   │  └─ [30ms] HTTP GET /api/inventory
   └─ [10ms] 返回响应
```

## ⚠️ 注意事项

1. **必须注册中间件** - GinTraceMiddleware 必须最先注册
2. **上下文传递** - 确保 context 在整个调用链中传递
3. **Span 要关闭** - 使用 `defer span.End()` 确保 Span 关闭
4. **采样率权衡** - 高采样率会增加性能开销
5. **TraceID 唯一性** - 由中间件自动保证，不需要手动生成

## 🔗 相关模块

- [Log](log.md) - 自动记录 TraceID
- [Metrics](metrics.md) - 追踪性能指标
- [Middlewares](middlewares.md) - Trace 中间件

## 📖 推荐阅读

- [OpenTelemetry 官方文档](https://opentelemetry.io/docs/)
- [Jaeger 官方文档](https://www.jaegertracing.io/docs/)
- [TraceID vs SpanID 指南](trace_guide.md)

# Traffic - 流量控制

流量控制接口定义，支持限流、熔断等流控策略的扩展实现。

## 📖 功能特性

- ✅ 统一的流量控制接口
- ✅ 支持多种流控策略（限流/熔断）
- ✅ 可插拔的 Controller 实现
- ✅ 默认空实现（DummyController）
- ✅ 线程安全
- ✅ 易于扩展

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "context"
    "github.com/liukunxin/go-infra/pkg/traffic"
    "log"
)

func main() {
    // 1. 使用默认实现（不做任何限制）
    // traffic 模块会自动初始化为 DummyController

    // 2. 尝试通过流控
    pass, blockErr := traffic.GetController().TryPass("order_create")
    if blockErr != nil {
        log.Printf("请求被限流: %s", blockErr.BlockMsg())
        return
    }
    defer pass.Done()

    // 3. 业务逻辑
    err := processOrder()
    if err != nil {
        pass.Error(err)  // 记录错误
        return
    }
}
```

## 📋 接口定义

### Controller 接口

```go
type Controller interface {
    TryPass(resource string, opts ...TryPassOption) (Pass, BlockError)
}
```

### Pass 接口

```go
type Pass interface {
    Error(err error)  // 记录错误
    Done()            // 完成调用
}
```

### BlockError 接口

```go
type BlockError interface {
    error
    BlockType() BlockType  // 阻塞类型
    BlockMsg() string      // 阻塞原因
}
```

### BlockType 类型

```go
const (
    BlockTypeLimit           // 限流
    BlockTypeCircuitBreaking // 熔断
    BlockTypeInternal        // 内部错误
)
```

## 💡 使用示例

### 示例1：在 HTTP Handler 中使用

```go
func orderHandler(c *gin.Context) {
    // 尝试通过流控
    pass, blockErr := traffic.GetController().TryPass("api:/order/create")
    if blockErr != nil {
        // 根据类型返回不同错误
        switch blockErr.BlockType() {
        case traffic.BlockTypeLimit:
            c.JSON(429, gin.H{"error": "请求过于频繁"})
        case traffic.BlockTypeCircuitBreaking:
            c.JSON(503, gin.H{"error": "服务暂时不可用"})
        default:
            c.JSON(500, gin.H{"error": blockErr.BlockMsg()})
        }
        return
    }
    defer pass.Done()

    // 处理订单
    order, err := createOrder(c)
    if err != nil {
        pass.Error(err)  // 记录错误（用于熔断判断）
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, order)
}
```

### 示例2：在 Service 层使用

```go
type OrderService struct {
    // ...
}

func (s *OrderService) CreateOrder(ctx context.Context, req *CreateOrderReq) (*Order, error) {
    // 流控检查
    pass, blockErr := traffic.GetController().TryPass("service:order:create")
    if blockErr != nil {
        return nil, fmt.Errorf("流控拒绝: %s", blockErr.BlockMsg())
    }
    defer pass.Done()

    // 业务逻辑
    order, err := s.repo.Create(ctx, req)
    if err != nil {
        pass.Error(err)  // 记录错误
        return nil, err
    }

    return order, nil
}
```

### 示例3：自定义 Controller（Sentinel 实现）

```go
import (
    sentinel "github.com/alibaba/sentinel-golang/api"
    "github.com/alibaba/sentinel-golang/core/base"
    "github.com/liukunxin/go-infra/pkg/traffic"
)

type SentinelController struct{}

func (c *SentinelController) TryPass(resource string, opts ...traffic.TryPassOption) (traffic.Pass, traffic.BlockError) {
    entry, err := sentinel.Entry(resource)
    if err != nil {
        // 被流控
        return nil, &SentinelBlockError{err: err}
    }
    
    return &SentinelPass{entry: entry}, nil
}

type SentinelPass struct {
    entry *base.SentinelEntry
}

func (p *SentinelPass) Error(err error) {
    sentinel.TraceError(p.entry, err)
}

func (p *SentinelPass) Done() {
    p.entry.Exit()
}

type SentinelBlockError struct {
    err error
}

func (e *SentinelBlockError) Error() string {
    return e.err.Error()
}

func (e *SentinelBlockError) BlockType() traffic.BlockType {
    // 根据 Sentinel 错误类型判断
    return traffic.BlockTypeLimit
}

func (e *SentinelBlockError) BlockMsg() string {
    return "请求过于频繁"
}

// 初始化
func initSentinel() {
    // 初始化 Sentinel
    sentinel.InitDefault()
    
    // 配置规则...
    
    // 设置自定义 Controller
    traffic.SetController(&SentinelController{})
}
```

### 示例4：自定义限流 Controller

```go
import (
    "golang.org/x/time/rate"
    "sync"
)

type RateLimitController struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
}

func NewRateLimitController() *RateLimitController {
    return &RateLimitController{
        limiters: make(map[string]*rate.Limiter),
    }
}

func (c *RateLimitController) AddRule(resource string, rps int) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.limiters[resource] = rate.NewLimiter(rate.Limit(rps), rps)
}

func (c *RateLimitController) TryPass(resource string, opts ...traffic.TryPassOption) (traffic.Pass, traffic.BlockError) {
    c.mu.RLock()
    limiter, exists := c.limiters[resource]
    c.mu.RUnlock()

    if !exists {
        // 没有规则，直接通过
        return &dummyPass{}, nil
    }

    if !limiter.Allow() {
        return nil, &LimitBlockError{resource: resource}
    }

    return &dummyPass{}, nil
}

type LimitBlockError struct {
    resource string
}

func (e *LimitBlockError) Error() string {
    return fmt.Sprintf("资源 %s 访问过于频繁", e.resource)
}

func (e *LimitBlockError) BlockType() traffic.BlockType {
    return traffic.BlockTypeLimit
}

func (e *LimitBlockError) BlockMsg() string {
    return "请求频率超限"
}

// 使用示例
func main() {
    controller := NewRateLimitController()
    controller.AddRule("api:/order/create", 100)  // 100 QPS
    controller.AddRule("api:/user/query", 1000)   // 1000 QPS
    
    traffic.SetController(controller)
}
```

### 示例5：熔断器实现

```go
type CircuitBreakerController struct {
    breakers map[string]*CircuitBreaker
    mu       sync.RWMutex
}

type CircuitBreaker struct {
    failureThreshold int
    failureCount     int32
    state            int32  // 0=closed, 1=open
    lastFailTime     time.Time
    mu               sync.Mutex
}

func (cb *CircuitBreaker) TryPass() bool {
    state := atomic.LoadInt32(&cb.state)
    if state == 1 {
        // 熔断状态
        cb.mu.Lock()
        defer cb.mu.Unlock()
        
        // 检查是否可以半开
        if time.Since(cb.lastFailTime) > 10*time.Second {
            atomic.StoreInt32(&cb.state, 0)
            atomic.StoreInt32(&cb.failureCount, 0)
            return true
        }
        return false
    }
    return true
}

func (cb *CircuitBreaker) RecordError() {
    count := atomic.AddInt32(&cb.failureCount, 1)
    if count >= int32(cb.failureThreshold) {
        cb.mu.Lock()
        defer cb.mu.Unlock()
        
        atomic.StoreInt32(&cb.state, 1)  // 打开熔断器
        cb.lastFailTime = time.Now()
    }
}

func (cb *CircuitBreaker) RecordSuccess() {
    atomic.StoreInt32(&cb.failureCount, 0)
}
```

## 🎯 最佳实践

### 1. 资源命名规范

```go
// ✅ 好的命名
"api:/order/create"          // API 路径
"service:order:query"        // 服务方法
"db:mysql:query"             // 数据库操作
"cache:redis:get"            // 缓存操作

// ❌ 不好的命名
"order"                      // 不够具体
"api1"                       // 无意义
```

### 2. 及时调用 Done()

```go
// ✅ 好的做法
pass, blockErr := traffic.GetController().TryPass(resource)
if blockErr != nil {
    return err
}
defer pass.Done()  // 确保调用

// ❌ 不好的做法
pass, _ := traffic.GetController().TryPass(resource)
// 忘记调用 Done()
```

### 3. 记录错误

```go
// ✅ 好的做法 - 记录错误（用于熔断统计）
if err := doSomething(); err != nil {
    pass.Error(err)
    return err
}

// ❌ 不好的做法 - 不记录错误
if err := doSomething(); err != nil {
    return err  // 熔断器无法统计错误
}
```

### 4. 根据场景选择策略

```go
// API 入口 - 限流
pass, _ := traffic.GetController().TryPass("api:/order/create")

// 外部调用 - 熔断
pass, _ := traffic.GetController().TryPass("rpc:payment-service")

// 数据库操作 - 限流+熔断
pass, _ := traffic.GetController().TryPass("db:mysql:orders")
```

## 📊 集成流行的流控框架

### Sentinel

```bash
go get github.com/alibaba/sentinel-golang
```

### Hystrix

```bash
go get github.com/afex/hystrix-go
```

### 自定义实现

```go
// 实现 Controller 接口即可
type MyController struct {
    // ...
}

func (c *MyController) TryPass(resource string, opts ...traffic.TryPassOption) (traffic.Pass, traffic.BlockError) {
    // 自定义逻辑
}
```

## ⚠️ 注意事项

1. **默认不限流** - DummyController 不做任何限制
2. **必须调用 Done()** - 用于资源释放和统计
3. **记录错误** - 熔断器依赖错误统计
4. **性能开销** - 流控会增加少量性能开销
5. **合理配置阈值** - 根据实际场景设置限流值

## 🔗 相关模块

- [Metrics](metrics.md) - 流控指标监控
- [Log](log.md) - 流控日志记录
- [Middlewares](middlewares.md) - HTTP 流控中间件

## 📖 推荐阅读

- [Sentinel 官方文档](https://sentinelguard.io/zh-cn/)
- [Hystrix Wiki](https://github.com/Netflix/Hystrix/wiki)
- [微服务流量控制实践](https://www.alibabacloud.com/blog/595162)

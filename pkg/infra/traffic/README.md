# Traffic - 流量控制

限流 + 熔断一体化解决方案，基于统一 `Controller` 接口，支持单独使用或组合叠加。

---

## 核心概念

```
业务代码调用 TryPass("资源名")
    ├── 允许通过 → 返回 Pass token
    │       ├── 业务成功 → pass.Done()
    │       └── 业务失败 → pass.Error(err)
    └── 拒绝 → 返回 BlockError（含拒绝原因）
```

- **Pass.Done()** — 告知流控"本次请求成功"
- **Pass.Error(err)** — 告知流控"本次请求失败"（熔断器会计入失败率）
- 两者**只有第一次调用生效**，可以安全地 `defer pass.Done()` 再按需调 `pass.Error(err)`

---

## 限流（RateLimitController）

使用令牌桶算法，限制每秒最多允许多少请求通过。

### 初始化

```go
import "github.com/yourorg/go-infra/pkg/infra/traffic"

// 每秒最多 500 个请求，允许瞬间突发 50 个
ctrl := traffic.NewRateLimitController(500, 50)
traffic.Init(traffic.WithController(ctrl))
```

### 使用

```go
func createOrder(ctx context.Context) error {
    pass, blockErr := traffic.GetController().TryPass("order_create")
    if blockErr != nil {
        // 超出限流阈值，直接返回，不执行业务逻辑
        return errors.New("系统繁忙，请稍后重试")
    }
    defer pass.Done()  // 正常结束时标记成功

    // 执行业务逻辑
    if err := doCreateOrder(ctx); err != nil {
        pass.Error(err)
        return err
    }
    return nil
}
```

---

## 熔断（CircuitBreakerController）

当某个资源的错误率超过阈值时，自动"断开"保护下游，一段时间后自动探活恢复。

### 三种状态

```
Closed（正常）──[错误率超阈值]──▶ Open（熔断）
Open   ──[冷却期结束]──▶ Half-Open（探活）
Half-Open ──[探活成功]──▶ Closed（恢复）
Half-Open ──[探活失败]──▶ Open（继续熔断）
```

### 初始化

```go
ctrl := traffic.NewCircuitBreakerController(traffic.CircuitBreakerConfig{
    ErrorRateThreshold:       0.5,              // 错误率超 50% 则熔断
    MinRequests:              10,               // 窗口内至少 10 次请求才触发判断
    WindowSize:               20,               // 滑动窗口大小（最近 20 次请求）
    CooldownPeriod:           5 * time.Second,  // 熔断冷却 5 秒后进入探活
    HalfOpenMaxRequests:      1,                // 探活期最多允许 1 个请求通过
    HalfOpenSuccessThreshold: 1,                // 探活成功 1 次即恢复
})
traffic.Init(traffic.WithController(ctrl))
```

### 使用（与限流完全相同的调用方式）

```go
func callDownstream(ctx context.Context) error {
    pass, blockErr := traffic.GetController().TryPass("payment_service")
    if blockErr != nil {
        // 熔断器打开，快速失败
        return errors.New("支付服务暂时不可用")
    }
    defer pass.Done()

    if err := callPaymentService(ctx); err != nil {
        pass.Error(err)   // 失败计入熔断器
        return err
    }
    return nil
}
```

### 查看熔断状态（健康检查 / 监控）

```go
cb := traffic.GetController().(*traffic.CircuitBreakerController)
for _, s := range cb.States() {
    fmt.Printf("资源: %-20s 状态: %-10s 错误率: %.1f%% 请求数: %d\n",
        s.Resource, s.State, s.FailRate*100, s.Total)
}
```

---

## 限流 + 熔断组合（CompositeController）

同时启用两种策略时，**先限流再熔断**，任一拒绝即返回。

```go
rateLimiter := traffic.NewRateLimitController(500, 50)
circuitBreaker := traffic.NewCircuitBreakerController(traffic.CircuitBreakerConfig{
    ErrorRateThreshold: 0.5,
    MinRequests:        10,
    CooldownPeriod:     5 * time.Second,
})

ctrl := traffic.NewCompositeController(rateLimiter, circuitBreaker)
traffic.Init(traffic.WithController(ctrl))
```

调用方式**不变**，业务代码无需关心底层用了哪种策略：

```go
pass, blockErr := traffic.GetController().TryPass("order_create")
if blockErr != nil {
    switch blockErr.BlockType() {
    case traffic.BlockTypeRateLimit:
        return errors.New("请求频率过高，请稍后重试")
    case traffic.BlockTypeCircuitBreaking:
        return errors.New("服务暂时不可用，请稍后重试")
    }
}
defer pass.Done()
```

---

## 不做任何限制（测试 / 默认）

```go
// Init 不传参数，或使用 DummyController
traffic.Init()  // 等价于 traffic.Init(traffic.WithController(traffic.NewDummyController()))
```

---

## 推荐配置参考

| 场景                   | 建议配置                                                              |
|------------------------|----------------------------------------------------------------------|
| 对外 API 限流           | `RateLimit(1000, 100)` — QPS 1000，突发 100                          |
| 调用不稳定下游          | `CircuitBreaker{ErrorRate: 0.5, MinReq: 10, Cooldown: 5s}`           |
| 对外接口 + 下游保护     | `Composite(RateLimit, CircuitBreaker)`                               |

---

## 接口定义速查

```go
// Controller — 流量控制器
type Controller interface {
    TryPass(resource string, opts ...TryPassOption) (Pass, BlockError)
}

// Pass — 通过令牌，用于回报结果
type Pass interface {
    Done()           // 标记成功
    Error(err error) // 标记失败
}

// BlockError — 被拒绝时返回的错误
type BlockError interface {
    error
    BlockType() BlockType  // RateLimit | CircuitBreaking
    BlockMsg()  string
}
```

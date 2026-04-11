# TraceID vs SpanID 使用指南

## 概念对比

### TraceID - 请求链路标识
- **作用**：标识整个请求链路，从请求进入到响应返回
- **特点**：一个请求只有一个TraceID，所有操作共享
- **用途**：日志关联、问题排查、请求追踪

### SpanID - 操作片段标识  
- **作用**：标识链路中的每个具体操作
- **特点**：一个请求有多个SpanID，每个操作一个
- **用途**：性能分析、调用树构建、细粒度追踪

## 使用场景对比

### 场景1：简单的日志关联（只需TraceID）✅

**适用于：**
- 单体应用或简单的微服务
- 主要用于日志聚合和问题排查
- 不需要详细的性能分析

**示例：**
```
[TraceID: abc123] 用户登录请求开始
[TraceID: abc123] 查询用户信息
[TraceID: abc123] 验证密码成功
[TraceID: abc123] 生成token
[TraceID: abc123] 登录成功
```

**优点：**
- ✅ 简单易懂
- ✅ 性能开销小
- ✅ 满足90%的日志追踪需求

### 场景2：复杂的分布式追踪（需要TraceID + SpanID）🔍

**适用于：**
- 复杂的微服务架构
- 需要详细的性能分析
- 使用Jaeger/Zipkin等追踪工具

**示例：**
```
TraceID: abc123
├─ [SpanID: span1] HTTP /api/order (总耗时: 200ms)
   ├─ [SpanID: span2] 查询用户服务 (50ms)
   │  └─ [SpanID: span3] MySQL: select users (30ms)
   ├─ [SpanID: span4] 查询库存服务 (80ms)
   │  ├─ [SpanID: span5] Redis: get stock (10ms)
   │  └─ [SpanID: span6] MySQL: update stock (60ms)
   └─ [SpanID: span7] 创建订单 (40ms)
```

**优点：**
- ✅ 精确定位性能瓶颈
- ✅ 可视化调用关系
- ✅ 细粒度的操作追踪

**缺点：**
- ❌ 需要额外的追踪基础设施
- ❌ 增加存储和计算成本
- ❌ 增加系统复杂度

## 决策树

```
是否使用 SpanID？
│
├─ 只需要日志关联？ → 只用 TraceID ✅
│
├─ 需要性能分析和调用树？ → 使用 TraceID + SpanID 🔍
│
├─ 有 Jaeger/Zipkin 等工具？ → 使用 TraceID + SpanID 🔍
│
└─ 简单的问题排查？ → 只用 TraceID ✅
```

## 实现建议

### 方案A：简化版（推荐给大多数团队）

```go
// 只记录 TraceID
log.WithContext(ctx).Info("用户登录")
// 输出: [trace_id=abc123] 用户登录
```

**配置：**
```go
// pkg/log/log.go
func WithContext(ctx context.Context) *ContextLogger {
    // 只提取 TraceID
    traceID := trace.GetTraceID(ctx)
    return &ContextLogger{
        l:       logger,
        traceId: traceID,
        spanId:  "", // 不使用 SpanID
    }
}
```

### 方案B：完整版（适合复杂场景）

```go
// 记录 TraceID + SpanID
log.WithContext(ctx).Info("数据库查询")
// 输出: [trace_id=abc123] [span_id=span2] 数据库查询
```

**配置：**
```go
// pkg/log/log.go
func WithContext(ctx context.Context) *ContextLogger {
    // 提取 TraceID 和 SpanID
    span := trace.SpanFromContext(ctx)
    spanCtx := span.SpanContext()
    traceID := spanCtx.TraceID().String()
    spanID := spanCtx.SpanID().String()
    return &ContextLogger{
        l:       logger,
        traceId: traceID,
        spanId:  spanID,
    }
}
```

## 性能对比

| 项目 | 只用TraceID | TraceID + SpanID |
|-----|------------|-----------------|
| CPU开销 | 低 | 中等 |
| 内存开销 | 低 | 中等 |
| 存储开销 | 低 | 高（需要存储span数据）|
| 查询复杂度 | 简单 | 复杂 |
| 学习成本 | 低 | 高 |

## 最佳实践

### 1. 根据业务场景选择

- **小型项目/MVP阶段**：只用TraceID
- **成熟的微服务**：考虑TraceID + SpanID
- **性能敏感**：只用TraceID
- **问题频发**：考虑TraceID + SpanID

### 2. 渐进式采用

```
阶段1: 只用 TraceID
  ↓ （业务增长，需要更细粒度追踪）
阶段2: 关键路径添加 Span
  ↓ （引入链路追踪工具）
阶段3: 全面使用 TraceID + SpanID
```

### 3. 成本考虑

**只用TraceID：**
- 几乎无额外成本
- 日志存储正常增长

**TraceID + SpanID：**
- 需要Jaeger/Zipkin等基础设施
- 存储成本增加30-50%
- 需要专人维护

## 总结

**对于大多数团队，建议从只使用TraceID开始：**

✅ 满足日常日志关联需求
✅ 简单易维护
✅ 性能开销小
✅ 成本低

**只有在以下情况才考虑SpanID：**

- 🔍 复杂的微服务架构
- 🔍 频繁出现性能问题
- 🔍 已有链路追踪基础设施
- 🔍 团队有专业的运维能力

## 当前项目建议

根据你们"很少用到SpanID"的反馈，建议：

1. **简化日志模块**：只记录TraceID
2. **保留SpanID接口**：预留扩展能力
3. **按需启用**：需要时再开启详细追踪

这样既能满足当前需求，又保留了未来扩展的灵活性。

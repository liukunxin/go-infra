# Log - 高性能日志模块

基于环形队列的异步日志系统，支持链路追踪集成，性能优异，并发安全。

## 📖 功能特性

- ✅ 异步日志输出（基于环形队列）
- ✅ 自动集成TraceID/SpanID
- ✅ 结构化日志（支持字段）
- ✅ 对象池优化（减少GC压力）
- ✅ 无锁设计（高并发友好）
- ✅ 支持多种日志级别
- ✅ 可自定义Formatter和Provider

## 🚀 快速开始

### 基础用法

```go
package main

import (
    "github.com/liukunxin/go-infra/pkg/log"
)

func main() {
    // 1. 初始化日志
    log.Init(log.Config{
        Level:      "info",
        OutputPath: "stdout",  // 或文件路径
    })

    // 2. 基础日志
    log.New().Info("应用启动成功")
    log.New().Error("发生错误: %s", "数据库连接失败")

    // 3. 带上下文的日志（自动包含TraceID）
    ctx := getRequestContext()
    log.WithContext(ctx).Info("处理用户请求")

    // 4. 带字段的结构化日志
    log.WithContext(ctx).WithFields(map[string]interface{}{
        "user_id": 123,
        "action":  "login",
        "ip":      "192.168.1.1",
    }).Info("用户登录成功")
}
```

## 📋 日志级别

```go
const (
    LevelDebug = 0  // 调试信息
    LevelInfo  = 1  // 一般信息
    LevelWarn  = 2  // 警告信息
    LevelError = 3  // 错误信息
    LevelFatal = 4  // 致命错误
)
```

## 💡 使用示例

### 示例1：在Gin中使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/log"
    "github.com/liukunxin/go-infra/pkg/middlewares"
)

func main() {
    // 初始化日志
    log.Init(log.Config{Level: "info"})

    router := gin.Default()
    
    // 注册日志中间件
    router.Use(middlewares.HttpLogRecord())

    router.GET("/user/:id", func(c *gin.Context) {
        ctx := c.Request.Context()
        userID := c.Param("id")

        // 带上下文的日志（自动包含TraceID）
        log.WithContext(ctx).WithFields(map[string]interface{}{
            "user_id": userID,
        }).Info("查询用户信息")

        // 业务逻辑...
        user, err := getUserByID(userID)
        if err != nil {
            log.WithContext(ctx).Error("查询失败: %v", err)
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }

        log.WithContext(ctx).Info("查询成功")
        c.JSON(200, user)
    })

    router.Run(":8080")
}
```

### 示例2：结构化日志

```go
// 记录详细的业务操作
log.WithContext(ctx).WithFields(map[string]interface{}{
    "order_id":     "ORD12345",
    "user_id":      123,
    "amount":       99.99,
    "payment_type": "alipay",
    "status":       "success",
}).Info("订单支付成功")

// 日志输出示例：
// [2024-01-15 10:23:45] [INFO] [trace_id=abc123] 订单支付成功 order_id=ORD12345 user_id=123 amount=99.99 payment_type=alipay status=success
```

### 示例3：错误日志

```go
func processPayment(ctx context.Context, order *Order) error {
    logger := log.WithContext(ctx).WithFields(map[string]interface{}{
        "order_id": order.ID,
        "amount":   order.Amount,
    })

    logger.Info("开始处理支付")

    // 业务逻辑
    if err := validateOrder(order); err != nil {
        logger.Error("订单验证失败: %v", err)
        return err
    }

    if err := callPaymentAPI(order); err != nil {
        logger.Error("调用支付API失败: %v", err)
        return err
    }

    logger.Info("支付处理成功")
    return nil
}
```

### 示例4：不同级别的日志

```go
func exampleLogLevels(ctx context.Context) {
    logger := log.WithContext(ctx)

    // Debug - 调试信息
    logger.Debug("调试信息: 变量x的值为 %d", 42)

    // Info - 一般信息
    logger.Info("用户登录成功")

    // Warn - 警告信息
    logger.Warn("缓存未命中，使用数据库查询")

    // Error - 错误信息
    logger.Error("数据库连接失败: %v", err)

    // Fatal - 致命错误（会导致程序退出）
    // logger.Fatal("配置文件加载失败")  // 慎用！
}
```

### 示例5：性能敏感场景

```go
// 高并发场景下的日志
func handleHighConcurrency(ctx context.Context, userID int64) {
    // ✅ 推荐：只在需要时创建logger
    if needLog {
        log.WithContext(ctx).Info("处理请求: %d", userID)
    }

    // ✅ 推荐：使用字段而非字符串拼接
    log.WithContext(ctx).WithFields(map[string]interface{}{
        "user_id": userID,
        "action":  "query",
    }).Info("用户操作")

    // ❌ 避免：频繁的字符串格式化
    // log.WithContext(ctx).Info(fmt.Sprintf("user_%d_query", userID))
}
```

## 🔧 配置选项

### Config结构

```go
type Config struct {
    Level      string  // 日志级别: "debug", "info", "warn", "error", "fatal"
    OutputPath string  // 输出路径: "stdout", "stderr" 或文件路径
    BufferSize int     // 环形队列大小（默认1024）
}
```

### 初始化配置

```go
// 开发环境配置
log.Init(log.Config{
    Level:      "debug",
    OutputPath: "stdout",
    BufferSize: 1024,
})

// 生产环境配置
log.Init(log.Config{
    Level:      "info",
    OutputPath: "/var/log/app.log",
    BufferSize: 4096,  // 高并发场景增大缓冲区
})
```

## 🎯 最佳实践

### 1. 使用WithContext自动记录TraceID

```go
// ✅ 好的做法 - 自动包含TraceID
log.WithContext(ctx).Info("用户操作")
// 输出: [2024-01-15 10:23:45] [INFO] [trace_id=abc123] 用户操作

// ❌ 不好的做法 - 丢失链路信息
log.New().Info("用户操作")
// 输出: [2024-01-15 10:23:45] [INFO] 用户操作
```

### 2. 使用WithFields记录结构化信息

```go
// ✅ 好的做法 - 结构化日志
log.WithContext(ctx).WithFields(map[string]interface{}{
    "user_id": 123,
    "action":  "login",
    "ip":      clientIP,
}).Info("用户操作")

// ❌ 不好的做法 - 字符串拼接
log.WithContext(ctx).Info(fmt.Sprintf("用户%d从%s登录", 123, clientIP))
```

### 3. 避免敏感信息泄露

```go
// ✅ 好的做法 - 脱敏处理
log.WithContext(ctx).WithFields(map[string]interface{}{
    "user_id": user.ID,
    "phone":   maskPhone(user.Phone),  // 138****5678
}).Info("用户注册")

// ❌ 不好的做法 - 暴露敏感信息
log.WithContext(ctx).WithFields(map[string]interface{}{
    "user_id":  user.ID,
    "password": user.Password,  // 危险！
    "phone":    user.Phone,
}).Info("用户注册")
```

### 4. 合理选择日志级别

```go
// Debug - 开发调试用
log.WithContext(ctx).Debug("SQL: %s", sql)

// Info - 正常业务流程
log.WithContext(ctx).Info("订单创建成功")

// Warn - 异常但可恢复
log.WithContext(ctx).Warn("缓存miss，降级到数据库")

// Error - 需要关注的错误
log.WithContext(ctx).Error("支付失败: %v", err)

// Fatal - 致命错误，程序无法继续
// log.WithContext(ctx).Fatal("配置文件不存在")
```

### 5. 在循环中避免过度日志

```go
// ❌ 不好的做法 - 大量日志
for i := 0; i < 10000; i++ {
    log.WithContext(ctx).Info("处理第%d条记录", i)  // 10000条日志！
}

// ✅ 好的做法 - 批量记录
for i := 0; i < 10000; i++ {
    // 处理逻辑...
    if i%1000 == 0 {
        log.WithContext(ctx).Info("已处理%d条记录", i)
    }
}
log.WithContext(ctx).Info("总共处理10000条记录")
```

## 🚀 性能优化

### 异步日志原理

```
应用代码
  ↓
  调用 log.Info()
  ↓
入队到环形队列 (非阻塞)
  ↓
  立即返回 ← 应用继续执行
  
后台消费者
  ↓
从队列取出日志
  ↓
格式化并写入
```

### 性能数据

- **单条日志耗时**: < 100ns（入队操作）
- **支持QPS**: > 1,000,000/秒
- **内存分配**: 使用对象池，减少GC压力
- **并发安全**: 无锁设计

### 性能建议

```go
// 1. 高并发场景增大缓冲区
log.Init(log.Config{
    BufferSize: 8192,  // 默认1024
})

// 2. 避免在热路径中创建大量临时对象
// ✅ 好的做法
fields := map[string]interface{}{
    "user_id": userID,
}
for _, item := range items {
    fields["item_id"] = item.ID
    log.WithContext(ctx).WithFields(fields).Info("处理商品")
}

// 3. 生产环境使用Info级别
log.Init(log.Config{
    Level: "info",  // 不输出Debug日志
})
```

## ⚠️ 注意事项

1. **Fatal级别会导致程序退出** - 只在致命错误时使用
2. **上下文传递** - 确保ctx从请求入口传递到各个函数
3. **日志文件轮转** - 生产环境需要配置日志轮转（logrotate）
4. **性能监控** - 监控日志队列是否经常满
5. **避免循环依赖** - 日志模块应该最先初始化

## 🔗 相关模块

- [Trace](trace.md) - 提供TraceID
- [Middlewares](middlewares.md) - HTTP日志中间件
- [Metrics](metrics.md) - 日志性能监控

## 📊 日志输出格式示例

```
[时间] [级别] [trace_id] 消息 字段1=值1 字段2=值2

示例：
[2024-01-15 10:23:45.123] [INFO] [trace_id=abc123def456] 用户登录成功 user_id=123 ip=192.168.1.1 action=login
```

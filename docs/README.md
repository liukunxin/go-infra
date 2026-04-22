# go-infra 文档中心

欢迎使用 go-infra 基础设施库！这里包含了所有模块的详细使用文档。

## 📚 文档导航

### 🔥 核心模块

| 模块 | 说明 | 文档链接 |
|-----|------|---------|
| **Log** | 高性能异步日志系统 | [查看文档](log.md) |
| **Trace** | 分布式链路追踪 | [查看文档](trace.md) |
| **Errors** | 统一错误处理 | [查看文档](errors.md) |
| **Metrics** | Prometheus监控 | [查看文档](metrics.md) |

### 💾 数据存储

| 模块 | 说明 | 文档链接 |
|-----|------|---------|
| **MySQL** | GORM封装 + 连接池 | [查看文档](mysql.md) |
| **Redis** | 单机/集群模式 | [查看文档](redis.md) |
| **Milvus** | 向量数据库客户端 | [查看文档](milvus.md) |

### 🛠️ 工具模块

| 模块 | 说明 | 文档链接 |
|-----|------|---------|
| **Traffic** | 流量控制（限流/熔断） | [查看文档](traffic.md) |
| **HTTP Client** | HTTP客户端 + 连接池 | [查看文档](http_client.md) |
| **Controller** | Gin基础控制器 | [查看文档](controller.md) |
| **Middlewares** | Gin中间件集合 | [查看文档](middlewares.md) |
| **Pay** | 微信 APIv3 / 支付宝 RSA2 支付 | [查看文档](pay.md) |

## 🚀 快速开始

### 1. 安装

```bash
go get github.com/liukunxin/go-infra
```

### 2. 基础示例

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/liukunxin/go-infra/pkg/log"
    "github.com/liukunxin/go-infra/pkg/trace"
    "github.com/liukunxin/go-infra/pkg/metrics"
    "github.com/liukunxin/go-infra/pkg/middlewares"
)

func main() {
    // 初始化
    log.Init(log.Config{Level: "info"})
    trace.Init()
    
    router := gin.Default()
    metrics.InitMetrics("my-service", router)
    
    // 中间件
    router.Use(middlewares.GinTraceMiddleware())
    router.Use(middlewares.HttpLogRecord())
    
    // 路由
    router.GET("/hello", func(c *gin.Context) {
        log.WithContext(c.Request.Context()).Info("处理请求")
        c.JSON(200, gin.H{"message": "Hello World"})
    })
    
    router.Run(":8080")
}
```

## 📖 按场景查找

### 🎯 我想记录日志
- [Log 模块文档](log.md) - 学习如何使用高性能日志系统
- [如何在Gin中使用日志](log.md#示例1在gin中使用)
- [结构化日志最佳实践](log.md#最佳实践)

### 🔍 我想追踪请求链路
- [Trace 模块文档](trace.md) - 学习分布式链路追踪
- [如何确保每个请求唯一TraceID](trace.md#traceid-使用说明)
- [如何在日志中自动包含TraceID](log.md#示例1在gin中使用)

### 🗄️ 我想连接数据库
- [MySQL 模块文档](mysql.md) - 学习使用 GORM 和连接池
- [Redis 模块文档](redis.md) - 学习使用 Redis 客户端
- [Milvus 模块文档](milvus.md) - 学习使用向量数据库

### 📊 我想监控服务
- [Metrics 模块文档](metrics.md) - 学习使用 Prometheus 监控
- [如何自定义业务指标](metrics.md#示例2自定义-counter)
- [常用监控查询](metrics.md#grafana-dashboard-查询)

### 🚦 我想做流量控制
- [Traffic 模块文档](traffic.md) - 学习限流和熔断
- [如何集成Sentinel](traffic.md#示例3自定义-controllersentinel-实现)
- [流控最佳实践](traffic.md#最佳实践)

### ❌ 我想统一错误处理
- [Errors 模块文档](errors.md) - 学习统一错误处理
- [如何定义业务错误码](errors.md#示例3自定义业务错误码)
- [错误响应格式](errors.md#示例4错误响应格式)

### 💳 我想接入支付
- [Pay 模块文档](pay.md) - 微信 JSAPI/Native、支付宝 APP/当面付、查单与回调

## 🎓 学习路径

### 入门级（必学）

1. [Log](log.md) - 日志是最基础的
2. [Trace](trace.md) - 了解链路追踪
3. [Errors](errors.md) - 统一错误处理

### 进阶级（推荐）

4. [Metrics](metrics.md) - 服务监控
5. [MySQL](mysql.md) 或 [Redis](redis.md) - 数据存储
6. [Middlewares](middlewares.md) - Gin 中间件

### 高级级（选学）

7. [Traffic](traffic.md) - 流量控制
8. [Milvus](milvus.md) - 向量数据库（AI场景）
9. [Pay](pay.md) - 支付对接（微信 / 支付宝）

## 💡 常见问题

### Q: 日志会不会影响性能？
A: 不会。我们使用异步日志，入队耗时 < 100ns，支持 >100万 QPS。详见 [Log 性能优化](log.md#性能优化)

### Q: TraceID 是怎么保证唯一的？
A: 由 OpenTelemetry SDK 自动生成，遵循 W3C Trace Context 标准。详见 [Trace 文档](trace.md#traceid-使用说明)

### Q: 需要配置 Jaeger 吗？
A: 不需要。默认只生成 TraceID 用于日志关联。如需可视化，可选配置 Jaeger。详见 [Trace 配置](trace.md#配置-exporterjaeger)

### Q: MySQL 连接池默认配置是什么？
A: MaxOpenConns=100, MaxIdleConns=10, ConnMaxLifetime=1h。详见 [MySQL 配置](mysql.md#默认配置说明)

### Q: 如何集成到现有项目？
A: 逐步集成，先加日志，再加追踪，最后加监控。详见 [最佳实践](#最佳实践)

## 🤝 贡献指南

欢迎贡献代码和文档！

1. Fork 项目
2. 创建功能分支
3. 提交 Pull Request
4. 等待代码审查

## 📮 获取帮助

- GitHub Issues: [提交问题](https://github.com/liukunxin/go-infra/issues)
- 文档问题: [编辑文档](https://github.com/liukunxin/go-infra/tree/main/docs)

## 🔗 相关链接

- [项目主页](https://github.com/liukunxin/go-infra)
- [变更日志](https://github.com/liukunxin/go-infra/releases)
- [示例项目](https://github.com/liukunxin/go-infra/tree/main/examples)

---

**最后更新时间**: 2026-04-22
**文档版本**: v1.1

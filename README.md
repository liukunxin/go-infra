# go-infra

> 一个高性能、生产就绪的 Go 语言基础设施 SDK 库

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.23-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 📖 简介

`go-infra` 是一个经过生产环境验证的 Go 语言基础设施库，提供了微服务开发中常用的功能模块，包括日志、追踪、监控、数据库、缓存等，帮助开发者快速构建高性能、可观测的应用。

### ✨ 核心特性

- 🚀 **高性能优化** - 对象池、连接池等优化，支持高并发场景
- 🔍 **完整的可观测性** - 集成日志、追踪、监控（基于 OpenTelemetry）
- 🛡️ **生产就绪** - 经过性能优化和并发安全验证
- 📦 **开箱即用** - 提供合理的默认配置
- 🔧 **灵活配置** - 支持自定义配置，满足不同场景需求
- 📝 **详细文档** - 每个模块都有完整的使用示例

## 📦 功能模块

| 模块 | 说明 | 文档 |
|-----|------|------|
| **errors** | 统一错误处理和HTTP状态码封装 | [查看文档](docs/errors.md) |
| **log** | 高性能异步日志库（支持链路追踪） | [查看文档](docs/log.md) |
| **trace** | 分布式链路追踪（基于OpenTelemetry） | [查看文档](docs/trace.md) |
| **metrics** | Prometheus监控指标采集 | [查看文档](docs/metrics.md) |
| **mysql** | MySQL/GORM客户端（连接池管理） | [查看文档](docs/mysql.md) |
| **redis** | Redis客户端（支持单机/集群） | [查看文档](docs/redis.md) |
| **milvus** | Milvus向量数据库客户端（连接池） | [查看文档](docs/milvus.md) |
| **traffic** | 流量控制（限流/熔断接口） | [查看文档](docs/traffic.md) |
| **http_client** | HTTP客户端（连接池复用） | [查看文档](docs/http_client.md) |
| **controller** | Gin基础控制器（统一响应格式） | [查看文档](docs/controller.md) |
| **middlewares** | Gin中间件（日志/追踪/CORS等） | [查看文档](docs/middlewares.md) |

## 🚀 快速开始

### 安装

```bash
go get github.com/liukunxin/go-infra
```

### 基础示例

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
    // 1. 初始化日志
    log.Init(log.Config{
        Level: "info",
    })

    // 2. 初始化链路追踪
    trace.Init(
        trace.WithServiceName("my-service"),
    )

    // 3. 创建 Gin 路由
    router := gin.Default()

    // 4. 初始化监控（自动注册 /metrics 路由）
    metrics.InitMetrics("my-service", router)

    // 5. 注册中间件
    router.Use(middlewares.GinTraceMiddleware())  // 链路追踪
    router.Use(middlewares.HttpLogRecord())       // 日志记录

    // 6. 定义路由
    router.GET("/hello", func(c *gin.Context) {
        log.WithContext(c.Request.Context()).Info("处理hello请求")
        c.JSON(200, gin.H{"message": "Hello World"})
    })

    // 7. 启动服务
    router.Run(":8080")
}
```

## 📚 详细文档

### 核心模块

- [错误处理 (errors)](docs/errors.md) - 统一的错误处理和状态码管理
- [日志系统 (log)](docs/log.md) - 高性能异步日志，支持链路追踪
- [链路追踪 (trace)](docs/trace.md) - 基于OpenTelemetry的分布式追踪
- [监控指标 (metrics)](docs/metrics.md) - Prometheus指标采集

### 数据存储

- [MySQL](docs/mysql.md) - GORM封装，连接池管理
- [Redis](docs/redis.md) - Redis客户端，支持单机/集群模式
- [Milvus](docs/milvus.md) - 向量数据库客户端，连接池管理

### 工具模块

- [HTTP客户端](docs/http_client.md) - 带连接池的HTTP客户端
- [流量控制 (traffic)](docs/traffic.md) - 限流/熔断接口
- [中间件 (middlewares)](docs/middlewares.md) - Gin常用中间件

## 🏗️ 架构设计

### 可观测性架构

```
┌─────────────────────────────────────────────────────────┐
│                      应用程序                             │
├─────────────────────────────────────────────────────────┤
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │   Log    │  │  Trace   │  │ Metrics  │              │
│  │ (日志)    │  │ (追踪)    │  │ (监控)    │              │
│  └─────┬────┘  └─────┬────┘  └─────┬────┘              │
│        │             │              │                   │
│        └─────────────┴──────────────┘                   │
│                      │                                  │
│              OpenTelemetry SDK                          │
├─────────────────────────────────────────────────────────┤
│        Exporter (Jaeger/Prometheus/etc)                │
└─────────────────────────────────────────────────────────┘
```

### 连接池管理

```
应用程序
  ├─ MySQL连接池 (GORM)
  │   └─ 默认配置: MaxOpen=100, MaxIdle=10
  ├─ Redis连接池
  │   └─ 支持单机/集群模式
  ├─ HTTP连接池
  │   └─ 支持连接复用和超时控制
  └─ Milvus连接池
      └─ 自定义连接池实现
```

## 🔥 性能优化

本库在以下方面进行了性能优化：

- ✅ **对象池** - 日志、HTTP等模块使用sync.Pool减少GC压力
- ✅ **连接池** - 数据库、Redis、HTTP连接池复用
- ✅ **异步日志** - 基于环形队列的无锁日志系统
- ✅ **并发安全** - 所有模块都经过并发安全验证
- ✅ **零拷贝** - 减少不必要的内存分配和复制

## 💡 最佳实践

### 初始化顺序

建议按以下顺序初始化各模块：

```go
1. 日志 (log)          - 最先初始化，其他模块可能依赖
2. 追踪 (trace)        - 尽早初始化，用于记录启动过程
3. 数据库 (mysql/redis) - 建立数据库连接
4. 路由 (gin)          - 创建HTTP服务
5. 监控 (metrics)      - 注册监控端点
6. 中间件              - 注册全局中间件
```

### 错误处理

```go
import kerr "github.com/liukunxin/go-infra/pkg/errors"

// 业务逻辑中
if err != nil {
    return kerr.WarpError(kerr.StatusBadRequest, 40001, err)
}

// 在Controller中
func (b *GinBase) MyHandler(c *gin.Context) {
    if err := doSomething(); err != nil {
        b.ErrorResponse(c, err)  // 自动处理错误响应
        return
    }
    b.SuccessResponse(c, data)
}
```

### 日志记录

```go
// 带上下文的日志（自动包含TraceID）
log.WithContext(ctx).Info("用户登录成功")

// 带字段的日志
log.WithContext(ctx).WithFields(map[string]interface{}{
    "user_id": 123,
    "action": "login",
}).Info("用户操作")
```

## 📊 监控指标

自动采集的指标：

- `http_requests_total` - HTTP请求总数（按method、path、status分组）
- `http_request_duration_ms` - HTTP请求延迟（直方图）

可通过 `/metrics` 端点访问。

## 🤝 贡献

欢迎提交Issue和Pull Request！

### 开发规范

- 所有公开API需要添加注释
- 新增功能需要提供使用示例
- 性能敏感模块需要进行基准测试
- 确保并发安全

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

## 🔗 相关链接

- [OpenTelemetry](https://opentelemetry.io/)
- [Gin Web Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [Prometheus](https://prometheus.io/)

## 📮 联系方式

- GitHub: [liukunxin/go-infra](https://github.com/liukunxin/go-infra)
- Issues: [提交问题](https://github.com/liukunxin/go-infra/issues)

---

**⭐ 如果这个项目对你有帮助，请给个Star！**

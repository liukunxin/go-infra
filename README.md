# go-infra

> 一个高性能、生产就绪的 Go 语言基础设施 SDK 库

[![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.25-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 📖 简介

`go-infra` 是一个经过生产环境验证的 Go 语言基础设施库，提供了微服务开发中常用的功能模块，包括日志、追踪、监控、数据库、缓存等，帮助开发者快速构建高性能、可观测的应用。

### ✨ 核心特性

- 🚀 **高性能优化** - 对象池、连接池等优化，支持高并发场景
- 🔍 **完整的可观测性** - 集成日志、追踪、监控（基于 OpenTelemetry）
- 🛡️ **生产就绪** - 经过性能优化和并发安全验证
- 📦 **开箱即用** - 提供合理的默认配置
- 🔧 **灵活配置** - 支持自定义配置，满足不同场景需求
- 🤖 **AI 统一接入** - 提供 LLM 多厂商统一调用与主备切换能力
- 📝 **详细文档** - 每个模块都有完整的使用示例

## 📦 功能模块

| 层级 | 模块 | 说明 | 文档 |
|------|------|------|------|
| **base** | **log** | 高性能异步日志（支持链路追踪） | [查看文档](pkg/base/log/README.md) |
| **base** | **trace** | 分布式链路追踪（OpenTelemetry） | [查看文档](pkg/base/trace/README.md) |
| **base** | **config** | 通用配置加载（YAML + 环境覆盖 + 加密） | [查看文档](pkg/base/config/README.md) |
| **base** | **errors** | 统一错误处理和 HTTP 状态码封装 | [查看文档](pkg/base/errors/README.md) |
| **base** | **env** | 环境变量与模式管理 | `pkg/base/env/` |
| **base** | **uuid** | Snowflake / UUID ID 生成 | `pkg/base/uuid/` |
| **base** | **datetime** | API JSON 时间类型（`2006-01-02 15:04:05`） | `pkg/base/datetime/` |
| **base** | **xutil** | 通用泛型工具函数（含日/周/月区间计算） | `pkg/base/xutil/` |
| **infra** | **mysql** | MySQL/GORM 客户端（连接池管理） | [查看文档](pkg/infra/mysql/README.md) |
| **infra** | **redis** | Redis 客户端（单机/集群） | [查看文档](pkg/infra/redis/README.md) |
| **infra** | **milvus** | Milvus 向量数据库客户端（连接池） | [查看文档](pkg/infra/milvus/README.md) |
| **infra** | **metrics** | Prometheus 监控指标采集 | [查看文档](pkg/infra/metrics/README.md) |
| **infra** | **traffic** | 流量控制（限流/熔断接口） | [查看文档](pkg/infra/traffic/README.md) |
| **infra** | **http_client** | HTTP 客户端（连接池复用） | [查看文档](pkg/infra/http_client/README.md) |
| **infra** | **grpc** | gRPC 客户端/服务端统一封装（拦截器/治理） | [查看文档](pkg/infra/grpc/README.md) |
| **infra** | **websocket** | WebSocket 长连接封装（心跳/重连/Hub） | [查看文档](pkg/infra/websocket/README.md) |
| **infra** | **llm** | 大模型统一调用 SDK（多厂商协议抽象） | [查看文档](pkg/infra/llm/README.md) |
| **infra** | **apollo** | Apollo 配置中心 | `pkg/infra/apollo/` |
| **infra** | **objstore** | S3 兼容对象存储（KS3/OSS/OBS/MinIO 等） | [查看文档](pkg/infra/objstore/README.md) |
| **biz** | **login** | 多方式登录（密码/手机/邮箱/微信）+ JWT | [查看文档](pkg/biz/login/README.md) |
| **biz** | **account** | 账号管理与多登录方式绑定 | [查看文档](pkg/biz/account/README.md) |
| **biz** | **pay** | 微信支付 APIv3 / 支付宝 RSA2 | [查看文档](pkg/biz/pay/README.md) |
| **biz** | **collab** | 跨端实时协作引擎（定序/去重/回放/订阅） | [查看文档](pkg/biz/collab/README.md) |
| **biz** | **controller** | Gin 基础控制器（统一响应格式） | `pkg/biz/controller/` |
| **biz** | **middlewares** | Gin 中间件（日志/追踪/CORS 等） | `pkg/biz/middlewares/` |

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
    "github.com/liukunxin/go-infra/pkg/base/log"
    "github.com/liukunxin/go-infra/pkg/base/trace"
    "github.com/liukunxin/go-infra/pkg/infra/metrics"
    "github.com/liukunxin/go-infra/pkg/biz/middlewares"
)

func main() {
    // 1. 初始化日志
    log.Init(log.Config{
        Level: log.LevelInfo,
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

### pkg/base — 原子基础能力

- [错误处理 (errors)](pkg/base/errors/README.md) - 统一的错误处理和状态码管理
- [日志系统 (log)](pkg/base/log/README.md) - 高性能异步日志，支持链路追踪
- [链路追踪 (trace)](pkg/base/trace/README.md) - 基于 OpenTelemetry 的分布式追踪

### pkg/infra — 基础设施能力

- [MySQL](pkg/infra/mysql/README.md) - GORM 封装，连接池管理
- [Redis](pkg/infra/redis/README.md) - Redis 客户端，支持单机/集群模式
- [Milvus](pkg/infra/milvus/README.md) - 向量数据库客户端，连接池管理
- [监控指标 (metrics)](pkg/infra/metrics/README.md) - Prometheus 指标采集
- [HTTP 客户端](pkg/infra/http_client/README.md) - 带连接池的 HTTP 客户端
- [gRPC](pkg/infra/grpc/README.md) - gRPC 客户端/服务端统一封装
- [WebSocket](pkg/infra/websocket/README.md) - WebSocket 长连接封装
- [流量控制 (traffic)](pkg/infra/traffic/README.md) - 限流/熔断接口
- [LLM SDK](pkg/infra/llm/README.md) - 大模型统一调用（支持 OpenAI 兼容协议）

### pkg/biz — 业务基础能力

- [实时协作 (collab)](pkg/biz/collab/README.md) - 跨端实时协作引擎（全局定序/幂等去重/历史回放/实时订阅）
- [登录 (login)](pkg/biz/login/README.md) - 多方式登录 + JWT
- [账号 (account)](pkg/biz/account/README.md) - 账号管理与登录绑定
- [支付 (pay)](pkg/biz/pay/README.md) - 微信 / 支付宝支付封装

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

## 🔐 配置加密

支持对敏感配置值（数据库密码、API Key 等）进行 AES-256-GCM 加密存储，运行时自动解密：

```yaml
# configs/config.prod.yml
mysql:
  host: 10.0.1.100
  password: "ENC(nonce+ciphertext的base64)"
redis:
  password: "ENC(...)"
```

```go
cfg := config.MustLoad[App](
    config.WithDecrypt(config.AESKeyFromEnv("CONFIG_ENCRYPT_KEY")),
)
```

- 不传 `WithDecrypt` 时行为完全不变，零侵入
- 密钥通过环境变量注入，推荐 K8s Secret 管理
- 使用 `go-infra-cli keygen / encrypt / decrypt` 命令生成密钥和加密值
- 详见 [配置模块文档](pkg/base/config/README.md)

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
import kerr "github.com/liukunxin/go-infra/pkg/base/errors"

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

### 新增包后的同步 Checklist

新增 `pkg/infra/xxx` 或 `pkg/biz/xxx` 包后，需同步更新 scaffold（否则 AI 辅助编码时不会感知到新包）：

- [ ] `scaffold/skills/go-infra-reference/SKILL.md` — 包地图表格新增一行
- [ ] `scaffold/skills/go-infra-reference/reference.md` — 补充典型用法代码示例
- [ ] `scaffold/single-starter/.cursor/rules/11-go-infra-api.mdc` — 如该包属于默认初始化能力，同步更新
- [ ] `scaffold/monorepo-starter/.cursor/rules/11-go-infra-api.mdc` — 同上

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

# HTTP Client - 连接池封装

基于标准库 `net/http` 的 `Transport` 连接池封装，提供带超时与可扩展 `RequestOption` 的请求方法，并支持取出底层 `*http.Client` 注入其它模块（如 Pay / LLM）。

## 功能特性

- ✅ 可配置超时、连接池、TLS 握手、响应体大小上限
- ✅ 客户端默认 Header + 单次请求 `WithHeader` / `WithHeaders`
- ✅ 自动把 ctx 中的 TraceID 写入 `X-Request-ID`（有则写，无则跳过；用户显式设置优先）
- ✅ 可选进程级单例：`Init` + `GetHTTPClient`
- ✅ `HTTPClient()` / `Transport()` 暴露标准库能力，便于注入与长连接复用连接池
- ✅ `DoRequest` 支持自定义 Method 与 `io.Reader` body，避免多余拷贝

## 快速开始

### 直接使用 NewClient

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/liukunxin/go-infra/pkg/infra/http_client"
)

func main() {
    c := http_client.NewClient(http_client.Config{
        Timeout: 10 * time.Second,
        DefaultHeaders: map[string]string{
            "User-Agent": "my-service/1.0",
        },
    })

    body, status, err := c.Get(context.Background(), "https://example.com/",
        http_client.WithHeader("Accept", "application/json"),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(status, len(body))
}
```

### 自定义 Header

```go
body, status, err := c.Post(ctx, url, payload,
    http_client.WithJSON(),
    http_client.WithHeader("Authorization", "Bearer "+token),
)
// ctx 带 TraceID 时会自动设置 X-Request-ID；无 TraceID 则不设置
```

用户自定义 `X-Request-ID` 优先，不会被 TraceID 覆盖：

```go
c.Get(ctx, url,
    http_client.WithHeader(http_client.HeaderRequestID, "my-explicit-id"),
)
```

### 进程内单例

```go
http_client.Init(http_client.Config{Timeout: 15 * time.Second})
cli := http_client.GetHTTPClient()
_, _, _ = cli.Post(ctx, url, []byte(`{}`), http_client.WithJSON())
```

### 与 Pay / LLM 共用连接池

```go
shared := http_client.NewClient(http_client.Config{Timeout: 30 * time.Second})
// wechat.NewClient(wechat.Config{ ..., HTTPClient: shared.HTTPClient() })
```

注意：注入出去的裸 `*http.Client` **不会**自动应用本包的 `DefaultHeaders` / `X-Request-ID` 自动绑定 / `RequestOption`；这些能力只在 `Client` 的 `Get`/`Post`/`Do`/`DoRequest` 路径生效。

### 共用连接池但使用不同超时（如 SSE / WebSocket）

```go
http_client.Init(http_client.Config{Timeout: 30 * time.Second})

transport := http_client.GetTransport() // nil-safe，未 Init 时返回 nil
if transport == nil {
    transport = http.DefaultTransport
}

shortCl := &http.Client{Transport: transport, Timeout: 10 * time.Second}
streamCl := &http.Client{Transport: transport, Timeout: 0} // 流式长连接
```

## 配置说明

| 字段 | 说明 |
|------|------|
| `Timeout` | 单次请求总超时（默认 30s） |
| `MaxIdleConns` | 最大空闲连接数（默认 100） |
| `MaxIdleConnsPerHost` | 每个 host 最大空闲连接（默认 10） |
| `MaxConnsPerHost` | 每个 host 最大连接数（0 = 不限制） |
| `IdleConnTimeout` | 空闲连接超时（默认 90s） |
| `TLSHandshakeTimeout` | TLS 握手超时（默认 10s） |
| `MaxResponseBodyBytes` | 响应体读取上限（默认 32MB；超出返回 `ErrResponseBodyTooLarge`） |
| `DefaultHeaders` | 每个请求默认 Header；单次 `RequestOption` 可覆盖 |

## RequestOption

| Option | 说明 |
|--------|------|
| `WithHeader(k, v)` | 设置单个 Header |
| `WithHeaders(map)` | 批量设置 Header |
| `WithContentType(ct)` | 设置 Content-Type |
| `WithJSON()` | `Content-Type: application/json` |

Header 应用顺序：`DefaultHeaders` → 请求 Option → 自动 `X-Request-ID`（仅当该头仍为空且 ctx 有 TraceID）。

## API 摘要

- `NewClient(cfg) *Client`
- `Init(cfg)` / `GetHTTPClient() *Client` / `GetTransport() http.RoundTripper`
- `Get` / `Head` / `Post` / `Put` / `Patch` / `Delete` / `DoRequest` / `Do`
- `HTTPClient()` / `Transport()` / `CloseIdleConnections()`

## 相关代码

- `pkg/infra/http_client/client.go`
- `pkg/infra/http_client/option.go`
- `pkg/infra/http_client/init.go`

# HTTP Client - 连接池封装

基于标准库 `net/http` 的 `Transport` 连接池封装，提供带超时的 `Get` / `Post`，并支持取出底层 `*http.Client` 注入其它模块（如 [Pay](pay.md)）。

## 功能特性

- ✅ 可配置超时、每 Host 连接数、空闲连接与 TLS 握手超时
- ✅ 可选进程级单例：`Init` + `GetHttpClient`
- ✅ `HTTPClient()` 暴露标准库客户端，便于 `pkg/pay` 等复用连接池

## 快速开始

### 直接使用 NewClient

```go
package main

import (
    "fmt"
    "time"

    "github.com/liukunxin/go-infra/pkg/http_client"
)

func main() {
    c := http_client.NewClient(http_client.Config{
        Timeout:             10 * time.Second,
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    })
    body, status, err := c.Get("https://example.com/", nil)
    if err != nil {
        panic(err)
    }
    fmt.Println(status, len(body))
}
```

### 进程内单例

```go
http_client.Init(http_client.Config{Timeout: 15 * time.Second})
cli := http_client.GetHttpClient()
_, _, _ = cli.Post(url, []byte(`{}`), map[string]string{"Content-Type": "application/json"})
```

### 与 Pay 模块共用连接池

```go
shared := http_client.NewClient(http_client.Config{Timeout: 30 * time.Second})
// wechat.NewClient(wechat.Config{ ..., HTTPClient: shared.HTTPClient() })
```

详见 [Pay 文档 - 可选注入 *http.Client](pay.md#可选注入-httpclient)。

### 共用连接池但使用不同超时（如 SSE / WebSocket）

全局 Client 携带固定的请求超时（`Config.Timeout`），长连接场景需要 `Timeout: 0`。
通过 `Transport()` / `GetTransport()` 取出连接池，自行组装 `*http.Client`：

```go
// 初始化阶段（进程启动时）
http_client.Init(http_client.Config{Timeout: 30 * time.Second})

// 使用阶段（可在任意位置调用）
transport := http_client.GetTransport()   // nil-safe，不 panic
if transport == nil {
    transport = &http.Transport{...}      // 未初始化时的 fallback
}

shortCl := &http.Client{Transport: transport, Timeout: 10 * time.Second}
streamCl := &http.Client{Transport: transport, Timeout: 0}  // SSE / 流式长连接
```

两个客户端共享同一套 TCP 连接池（`http.Transport`），仅 timeout 策略不同。

## 配置说明

| 字段 | 说明 |
|------|------|
| `Timeout` | 单次请求总超时 |
| `MaxIdleConns` | 最大空闲连接数 |
| `MaxIdleConnsPerHost` | 每个 host 最大空闲连接 |
| `MaxConnsPerHost` | 每个 host 最大连接数 |
| `IdleConnTimeout` | 空闲连接超时 |
| `TLSHandshakeTimeout` | TLS 握手超时（为 0 时默认 10s） |

## API 摘要

- `NewClient(cfg Config) *Client`：创建独立客户端实例。
- `Init(cfg)` / `GetHttpClient() *Client`：单例模式（未 `Init` 时 `GetHttpClient` 会 panic）。
- `GetTransport() http.RoundTripper`：nil-safe 取出共享连接池，未初始化时返回 nil。
- `(*Client) Get(url, headers) ([]byte, int, error)`
- `(*Client) Post(url, body, headers) ([]byte, int, error)`
- `(*Client) HTTPClient() *http.Client`：返回底层客户端，供需标准 `*http.Client` 的库注入使用。
- `(*Client) Transport() http.RoundTripper`：返回底层连接池，供需自定义 timeout 的长连接场景使用。

## 相关代码

- `pkg/http_client/client.go`
- `pkg/http_client/init.go`

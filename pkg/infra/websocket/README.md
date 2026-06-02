# WebSocket - 统一长连接封装

`pkg/infra/websocket` 提供 go-infra 的 WebSocket 能力封装：

- Server Upgrade + 生命周期管理（读写协程、优雅关闭）
- Client 自动重连（指数退避 + jitter）
- Ping/Pong 保活 + deadline 控制
- Hub（连接管理、分组广播）
- OTel 指标（连接数、消息量、重连、错误）

底层依赖使用维护活跃的 `github.com/coder/websocket`。

## 1. 服务端接入（最小示例）

```go
hub := websocket.NewHub()

wsServer := websocket.NewServer(
	websocket.Config{
		PingInterval:    20 * time.Second,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    10 * time.Second,
		SendQueueSize:   256,
		MaxMessageBytes: 4 << 20,
	},
	websocket.Handlers{
		OnConnect: func(c *websocket.Conn) {
			hub.Register(c)
		},
		OnMessage: func(c *websocket.Conn, m websocket.Message) {
			// 回显
			_ = c.Send(m.Type, m.Data)
		},
		OnClose: func(c *websocket.Conn, _ error) {
			hub.Unregister(c)
		},
	},
	func(r *http.Request) error {
		// 可接入 JWT 校验
		return nil
	},
	func(r *http.Request) bool {
		// 可按域名做 Origin 校验
		return true
	},
)

http.Handle("/ws", wsServer)
```

## 2. 客户端接入（最小示例）

```go
client := websocket.NewClient(
	websocket.ClientConfig{
		URL:                  "ws://127.0.0.1:8080/ws",
		Reconnect:            true,
		ReconnectBaseBackoff: 200 * time.Millisecond,
		ReconnectMaxBackoff:  5 * time.Second,
		BackoffJitterRatio:   0.2,
		Config: websocket.Config{
			PingInterval: 20 * time.Second,
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
	},
	websocket.ClientHandlers{
		OnConnect: func() {},
		OnMessage: func(m websocket.Message) {},
		OnError:   func(err error) {},
	},
)

ctx := context.Background()
go func() { _ = client.Run(ctx) }()
_ = client.SendText("hello")
```

## 3. Hub 广播

```go
hub.Broadcast(websocket.MessageText, []byte("global"))
hub.BroadcastGroup("room-a", websocket.MessageText, []byte("to room-a"))
```

## 4. 设计原则

- 默认值可直接用，业务只需要填 URL / 路由。
- 只保留必要开关：重连、压缩、背压策略。
- 连接生命周期统一由 SDK 管理，业务关注消息处理即可。

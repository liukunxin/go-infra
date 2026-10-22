# WebSocket — 统一长连接封装

`pkg/infra/websocket` 提供 go-infra 的 WebSocket **传输层**能力：

- Server Upgrade + 读写循环 + Ping/Pong
- 出站发送队列与明确的背压策略
- `OnTick` 周期钩子（业务续租约 / 打点，不进 SDK）
- 连接级元数据 `Set` / `Get`
- Client 自动重连（指数退避 + jitter）
- 可选进程内 Hub（分组广播，**非阻塞** fan-out）
- `Server.Shutdown`：拒新连接 + 尽力冲刷出站队列后再关
- OTel 指标（连接数、消息量、重连、错误）

底层依赖：`github.com/coder/websocket`。

## 边界（必读）

| SDK 负责 | 业务负责（不要塞进本包） |
|----------|--------------------------|
| Upgrade / Accept / Dial | 业务帧协议（msg_id、压缩位、业务 ACK） |
| 读写泵、Ping、发送队列 | Redis 租约 / Session / 房间权威态 |
| `OnTick` 回调时机 | 在 `OnTick` 里续租、触活 |
| `Conn` 不透明元数据 | user / task / role / epoch 等语义 |
| 进程内 Hub 广播 | 跨实例一致性（应落 Redis 等） |
| `Shutdown` 本机连接收口 | 产品级重连协议（携带游标等） |

原则：**本包只做「连接生命周期 + 可靠字节进出」**。听记 AUDIO_BLOCK、权益鉴权细节、ASR 桥等一律留在业务仓库。

## 1. 服务端（推荐：中间件鉴权）

鉴权已在 HTTP 中间件完成时，不要再塞 `HandshakeAuth`——Upgrade 前 identity 已就绪。

```go
hub := websocket.NewHub()

wsServer := websocket.NewServer(
	websocket.Config{
		PingInterval:  20 * time.Second,
		// ReadTimeout 默认 0：不因「长时间无业务消息」误杀仍能 ping 的连接。
		// 存活靠 PingInterval；仅在需要限制单次读耗时时再设正值。
		WriteTimeout:  10 * time.Second,
		SendQueueSize: 256,
		Backpressure:  websocket.BackpressureError,
	},
	websocket.Handlers{
		OnConnect: func(c *websocket.Conn) {
			hub.Register(c)
			c.Set("user_id", userID)
		},
		OnMessage: func(c *websocket.Conn, m websocket.Message) {
			_ = c.Send(m.Type, m.Data)
		},
		OnTick: func(c *websocket.Conn) {
			// 与 PingInterval 同频：续租约 / Touch Session；失败可 c.Close()。
		},
		OnClose: func(c *websocket.Conn, _ error) {
			hub.Unregister(c)
		},
	},
	nil,
	func(r *http.Request) bool { return true },
)

mux.Handle("/ws", authMiddleware(wsServer))

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = wsServer.Shutdown(ctx)
```

### Shutdown 语义

1. 置位拒绝新 Upgrade（与 `track` 同锁，避免漏关连接）  
2. 对存量连接 `beginDrain`：拒绝新 `Send`，写循环冲刷已入队帧后关 socket  
3. `ctx` 超时则对仍存活连接 **强制** `Close`（未写出的队列帧丢弃）

因此是 **尽力冲刷**，不是「保证每条出站消息都送达对端」。

### 备选：握手内鉴权

```go
auth := func(r *http.Request) error {
	if r.Header.Get("Authorization") == "" {
		return errors.New("unauthorized")
	}
	return nil
}
wsServer := websocket.NewServer(cfg, handlers, auth, checkOrigin)
```

## 2. 背压策略

| 策略 | 行为 |
|------|------|
| `BackpressureBlock`（默认） | 点对点 `Send` 阻塞直到有空位或连接关闭 |
| `BackpressureError` | 队列满立即返回 `ErrSendQueueFull`，**不丢该次调用的消息** |

`Hub.Broadcast` / `BroadcastGroup` **固定非阻塞**（`TrySend`），返回失败连接数；慢连接不会拖死整网广播。这与点对点 `Send` 的 `Backpressure` 配置无关。

兼容旧字段：`drop_on_backpressure: true` → `BackpressureError`。

```go
if err := c.Send(websocket.MessageBinary, frame); errors.Is(err, websocket.ErrSendQueueFull) {
	// 推送应用层背压信令，或减速读入
}
```

## 3. 客户端

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
			WriteTimeout: 10 * time.Second,
		},
	},
	websocket.ClientHandlers{
		OnConnect: func() {},
		OnMessage: func(m websocket.Message) {},
		OnTick:    func() {},
		OnError:   func(err error) {},
	},
)

ctx := context.Background()
go func() { _ = client.Run(ctx) }()
_ = client.SendText("hello")
```

## 4. Hub

适合**单进程**内广播。多副本权威态请用外部存储。

```go
failed := hub.Broadcast(websocket.MessageText, []byte("global"))
_ = hub.BroadcastGroup("room-a", websocket.MessageText, []byte("to room-a"))
_ = hub.Len()
```

## 5. 设计原则

- 默认值可直接用；业务只填超时 / 队列 / 背压策略。
- 生命周期由 SDK 管理；业务关注消息与 `OnTick`。
- 不为单一产品定制帧格式；保持包可被任意二进制/文本协议复用。

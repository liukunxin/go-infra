# collab — 跨端实时协作引擎

`pkg/biz/collab` 是 go-infra 提供的通用实时协作 SDK，为业务服务提供"多端看到同一份数据实时变化 + 断线恢复不丢数据"的能力。

## 核心能力

| 能力 | 实现方式 |
|------|---------|
| 全局单调递增定序 | Redis INCR + Lua 原子脚本 |
| 事件持久化 | Redis Stream（MAXLEN 自动裁剪） |
| 幂等去重 | Redis SET NX（可配窗口期） |
| 历史回放 | 支持从任意 seq 续读 |
| 自动快照 | 加速回放，业务自定义聚合逻辑 |
| 实时订阅 | Redis XREAD BLOCK 长轮询驱动回调 |

## 设计理念

SDK 管理 5 个元数据字段（`EventID`/`SessionID`/`Seq`/`Timestamp`/`SenderID`），业务方的所有数据放在 `Body`（`json.RawMessage`）中自行定义结构。引擎对 Body 做**纯透传**——写入时原样存储，回放/订阅时原样返回。

`SenderID` 是多端协作的通用字段，由业务方定义粒度——可以是用户 ID、设备 ID 或连接 ID，SDK 不限制。引擎用它做日志记录，未来可扩展推送过滤（不推给发送者自己）。

这意味着：
- 不同业务可以定义完全不同的 Body 结构
- 业务升级 Body 结构不需要改动 SDK 版本
- SDK 永远不会"多了不需要的字段"或"少了想要的字段"

## 快速开始

```go
import (
    "encoding/json"
    "github.com/liukunxin/go-infra/pkg/biz/collab"
    redisv8 "github.com/liukunxin/go-infra/pkg/infra/redis/v8"
)

// 使用 go-infra 全局 Redis 客户端创建引擎
engine := collab.New(redisv8.GetClient(),
    collab.WithNamespace("my-biz"),
    collab.WithSnapEvery(200),
    collab.WithSnapshotBuilder(func(events []collab.Envelope) (json.RawMessage, error) {
        // 业务自定义：解析 Body 并聚合为快照
        return json.Marshal(map[string]any{"aggregated": true})
    }),
)

// 1. 创建会话
sess, err := engine.CreateSession(ctx, "session-uuid-123")

// 2. 构造业务 Body（SDK 不关心内容）
body, _ := json.Marshal(map[string]any{
    "event_type": "write",
    "sender_id":  "mobile-user-1",
    "payload":    map[string]any{"sheet": "Sheet1", "items": items},
})

// 3. 写入事件（自动定序、去重、持久化）
result, err := engine.Append(ctx, collab.Envelope{
    SessionID: sess.ID,
    SenderID:  "mobile-user-1",
    Body:      body,
})
fmt.Println("seq:", result.Seq)

// 4. 订阅实时推送（在 goroutine 中）
go engine.Subscribe(ctx, sess.ID, func(evt collab.Envelope) {
    fmt.Printf("received: seq=%d body=%s\n", evt.Seq, string(evt.Body))
})

// 5. 断线重连回放（从 seq=100 之后的增量）
replay, err := engine.Replay(ctx, sess.ID, 100)

// 6. 关闭会话（10 分钟后 Redis 自动清理）
engine.CloseSession(ctx, sess.ID, 10*time.Minute)
```

## 业务 Body 示例

SDK 不限制 Body 内容，不同业务可以定义完全不同的结构：

**SnapSheet 业务**：
```json
{
  "event_type": "write",
  "sender_id": "mobile-user-1",
  "payload": { "sheet": "Sheet1", "items": [...] }
}
```

**语音速记业务**：
```json
{
  "event_type": "asr.partial",
  "sender_id": "pc-mic",
  "payload": { "text": "今天的会议...", "is_final": false }
}
```

## 配置项

| Option | 默认值 | 说明 |
|--------|--------|------|
| `WithNamespace(ns)` | `"collab"` | Redis key 前缀，不同业务使用不同 namespace 隔离 |
| `WithStreamMaxLen(n)` | `50000` | Stream 最大条目数（MAXLEN ~） |
| `WithSnapEvery(n)` | `500` | 每 N 条事件自动构建快照，0 禁用 |
| `WithDedupTTL(d)` | `30min` | 去重窗口时长 |
| `WithBlockTimeout(d)` | `5s` | Subscribe 的 XREAD BLOCK 超时 |
| `WithSnapshotBuilder(fn)` | `nil` | 快照构建函数，不提供则不做快照 |

## Redis 存储设计

所有 key 使用 `{sid}` hash tag 确保同一 session 的数据落在同一 Redis slot（集群友好）：

| 用途 | Key 模式 | Redis 类型 |
|------|---------|-----------|
| 事件流 | `{ns}:sess:{sid}:events` | Stream |
| 序号计数器 | `{ns}:sess:{sid}:seq` | String(int) |
| 快照 | `{ns}:sess:{sid}:snapshot` | String(JSON) |
| 会话元数据 | `{ns}:sess:{sid}:meta` | Hash |
| 去重标记 | `{ns}:dedup:{event_id}` | String |

## API 说明

### `New(rdb, opts...) *Engine`

创建引擎实例。Engine 线程安全，所有状态存储在 Redis 中，可多实例部署。

### `Engine.CreateSession(ctx, id) (Session, error)`

创建新会话。相同 id 重复创建返回 `ErrSessionExists`。

### `Engine.Append(ctx, evt) (Envelope, error)`

写入事件。流程：参数校验 → 自动填充（EventID/Timestamp）→ 去重 → 原子定序+持久化 → 异步快照检查。返回的 Envelope 中 `Seq` 已填充。Body 原样透传存储。

### `Engine.Replay(ctx, sessionID, fromSeq) (ReplayResult, error)`

回放历史。`fromSeq=0` 全量回放（优先加载快照加速）；`fromSeq>0` 增量回放。Body 原样返回。

### `Engine.Subscribe(ctx, sessionID, handler) error`

实时订阅。阻塞式，应在 goroutine 中调用，ctx 取消时退出。

### `Engine.CloseSession(ctx, sessionID, ttl) error`

关闭会话并设置 TTL，到期后 Redis 自动清理所有相关 key。

## 错误码

| 错误 | 含义 |
|------|------|
| `ErrDuplicate` | 事件重复（幂等拦截） |
| `ErrSessionClosed` | 会话已关闭 |
| `ErrSessionNotFound` | 会话不存在 |
| `ErrSessionExists` | 会话已存在 |
| `ErrEmptySessionID` | SessionID 为空 |

## 设计原则

1. **Body 透传**：SDK 对 `Envelope.Body` 不解析、不校验、不修改，序列化时原样嵌入 JSON
2. **零业务侵入**：引擎不 import 任何业务包，不知道业务 Body 里有什么
3. **线程安全**：Engine 无内部可变状态，所有状态在 Redis
4. **集群友好**：hash tag 保证同一 session 的操作在同一 slot
5. **向后兼容**：业务升级 Body 结构无需改动 SDK 版本
6. **最小依赖**：只依赖 `go-redis/v8` + `encoding/json` + 标准库 + `pkg/base/log`

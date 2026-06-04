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

## 快速开始

```go
import (
    "github.com/liukunxin/go-infra/pkg/biz/collab"
    redisv8 "github.com/liukunxin/go-infra/pkg/infra/redis/v8"
)

// 使用 go-infra 全局 Redis 客户端创建引擎
engine := collab.New(redisv8.GetClient(),
    collab.WithNamespace("my-biz"),   // 自定义 key 前缀
    collab.WithSnapEvery(500),        // 每 500 条事件自动快照
    collab.WithSnapshotBuilder(func(events []collab.Envelope) map[string]any {
        // 业务自定义：如何将事件列表折叠为快照
        return map[string]any{"state": "aggregated"}
    }),
)

// 1. 创建会话
sess, err := engine.CreateSession(ctx, "session-uuid-123")

// 2. 写入事件（自动定序、去重、持久化）
result, err := engine.Append(ctx, collab.Envelope{
    SessionID: sess.ID,
    EventType: "excel.rows.write",
    SenderID:  "mobile-user-1",
    Payload:   map[string]any{"rows": data},
})
fmt.Println("seq:", result.Seq)

// 3. 订阅实时推送（在 goroutine 中）
go engine.Subscribe(ctx, sess.ID, func(evt collab.Envelope) {
    // 推送给其他在线端
    websocket.Broadcast(evt)
})

// 4. 断线重连回放（从 seq=100 之后的增量）
replay, err := engine.Replay(ctx, sess.ID, 100)
// replay.Events = seq>100 的事件列表
// replay.Snapshot = 快照（如果有且有用）
// replay.LastSeq = 当前最大 seq

// 5. 关闭会话（10 分钟后 Redis 自动清理）
engine.CloseSession(ctx, sess.ID, 10*time.Minute)
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

写入事件。流程：参数校验 → 自动填充（EventID/Timestamp）→ 去重 → 原子定序+持久化 → 异步快照检查。返回的 Envelope 中 `Seq` 已填充。

### `Engine.Replay(ctx, sessionID, fromSeq) (ReplayResult, error)`

回放历史。`fromSeq=0` 全量回放（优先加载快照加速）；`fromSeq>0` 增量回放。

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

1. **零业务侵入**：引擎不 import 任何业务包，Payload 对引擎透明
2. **线程安全**：Engine 无内部可变状态，所有状态在 Redis
3. **集群友好**：hash tag 保证同一 session 的操作在同一 slot
4. **优雅降级**：SnapshotBuilder 可选，不提供时仅保留事件流
5. **最小依赖**：只依赖 `go-redis/v8` + 标准库 + `pkg/base/log`

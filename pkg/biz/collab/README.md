# collab — 跨端实时协作引擎

`pkg/biz/collab` 是 go-infra 提供的通用实时协作 SDK，为业务服务提供"多端看到同一份数据实时变化 + 断线恢复不丢数据"的能力。

## 适用场景

- 多人/多端实时编辑同一份数据（表格、文档、白板）
- 跨设备状态同步（手机采集 → PC 处理 → 两端实时可见）
- 需要断线恢复、事件重放的有序消息流

## 核心能力

| 能力 | 实现方式 | 说明 |
|------|---------|------|
| 全局单调递增定序 | Redis INCR + Lua 原子脚本 | 保证事件严格有序，无 seq 空洞 |
| 事件持久化 | Redis Stream（MAXLEN ~ 自动裁剪） | 高性能写入，自动淘汰旧事件 |
| 幂等去重 | Redis SET NX（可配窗口期） | 客户端重试不会产生重复事件 |
| 历史回放 | 支持从任意 seq 续读 | 断线重连只拉取增量 |
| 自动快照 | 异步构建，加速全量回放 | 业务自定义聚合逻辑 |
| 实时订阅 | Redis XREAD BLOCK 长轮询 | 毫秒级推送，不空转 CPU |
| 兜底 TTL | 自动续期 + 最大存活时间 | 防止遗忘 close 导致 key 永驻 |

## 设计理念

```
┌───────────────────────────────────────────────────┐
│                  Envelope                          │
├────────────────────────┬──────────────────────────┤
│   SDK 管理（5 字段）    │   业务透传               │
│                        │                          │
│  EventID   (幂等去重)   │  Body (json.RawMessage)  │
│  SessionID (会话路由)   │  ┌──────────────────┐   │
│  Seq       (全局定序)   │  │ 业务自定义结构    │   │
│  Timestamp (时间戳)     │  │ 引擎不解析不校验  │   │
│  SenderID  (发送方)     │  │ 原样存储原样返回  │   │
│                        │  └──────────────────┘   │
└────────────────────────┴──────────────────────────┘
```

**核心原则**：

- SDK 只管元数据，业务逻辑全部放 `Body`
- `Body` 是 `json.RawMessage`，写入时原样存储，回放/订阅时原样返回
- `SenderID` 是多端协作的通用字段，由业务定义粒度（用户 ID / 设备 ID / 连接 ID 均可）
- 不同业务可定义完全不同的 Body 结构，升级 Body 无需改动 SDK 版本

## 包结构

```
pkg/biz/collab/
├── engine.go      Engine 核心：New / Append / Replay / Subscribe / CloseSession
├── envelope.go    数据结构：Envelope / Session / Snapshot / ReplayResult
├── session.go     会话生命周期：create / get / isActive / close
├── sequencer.go   Redis INCR 全局定序
├── eventlog.go    Redis Stream 读写 + 消息解析
├── dedup.go       SET NX 幂等去重
├── snapshot.go    快照存储 / 加载 / 异步构建
├── subscriber.go  XREAD BLOCK 实时订阅
├── options.go     Functional Options 配置
├── keys.go        Redis key 命名规则（hash tag 集群友好）
├── lua.go         Lua 脚本（INCR + XADD 原子操作）
├── errors.go      错误定义
└── collab_test.go 单元测试（基于 miniredis）
```

## 快速开始

```go
import (
    "encoding/json"
    "github.com/liukunxin/go-infra/pkg/biz/collab"
    redisv8 "github.com/liukunxin/go-infra/pkg/infra/redis/v8"
)

// 1. 创建引擎
engine := collab.New(redisv8.GetClient(),
    collab.WithNamespace("snapsheet"),
    collab.WithMaxSessionTTL(24*time.Hour),  // 兜底：24h 无活动自动清理
    collab.WithSnapEvery(200),
    collab.WithSnapshotBuilder(mySnapshotBuilder),
)

// 2. 创建会话
sess, err := engine.CreateSession(ctx, "session-uuid-123")

// 3. 写入事件
body, _ := json.Marshal(map[string]any{
    "event_type": "write",
    "payload":    map[string]any{"sheet": "Sheet1", "items": items},
})
result, err := engine.Append(ctx, collab.Envelope{
    SessionID: sess.ID,
    SenderID:  "mobile-user-1",
    Body:      body,
})
fmt.Println("seq:", result.Seq)

// 4. 实时订阅（在 goroutine 中）
go engine.Subscribe(ctx, sess.ID, func(evt collab.Envelope) {
    // evt.SenderID 可用于过滤"不推给自己"
    // evt.Body 原样透传，业务自行解析
    pushToClient(evt)
})

// 5. 断线重连回放
replay, err := engine.Replay(ctx, sess.ID, lastKnownSeq)
// replay.Snapshot  → 快照（如有）
// replay.Events   → seq > lastKnownSeq 的增量事件
// replay.LastSeq  → 当前最大 seq

// 6. 关闭会话
engine.CloseSession(ctx, sess.ID, 10*time.Minute)
```

## 业务 Body 示例

Body 内容完全由业务自行定义，SDK 不做任何限制：

**SnapSheet（表格协作）**：
```json
{
  "event_type": "write",
  "payload": { "sheet": "Sheet1", "items": [{"type": "row", "data": {"姓名": "张三"}}] }
}
```

**语音速记（ASR 协作）**：
```json
{
  "event_type": "asr.partial",
  "payload": { "text": "今天的会议...", "is_final": false, "lang": "zh" }
}
```

**白板协作**：
```json
{
  "event_type": "stroke.add",
  "payload": { "path": [[0,0],[100,200]], "color": "#ff0000", "width": 2 }
}
```

## API 文档

### `New(rdb redis.UniversalClient, opts ...Option) *Engine`

创建引擎实例。Engine 线程安全，无内部可变状态，所有数据存储在 Redis 中，支持多实例水平部署。

### `Engine.CreateSession(ctx, id) (Session, error)`

创建新会话。`id` 由业务方生成（建议 UUID）。
- 重复创建返回 `ErrSessionExists`
- 如果配置了 `MaxSessionTTL`，meta key 会设置兜底过期时间

### `Engine.Append(ctx, evt) (Envelope, error)`

写入事件，核心写入流程：

```
参数校验 → 会话状态检查 → 自动填充 EventID/Timestamp
    → 幂等去重（SET NX）→ 原子定序+持久化（Lua: INCR+XADD）
    → 续期兜底 TTL → 异步快照检查
```

- 返回的 Envelope 中 `Seq` 已填充
- `EventID` 和 `Timestamp` 为空时自动生成
- `Body` 原样写入 Stream，不做任何修改

### `Engine.Replay(ctx, sessionID, fromSeq) (ReplayResult, error)`

历史回放：
- `fromSeq=0`：全量回放，优先加载快照加速
- `fromSeq>0`：增量回放，只返回 seq > fromSeq 的事件
- `Body` 原样返回，与写入时完全一致

### `Engine.Subscribe(ctx, sessionID, handler) error`

实时订阅，阻塞式调用（应在 goroutine 中运行）：
- 内部使用 Redis XREAD BLOCK 长轮询
- 有新事件时立即回调 `handler`
- `ctx` 取消时优雅退出
- 网络异常自动重试（500ms 间隔）

### `Engine.CloseSession(ctx, sessionID, ttl) error`

关闭会话：
- 将状态标记为 `closed`，后续 Append 会返回 `ErrSessionClosed`
- 对所有相关 Redis key 设置 TTL，到期后自动清理

## 配置项

| Option | 默认值 | 说明 |
|--------|--------|------|
| `WithNamespace(ns)` | `"collab"` | Redis key 前缀，不同业务使用不同 namespace 隔离 |
| `WithStreamMaxLen(n)` | `50000` | Stream 最大条目数（MAXLEN ~，近似裁剪） |
| `WithSnapEvery(n)` | `500` | 每 N 条事件自动构建快照，0 表示禁用 |
| `WithDedupTTL(d)` | `30min` | 去重窗口时长（过期后相同 EventID 可再次写入） |
| `WithBlockTimeout(d)` | `5s` | Subscribe 的 XREAD BLOCK 超时时间 |
| `WithMaxSessionTTL(d)` | `0`（不设置） | 兜底 TTL，防止遗忘 close；每次 Append 自动续期 |
| `WithSnapshotBuilder(fn)` | `nil` | 快照构建函数，不提供则不做自动快照 |

### SnapshotBuilder

```go
type SnapshotBuilder func(events []Envelope) (json.RawMessage, error)
```

业务方自定义快照聚合逻辑。引擎不知道如何折叠业务状态，当 `seq % SnapEvery == 0` 时异步调用此函数：

```go
collab.WithSnapshotBuilder(func(events []collab.Envelope) (json.RawMessage, error) {
    rows := []any{}
    for _, e := range events {
        var body struct {
            EventType string         `json:"event_type"`
            Payload   map[string]any `json:"payload"`
        }
        json.Unmarshal(e.Body, &body)
        if body.EventType == "write" {
            rows = append(rows, body.Payload["items"])
        }
    }
    return json.Marshal(map[string]any{"rows": rows})
})
```

## Redis 存储设计

所有 key 使用 `{sid}` hash tag 确保同一 session 的数据落在同一 Redis slot（集群友好）：

| 用途 | Key 模式 | Redis 类型 | 说明 |
|------|---------|-----------|------|
| 事件流 | `{ns}:sess:{sid}:events` | Stream | MAXLEN ~ 50000 |
| 序号计数器 | `{ns}:sess:{sid}:seq` | String(int) | INCR 原子递增 |
| 快照 | `{ns}:sess:{sid}:snapshot` | String(JSON) | 最新一份快照 |
| 会话元数据 | `{ns}:sess:{sid}:meta` | Hash | status / created_at |
| 去重标记 | `{ns}:dedup:{event_id}` | String | SET NX EX 30min |

### 数据生命周期

```
CreateSession ──→ key 创建（可选兜底 TTL）
     │
     ▼
Append (循环) ──→ 每次续期兜底 TTL
     │
     ▼
CloseSession ───→ 设置短 TTL（如 10min），到期自动清理
```

## 错误码

| 错误 | 含义 | 触发时机 |
|------|------|---------|
| `ErrDuplicate` | 事件重复 | 相同 EventID 在去重窗口内再次写入 |
| `ErrSessionClosed` | 会话已关闭 | 向已 close 的 session 写入 |
| `ErrSessionNotFound` | 会话不存在 | 操作一个不存在或已过期的 session |
| `ErrSessionExists` | 会话已存在 | 重复创建相同 ID 的 session |
| `ErrEmptySessionID` | SessionID 为空 | 调用方未填 SessionID |

## 设计原则

1. **Body 透传**：SDK 对 `Envelope.Body` 不解析、不校验、不修改，序列化时原样嵌入 JSON
2. **零业务侵入**：引擎不 import 任何业务包，不知道 Body 里有什么
3. **线程安全**：Engine 无内部可变状态，所有状态存储在 Redis
4. **集群友好**：`{sid}` hash tag 保证同一 session 的操作落在同一 slot
5. **向后兼容**：业务升级 Body 结构无需改动 SDK 版本
6. **防泄漏**：MaxSessionTTL 兜底机制防止 Redis key 永久残留
7. **最小依赖**：只依赖 `go-redis/v8` + `encoding/json` + 标准库 + `pkg/base/log`

## 性能参考

| 指标 | 预期值 | 说明 |
|------|--------|------|
| 单 session 写入 QPS | >10,000 ops/s | 取决于 Redis 性能 |
| Subscribe 延迟 | <5ms（同机房） | XREAD BLOCK 长轮询 |
| 回放速度 | ~50,000 事件/s | XRANGE 顺序读取 |
| CPU 空转 | 无 | XREAD BLOCK 阻塞，不忙轮询 |

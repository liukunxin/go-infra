package collab

import "encoding/json"

// Envelope 事件信封，跨端协作中传递的通用消息载体。
//
// SDK 管理 5 个元数据字段（去重、路由、定序、时间戳、发送方），
// 业务方的所有数据放在 Body 中自行定义结构，引擎对 Body 做纯透传。
type Envelope struct {
	// ── SDK 管理的字段 ──
	EventID   string `json:"event_id"`   // UUID，幂等去重键
	SessionID string `json:"session_id"` // 所属会话，路由键
	Seq       int64  `json:"seq"`        // 全局递增序号（引擎分配，调用方不填）
	Timestamp int64  `json:"timestamp"`  // unix 毫秒（引擎自动填充）
	SenderID  string `json:"sender_id"`  // 发送方标识，由业务定义粒度（可以是用户/设备/连接）

	// ── 业务透传（引擎不解析、不关心）──
	Body json.RawMessage `json:"body"` // 业务自定义内容，原样存储和回放
}

// Session 会话元数据。
type Session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`     // "active" / "closed"
	CreatedAt int64  `json:"created_at"` // unix 毫秒
}

// Snapshot 状态快照，用于加速回放。
type Snapshot struct {
	Seq      int64           `json:"seq"`
	Data     json.RawMessage `json:"data"`               // 由 SnapshotBuilder 产出，引擎不解析
	BuiltAt  int64           `json:"built_at"`            // unix 毫秒
	StreamID string          `json:"stream_id,omitempty"` // 快照时刻的 Stream 最新 ID，回放时从此处开始读取
}

// ReplayResult 回放结果。
type ReplayResult struct {
	Snapshot *Snapshot  `json:"snapshot,omitempty"`
	Events   []Envelope `json:"events"`
	LastSeq  int64      `json:"last_seq"`
}

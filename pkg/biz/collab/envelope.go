package collab

// Envelope 事件信封，跨端协作中传递的通用消息载体。
//
// 引擎只关心 EventID / SessionID / Seq，其余字段对引擎透明，由业务自行定义。
type Envelope struct {
	EventID   string         `json:"event_id"`   // UUID，幂等键
	EventType string         `json:"event_type"` // 事件类型（如 "excel.rows.write"，建议用 domain 前缀区分业务域）
	SessionID string         `json:"session_id"` // 所属会话
	Seq       int64          `json:"seq"`        // 全局递增序号（引擎分配，调用方不填）
	SenderID  string         `json:"sender_id"`  // 发送者标识
	Timestamp int64          `json:"timestamp"`  // unix 毫秒
	Payload   map[string]any `json:"payload"`    // 业务载荷（引擎不关心内容）
}

// Session 会话元数据。
type Session struct {
	ID        string `json:"id"`
	Status    string `json:"status"`     // "active" / "closed"
	CreatedAt int64  `json:"created_at"` // unix 毫秒
}

// Snapshot 状态快照，用于加速回放。
type Snapshot struct {
	Seq     int64          `json:"seq"`
	Data    map[string]any `json:"data"`
	BuiltAt int64          `json:"built_at"` // unix 毫秒
}

// ReplayResult 回放结果。
type ReplayResult struct {
	Snapshot *Snapshot  `json:"snapshot,omitempty"`
	Events   []Envelope `json:"events"`
	LastSeq  int64      `json:"last_seq"`
}

package collab

import "time"

// SnapshotBuilder 快照构建函数，由业务方提供。
// 引擎不知道如何折叠业务状态，该函数将事件列表聚合为快照数据。
type SnapshotBuilder func(events []Envelope) map[string]any

// Options 引擎配置。
type Options struct {
	Namespace       string          // Redis key 前缀，默认 "collab"
	StreamMaxLen    int64           // Stream MAXLEN，默认 50000
	SnapEvery       int64           // 每 N 条事件自动快照，默认 500（0=禁用）
	DedupTTL        time.Duration   // 去重窗口，默认 30 分钟
	BlockTimeout    time.Duration   // Subscribe XREAD BLOCK 超时，默认 5 秒
	SnapshotBuilder SnapshotBuilder // 快照构建函数（可选）
}

func defaultOptions() *Options {
	return &Options{
		Namespace:    "collab",
		StreamMaxLen: 50000,
		SnapEvery:    500,
		DedupTTL:     30 * time.Minute,
		BlockTimeout: 5 * time.Second,
	}
}

// Option 配置函数。
type Option func(*Options)

// WithNamespace 设置 Redis key 前缀。
func WithNamespace(ns string) Option {
	return func(o *Options) { o.Namespace = ns }
}

// WithStreamMaxLen 设置 Stream 最大长度。
func WithStreamMaxLen(n int64) Option {
	return func(o *Options) { o.StreamMaxLen = n }
}

// WithSnapEvery 设置自动快照间隔（每 N 条事件），0 表示禁用。
func WithSnapEvery(n int64) Option {
	return func(o *Options) { o.SnapEvery = n }
}

// WithDedupTTL 设置去重窗口时长。
func WithDedupTTL(d time.Duration) Option {
	return func(o *Options) { o.DedupTTL = d }
}

// WithBlockTimeout 设置 Subscribe XREAD BLOCK 超时。
func WithBlockTimeout(d time.Duration) Option {
	return func(o *Options) { o.BlockTimeout = d }
}

// WithSnapshotBuilder 设置快照构建函数。
// 如果不提供，引擎不会自动构建快照。
func WithSnapshotBuilder(b SnapshotBuilder) Option {
	return func(o *Options) { o.SnapshotBuilder = b }
}

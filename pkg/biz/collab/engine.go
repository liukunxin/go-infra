package collab

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gofrs/uuid"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

// Engine 实时协作引擎。
//
// 为业务服务提供"多端实时协作"能力：全局定序、事件持久化、幂等去重、历史回放、自动快照、实时订阅。
// 线程安全，所有状态存储在 Redis 中。
type Engine struct {
	rdb  redis.UniversalClient
	opts *Options

	seq      *sequencer
	eventLog *eventLog
	dedup    *dedup
	snap     *snapshotStore
	session  *sessionManager
	sub      *subscriber
}

// New 创建引擎实例。
func New(rdb redis.UniversalClient, opts ...Option) *Engine {
	o := defaultOptions()
	for _, fn := range opts {
		fn(o)
	}

	ns := o.Namespace
	return &Engine{
		rdb:      rdb,
		opts:     o,
		seq:      newSequencer(rdb, ns),
		eventLog: newEventLog(rdb, ns),
		dedup:    newDedup(rdb, ns, o.DedupTTL),
		snap:     newSnapshotStore(rdb, ns),
		session:  newSessionManager(rdb, ns),
		sub:      newSubscriber(rdb, ns, o.BlockTimeout),
	}
}

// CreateSession 创建新会话。
func (e *Engine) CreateSession(ctx context.Context, id string) (Session, error) {
	if id == "" {
		return Session{}, ErrEmptySessionID
	}
	return e.session.create(ctx, id)
}

// Append 写入事件（原子：去重 → 分配 seq → 持久化 → 触发快照检查）。
// 返回已定序的事件（seq 已填充）。
func (e *Engine) Append(ctx context.Context, evt Envelope) (Envelope, error) {
	if evt.SessionID == "" {
		return Envelope{}, ErrEmptySessionID
	}

	// 检查会话状态
	active, err := e.session.isActive(ctx, evt.SessionID)
	if err != nil {
		return Envelope{}, err
	}
	if !active {
		return Envelope{}, ErrSessionClosed
	}

	// 自动填充
	if evt.EventID == "" {
		evt.EventID = uuid.Must(uuid.NewV4()).String()
	}
	if evt.Timestamp == 0 {
		evt.Timestamp = time.Now().UnixMilli()
	}

	// 去重检查
	isNew, err := e.dedup.check(ctx, evt.EventID)
	if err != nil {
		return Envelope{}, err
	}
	if !isNew {
		log.New().Infof("collab: duplicate event intercepted, event_id=%s session=%s sender=%s", evt.EventID, evt.SessionID, evt.SenderID)
		return Envelope{}, ErrDuplicate
	}

	// 原子定序 + 持久化（Lua 脚本）
	seqK := seqKey(e.opts.Namespace, evt.SessionID)
	streamK := streamKey(e.opts.Namespace, evt.SessionID)

	// 先序列化（不含 seq，seq 由 Redis 分配后回填）
	payload, err := json.Marshal(evt)
	if err != nil {
		return Envelope{}, err
	}

	seq, err := e.rdb.Eval(ctx, luaAppend, []string{seqK, streamK},
		strconv.FormatInt(e.opts.StreamMaxLen, 10),
		string(payload),
	).Int64()
	if err != nil {
		return Envelope{}, err
	}

	evt.Seq = seq

	// 快照检查（异步）
	if e.opts.SnapshotBuilder != nil && e.opts.SnapEvery > 0 && seq%e.opts.SnapEvery == 0 {
		e.snap.buildAsync(e.eventLog, evt.SessionID, seq, e.opts.SnapshotBuilder)
	}

	return evt, nil
}

// Replay 从指定 seq 开始回放历史事件。
// fromSeq=0 表示全量回放（会优先加载快照）。
func (e *Engine) Replay(ctx context.Context, sessionID string, fromSeq int64) (ReplayResult, error) {
	if sessionID == "" {
		return ReplayResult{}, ErrEmptySessionID
	}

	// 检查会话是否存在
	if _, err := e.session.get(ctx, sessionID); err != nil {
		return ReplayResult{}, err
	}

	var result ReplayResult
	effectiveFromSeq := fromSeq

	// 尝试加载快照
	if fromSeq == 0 {
		snap, err := e.snap.load(ctx, sessionID)
		if err != nil {
			return ReplayResult{}, err
		}
		if snap != nil {
			result.Snapshot = snap
			effectiveFromSeq = snap.Seq
		}
	}

	// 读取事件
	events, err := e.eventLog.readAll(ctx, sessionID)
	if err != nil {
		return ReplayResult{}, err
	}

	// 过滤 seq > effectiveFromSeq 的事件
	filtered := make([]Envelope, 0, len(events))
	for i := range events {
		if events[i].Seq > effectiveFromSeq {
			filtered = append(filtered, events[i])
		}
	}
	result.Events = filtered

	// 获取 lastSeq
	lastSeq, err := e.seq.current(ctx, sessionID)
	if err != nil {
		return ReplayResult{}, err
	}
	result.LastSeq = lastSeq

	return result, nil
}

// Subscribe 订阅会话的实时事件（阻塞式，应在 goroutine 中调用）。
// 内部使用 Redis XREAD BLOCK，有新事件时回调 handler。
// ctx 取消时退出。
func (e *Engine) Subscribe(ctx context.Context, sessionID string, handler func(Envelope)) error {
	if sessionID == "" {
		return ErrEmptySessionID
	}
	return e.sub.listen(ctx, sessionID, handler)
}

// CloseSession 关闭会话（设置 Redis key TTL，到期后自动清理）。
func (e *Engine) CloseSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	if sessionID == "" {
		return ErrEmptySessionID
	}
	log.New().Infof("collab: session closing, session=%s ttl=%v", sessionID, ttl)
	return e.session.close(ctx, sessionID, ttl)
}

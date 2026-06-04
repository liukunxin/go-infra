package collab

import (
	"context"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

// sessionManager 会话生命周期管理。
type sessionManager struct {
	rdb redis.UniversalClient
	ns  string
}

func newSessionManager(rdb redis.UniversalClient, ns string) *sessionManager {
	return &sessionManager{rdb: rdb, ns: ns}
}

// create 创建新会话，如果已存在返回 ErrSessionExists。
func (sm *sessionManager) create(ctx context.Context, id string) (Session, error) {
	key := metaKey(sm.ns, id)
	now := time.Now().UnixMilli()

	// HSETNX status 字段作为原子性检测是否已存在
	ok, err := sm.rdb.HSetNX(ctx, key, "status", "active").Result()
	if err != nil {
		return Session{}, err
	}
	if !ok {
		return Session{}, ErrSessionExists
	}

	if err = sm.rdb.HSet(ctx, key, "created_at", strconv.FormatInt(now, 10)).Err(); err != nil {
		return Session{}, err
	}

	return Session{
		ID:        id,
		Status:    "active",
		CreatedAt: now,
	}, nil
}

// get 获取会话元数据。
func (sm *sessionManager) get(ctx context.Context, id string) (Session, error) {
	key := metaKey(sm.ns, id)
	vals, err := sm.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return Session{}, err
	}
	if len(vals) == 0 {
		return Session{}, ErrSessionNotFound
	}

	createdAt, _ := strconv.ParseInt(vals["created_at"], 10, 64)
	return Session{
		ID:        id,
		Status:    vals["status"],
		CreatedAt: createdAt,
	}, nil
}

// isActive 检查会话是否处于活跃状态。
func (sm *sessionManager) isActive(ctx context.Context, id string) (bool, error) {
	key := metaKey(sm.ns, id)
	status, err := sm.rdb.HGet(ctx, key, "status").Result()
	if err == redis.Nil {
		return false, ErrSessionNotFound
	}
	if err != nil {
		return false, err
	}
	return status == "active", nil
}

// close 关闭会话并为相关 key 设置过期时间。
func (sm *sessionManager) close(ctx context.Context, id string, ttl time.Duration) error {
	key := metaKey(sm.ns, id)

	// 检查是否存在
	exists, err := sm.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrSessionNotFound
	}

	// 设置状态为 closed
	if err = sm.rdb.HSet(ctx, key, "status", "closed").Err(); err != nil {
		return err
	}

	// 对所有相关 key 设置过期时间
	keys := []string{
		metaKey(sm.ns, id),
		streamKey(sm.ns, id),
		seqKey(sm.ns, id),
		snapshotKey(sm.ns, id),
	}
	for _, k := range keys {
		sm.rdb.Expire(ctx, k, ttl)
	}
	return nil
}

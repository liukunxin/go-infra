package code

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	// ErrNotFound 验证码不存在或已过期。
	ErrNotFound = errors.New("code: not found or expired")
	// ErrMismatch 验证码不匹配。
	ErrMismatch = errors.New("code: mismatch")
)

// CodeStore 验证码存储接口。
//
// SDK 内置两种实现，按部署规模选择：
//   - NewMemoryStore / NewMemoryStoreWithCleanup：单进程、低量级场景，无额外依赖。
//   - NewRedisStore：多实例或量级较大场景，需要 Redis。
//
// 如有特殊需求（如自定义 KV 存储），实现此接口即可，无需改动上层逻辑。
type CodeStore interface {
	// Save 将 code 以 key 为键存入，有效期为 ttl。
	Save(ctx context.Context, key, code string, ttl time.Duration) error
	// Verify 校验 key 对应的验证码是否与 code 一致。
	// 校验成功后立即删除（一次性使用）；失败返回 ErrNotFound 或 ErrMismatch。
	Verify(ctx context.Context, key, code string) error
	// Exists 检查 key 是否存在（用于发送频率限制等场景）。
	Exists(ctx context.Context, key string) (bool, error)
}

type redisStore struct {
	rdb redis.UniversalClient
}

// NewRedisStore 使用 go-redis UniversalClient 创建 CodeStore。
// 支持单节点与集群模式，可直接传入 pkg/infra/redis 的 GetClient() 返回值。
func NewRedisStore(rdb redis.UniversalClient) CodeStore {
	return &redisStore{rdb: rdb}
}

func (s *redisStore) Save(ctx context.Context, key, code string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key, code, ttl).Err()
}

func (s *redisStore) Verify(ctx context.Context, key, code string) error {
	stored, err := s.rdb.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if stored != code {
		return ErrMismatch
	}
	// 校验成功后删除，防止重复使用
	_ = s.rdb.Del(ctx, key).Err()
	return nil
}

func (s *redisStore) Exists(ctx context.Context, key string) (bool, error) {
	n, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

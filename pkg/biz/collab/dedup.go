package collab

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// dedup 基于 Redis SET NX 实现幂等去重。
type dedup struct {
	rdb redis.UniversalClient
	ns  string
	ttl time.Duration
}

func newDedup(rdb redis.UniversalClient, ns string, ttl time.Duration) *dedup {
	return &dedup{rdb: rdb, ns: ns, ttl: ttl}
}

// check 检查事件是否重复。
// 返回 true 表示事件是新的（成功占位），false 表示重复。
func (d *dedup) check(ctx context.Context, eventID string) (bool, error) {
	key := dedupKey(d.ns, eventID)
	ok, err := d.rdb.SetNX(ctx, key, "1", d.ttl).Result()
	if err != nil {
		return false, err
	}
	return ok, nil
}

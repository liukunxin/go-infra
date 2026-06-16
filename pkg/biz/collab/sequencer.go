package collab

import (
	"context"
	"errors"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// sequencer 基于 Redis INCR 实现全局单调递增定序。
type sequencer struct {
	rdb redis.UniversalClient
	ns  string
}

func newSequencer(rdb redis.UniversalClient, ns string) *sequencer {
	return &sequencer{rdb: rdb, ns: ns}
}

// current 获取当前最大序号，如果 key 不存在返回 0。
func (s *sequencer) current(ctx context.Context, sessionID string) (int64, error) {
	val, err := s.rdb.Get(ctx, seqKey(s.ns, sessionID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(val, 10, 64)
}

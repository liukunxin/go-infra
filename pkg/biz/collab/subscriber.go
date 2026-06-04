package collab

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

// subscriber 实时事件订阅，基于 Redis XREAD BLOCK 长轮询。
type subscriber struct {
	rdb          redis.UniversalClient
	ns           string
	blockTimeout time.Duration
}

func newSubscriber(rdb redis.UniversalClient, ns string, blockTimeout time.Duration) *subscriber {
	return &subscriber{rdb: rdb, ns: ns, blockTimeout: blockTimeout}
}

// listen 订阅指定 session 的实时事件。
// 阻塞式调用，应在 goroutine 中运行，ctx 取消时退出。
func (s *subscriber) listen(ctx context.Context, sessionID string, handler func(Envelope)) error {
	lastID := "$"
	key := streamKey(s.ns, sessionID)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		results, err := s.rdb.XRead(ctx, &redis.XReadArgs{
			Streams: []string{key, lastID},
			Block:   s.blockTimeout,
			Count:   100,
		}).Result()

		if err == redis.Nil {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			log.New().Warnf("collab: subscribe read error, session=%s err=%v", sessionID, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, stream := range results {
			for _, msg := range stream.Messages {
				lastID = msg.ID
				evt, parseErr := parseMessage(msg)
				if parseErr != nil {
					continue
				}
				handler(evt)
			}
		}
	}
}

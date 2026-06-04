package collab

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-redis/redis/v8"
)

// eventLog 封装 Redis Stream 的读写操作。
type eventLog struct {
	rdb redis.UniversalClient
	ns  string
}

func newEventLog(rdb redis.UniversalClient, ns string) *eventLog {
	return &eventLog{rdb: rdb, ns: ns}
}

// readFrom 从指定 Stream ID 开始读取事件（含起始位置之后的消息）。
// startID 使用 Redis Stream ID 格式，如 "0-0" 表示从头读取。
// count 为每次 XRANGE 返回的最大条数，0 表示不限制。
func (el *eventLog) readFrom(ctx context.Context, sessionID, startID string, count int64) ([]Envelope, error) {
	key := streamKey(el.ns, sessionID)

	var msgs []redis.XMessage
	var err error
	if count > 0 {
		msgs, err = el.rdb.XRangeN(ctx, key, startID, "+", count).Result()
	} else {
		msgs, err = el.rdb.XRange(ctx, key, startID, "+").Result()
	}
	if err != nil {
		return nil, err
	}

	events := make([]Envelope, 0, len(msgs))
	for _, msg := range msgs {
		evt, parseErr := parseMessage(msg)
		if parseErr != nil {
			continue
		}
		events = append(events, evt)
	}
	return events, nil
}

// readAll 读取 session 的全部事件。
func (el *eventLog) readAll(ctx context.Context, sessionID string) ([]Envelope, error) {
	return el.readFrom(ctx, sessionID, "0-0", 0)
}

// parseMessage 从 Redis Stream Message 解析出 Envelope。
// Stream 消息中 seq 字段为权威序号，data 字段为 JSON 载荷。
func parseMessage(msg redis.XMessage) (Envelope, error) {
	var evt Envelope
	data, ok := msg.Values["data"]
	if !ok {
		return evt, nil
	}
	if err := json.Unmarshal([]byte(data.(string)), &evt); err != nil {
		return evt, err
	}
	// 以 Stream 中存储的 seq 为准（Lua 原子分配）
	if seqStr, exists := msg.Values["seq"]; exists {
		if s, ok := seqStr.(string); ok {
			seq, _ := strconv.ParseInt(s, 10, 64)
			evt.Seq = seq
		}
	}
	return evt, nil
}

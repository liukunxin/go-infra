package collab

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

// snapshotStore 快照的存储与加载。
type snapshotStore struct {
	rdb           redis.UniversalClient
	ns            string
	maxSessionTTL time.Duration
}

func newSnapshotStore(rdb redis.UniversalClient, ns string, maxTTL time.Duration) *snapshotStore {
	return &snapshotStore{rdb: rdb, ns: ns, maxSessionTTL: maxTTL}
}

// load 加载最新快照，不存在时返回 nil。
func (ss *snapshotStore) load(ctx context.Context, sessionID string) (*Snapshot, error) {
	key := snapshotKey(ss.ns, sessionID)
	val, err := ss.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var snap Snapshot
	if err = json.Unmarshal([]byte(val), &snap); err != nil {
		return nil, err
	}
	return &snap, nil
}

// save 保存快照。
func (ss *snapshotStore) save(ctx context.Context, sessionID string, snap *Snapshot) error {
	key := snapshotKey(ss.ns, sessionID)
	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return ss.rdb.Set(ctx, key, raw, ss.maxSessionTTL).Err()
}

// buildAsync 异步构建快照（在 goroutine 中调用）。
func (ss *snapshotStore) buildAsync(el *eventLog, sessionID string, seq int64, builder SnapshotBuilder) {
	go func() {
		ctx := context.Background()
		events, err := el.readAll(ctx, sessionID)
		if err != nil {
			log.New().Warnf("collab: snapshot build failed, session=%s err=%v", sessionID, err)
			return
		}

		data, err := builder(events)
		if err != nil {
			log.New().Warnf("collab: snapshot builder error, session=%s err=%v", sessionID, err)
			return
		}

		snap := &Snapshot{
			Seq:     seq,
			Data:    data,
			BuiltAt: time.Now().UnixMilli(),
		}
		if err = ss.save(ctx, sessionID, snap); err != nil {
			log.New().Warnf("collab: snapshot save failed, session=%s err=%v", sessionID, err)
			return
		}
		log.New().Infof("collab: snapshot built, session=%s seq=%d", sessionID, seq)
	}()
}

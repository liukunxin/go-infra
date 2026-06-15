package collab

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

// snapshotStore 快照的存储与加载。
type snapshotStore struct {
	rdb           redis.UniversalClient
	ns            string
	maxSessionTTL time.Duration

	// building tracks which sessions currently have a snapshot build in progress,
	// preventing concurrent builds for the same session.
	building sync.Map // sessionID → struct{}
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

// buildAsync 异步构建快照。
// 同一 session 同时只允许一个 build 运行，防止并发 readAll 导致内存暴涨。
func (ss *snapshotStore) buildAsync(el *eventLog, sessionID string, seq int64, builder SnapshotBuilder) {
	if _, loaded := ss.building.LoadOrStore(sessionID, struct{}{}); loaded {
		return
	}

	go func() {
		defer ss.building.Delete(sessionID)

		ctx := context.Background()

		// 使用 readAllWithLastID 保证 events 和 lastStreamID 来自同一次 XRANGE，无竞态。
		events, lastStreamID, err := el.readAllWithLastID(ctx, sessionID)
		if err != nil {
			log.New().Warnf("collab: snapshot build failed, session=%s err=%v", sessionID, err)
			return
		}

		data, err := builder(events)
		if err != nil {
			log.New().Warnf("collab: snapshot builder error, session=%s err=%v", sessionID, err)
			return
		}

		// 使用实际读到的最大 seq，而非触发 seq。
		// 因为 readAll 时可能已有更新的事件写入，snap.Seq 必须反映快照真实覆盖范围。
		actualSeq := seq
		if len(events) > 0 && events[len(events)-1].Seq > actualSeq {
			actualSeq = events[len(events)-1].Seq
		}

		snap := &Snapshot{
			Seq:      actualSeq,
			Data:     data,
			BuiltAt:  time.Now().UnixMilli(),
			StreamID: lastStreamID,
		}
		if err = ss.save(ctx, sessionID, snap); err != nil {
			log.New().Warnf("collab: snapshot save failed, session=%s err=%v", sessionID, err)
			return
		}
		log.New().Infof("collab: snapshot built, session=%s seq=%d stream_id=%s", sessionID, actualSeq, lastStreamID)
	}()
}

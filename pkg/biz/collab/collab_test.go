package collab

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupTestEngine(t *testing.T, opts ...Option) (*Engine, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	engine := New(rdb, opts...)
	return engine, mr
}

func TestCreateSession(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	sess, err := engine.CreateSession(ctx, "sess-1")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if sess.ID != "sess-1" {
		t.Errorf("expected ID=sess-1, got %s", sess.ID)
	}
	if sess.Status != "active" {
		t.Errorf("expected Status=active, got %s", sess.Status)
	}

	// 重复创建应返回 ErrSessionExists
	_, err = engine.CreateSession(ctx, "sess-1")
	if err != ErrSessionExists {
		t.Errorf("expected ErrSessionExists, got %v", err)
	}

	// 空 ID 应返回 ErrEmptySessionID
	_, err = engine.CreateSession(ctx, "")
	if err != ErrEmptySessionID {
		t.Errorf("expected ErrEmptySessionID, got %v", err)
	}
}

func TestAppendAndReplay(t *testing.T) {
	engine, _ := setupTestEngine(t, WithNamespace("test"))
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")

	// 写入事件
	evt1, err := engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		EventType: "excel.rows.write",
		SenderID:  "user-1",
		Payload:   map[string]any{"row": 1},
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if evt1.Seq != 1 {
		t.Errorf("expected seq=1, got %d", evt1.Seq)
	}
	if evt1.EventID == "" {
		t.Error("expected EventID to be auto-generated")
	}

	evt2, err := engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		EventType: "excel.rows.write",
		SenderID:  "user-2",
		Payload:   map[string]any{"row": 2},
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if evt2.Seq != 2 {
		t.Errorf("expected seq=2, got %d", evt2.Seq)
	}

	// 全量回放
	result, err := engine.Replay(ctx, "sess-1", 0)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if len(result.Events) != 2 {
		t.Errorf("expected 2 events, got %d", len(result.Events))
	}
	if result.LastSeq != 2 {
		t.Errorf("expected LastSeq=2, got %d", result.LastSeq)
	}

	// 增量回放（从 seq=1 开始）
	result, err = engine.Replay(ctx, "sess-1", 1)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
	if result.Events[0].Seq != 2 {
		t.Errorf("expected event seq=2, got %d", result.Events[0].Seq)
	}
}

func TestDeduplicate(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")

	evt := Envelope{
		EventID:   "fixed-uuid",
		SessionID: "sess-1",
		EventType: "test.event",
		SenderID:  "user-1",
	}

	_, err := engine.Append(ctx, evt)
	if err != nil {
		t.Fatalf("first Append failed: %v", err)
	}

	// 重复写入应返回 ErrDuplicate
	_, err = engine.Append(ctx, evt)
	if err != ErrDuplicate {
		t.Errorf("expected ErrDuplicate, got %v", err)
	}
}

func TestAppendToClosedSession(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")
	engine.CloseSession(ctx, "sess-1", 10*time.Minute)

	_, err := engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		EventType: "test.event",
		SenderID:  "user-1",
	})
	if err != ErrSessionClosed {
		t.Errorf("expected ErrSessionClosed, got %v", err)
	}
}

func TestSubscribe(t *testing.T) {
	engine, _ := setupTestEngine(t, WithBlockTimeout(100*time.Millisecond))
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")

	var (
		mu       sync.Mutex
		received []Envelope
	)

	subCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)

	go func() {
		done <- engine.Subscribe(subCtx, "sess-1", func(evt Envelope) {
			mu.Lock()
			received = append(received, evt)
			mu.Unlock()
		})
	}()

	// 等待订阅就绪
	time.Sleep(50 * time.Millisecond)

	// 写入事件
	engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		EventType: "test.push",
		SenderID:  "user-1",
	})

	// 等待事件投递
	time.Sleep(200 * time.Millisecond)

	cancel()
	<-done

	mu.Lock()
	count := len(received)
	mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 received event, got %d", count)
	}
}

func TestSnapshotBuild(t *testing.T) {
	builder := func(events []Envelope) map[string]any {
		count := 0
		for range events {
			count++
		}
		return map[string]any{"count": count}
	}

	engine, _ := setupTestEngine(t,
		WithSnapEvery(3),
		WithSnapshotBuilder(builder),
	)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")

	for i := 0; i < 3; i++ {
		_, err := engine.Append(ctx, Envelope{
			SessionID: "sess-1",
			EventType: "test.event",
			SenderID:  "user-1",
			Payload:   map[string]any{"i": i},
		})
		if err != nil {
			t.Fatalf("Append #%d failed: %v", i, err)
		}
	}

	// 等待异步快照构建完成
	time.Sleep(100 * time.Millisecond)

	// 回放应包含快照
	result, err := engine.Replay(ctx, "sess-1", 0)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if result.Snapshot == nil {
		t.Fatal("expected snapshot to be present")
	}
	if result.Snapshot.Seq != 3 {
		t.Errorf("expected snapshot seq=3, got %d", result.Snapshot.Seq)
	}
}

func TestCloseSession(t *testing.T) {
	engine, mr := setupTestEngine(t)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")
	engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		EventType: "test.event",
		SenderID:  "user-1",
	})

	err := engine.CloseSession(ctx, "sess-1", 5*time.Minute)
	if err != nil {
		t.Fatalf("CloseSession failed: %v", err)
	}

	// 验证 TTL 已设置
	mr.FastForward(6 * time.Minute)

	// 会话 key 应已过期
	result, err := engine.Replay(ctx, "sess-1", 0)
	if err != ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after TTL, got err=%v result=%+v", err, result)
	}
}

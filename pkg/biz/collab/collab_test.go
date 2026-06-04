package collab

import (
	"context"
	"encoding/json"
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

func makeBody(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
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
		SenderID:  "mobile-user-1",
		Body:      makeBody(t, map[string]any{"event_type": "write", "row": 1}),
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
	if evt1.SenderID != "mobile-user-1" {
		t.Errorf("expected SenderID=mobile-user-1, got %s", evt1.SenderID)
	}

	evt2, err := engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		SenderID:  "pc-user-2",
		Body:      makeBody(t, map[string]any{"event_type": "write", "row": 2}),
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

	// 验证 Body 透传完整性
	var body map[string]any
	if err = json.Unmarshal(result.Events[0].Body, &body); err != nil {
		t.Fatalf("Body unmarshal failed: %v", err)
	}
	if body["event_type"] != "write" {
		t.Errorf("expected event_type=write, got %v", body["event_type"])
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
		Body:      makeBody(t, map[string]any{"event_type": "test"}),
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
		Body:      makeBody(t, map[string]any{"event_type": "test"}),
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
		Body:      makeBody(t, map[string]any{"event_type": "push", "msg": "hello"}),
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
	builder := func(events []Envelope) (json.RawMessage, error) {
		return json.Marshal(map[string]any{"count": len(events)})
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
			Body:      makeBody(t, map[string]any{"i": i}),
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

	// 验证快照 Data 内容
	var snapData map[string]any
	if err = json.Unmarshal(result.Snapshot.Data, &snapData); err != nil {
		t.Fatalf("snapshot Data unmarshal failed: %v", err)
	}
	if snapData["count"] != float64(3) {
		t.Errorf("expected snapshot count=3, got %v", snapData["count"])
	}
}

func TestCloseSession(t *testing.T) {
	engine, mr := setupTestEngine(t)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")
	engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		Body:      makeBody(t, map[string]any{"event_type": "test"}),
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

func TestBodyTransparency(t *testing.T) {
	engine, _ := setupTestEngine(t)
	ctx := context.Background()

	engine.CreateSession(ctx, "sess-1")

	// 构造一个复杂的业务 Body
	originalBody := map[string]any{
		"event_type": "asr.partial",
		"payload": map[string]any{
			"text":     "今天的会议内容",
			"is_final": false,
			"segments": []any{1, 2, 3},
		},
	}

	bodyBytes := makeBody(t, originalBody)
	appended, err := engine.Append(ctx, Envelope{
		SessionID: "sess-1",
		SenderID:  "pc-mic",
		Body:      bodyBytes,
	})
	if err != nil {
		t.Fatalf("Append failed: %v", err)
	}
	if appended.SenderID != "pc-mic" {
		t.Errorf("expected SenderID=pc-mic, got %s", appended.SenderID)
	}

	// 回放后验证 Body 完全一致 + SenderID 保留
	result, err := engine.Replay(ctx, "sess-1", 0)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	evt := result.Events[0]
	if evt.SenderID != "pc-mic" {
		t.Errorf("replay SenderID mismatch: got %s", evt.SenderID)
	}

	var recovered map[string]any
	if err = json.Unmarshal(evt.Body, &recovered); err != nil {
		t.Fatalf("Body unmarshal failed: %v", err)
	}
	if recovered["event_type"] != "asr.partial" {
		t.Errorf("Body event_type mismatch: %v", recovered["event_type"])
	}
	payload := recovered["payload"].(map[string]any)
	if payload["text"] != "今天的会议内容" {
		t.Errorf("Body payload.text mismatch: %v", payload["text"])
	}
}

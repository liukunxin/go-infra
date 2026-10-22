package websocket

import (
	"context"
	"testing"
	"time"
)

func TestHubBroadcastNonBlocking(t *testing.T) {
	h := NewHub()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	slow := &Conn{
		id:     "slow",
		cfg:    Config{SendQueueSize: 1, Backpressure: BackpressureBlock}.normalized(),
		ctx:    ctx,
		cancel: cancel,
		sendCh: make(chan outboundMessage, 1),
	}
	// Fill the queue so a blocking Send would stall.
	slow.sendCh <- outboundMessage{msgType: MessageText, data: []byte("x")}

	fast := &Conn{
		id:     "fast",
		cfg:    Config{SendQueueSize: 4, Backpressure: BackpressureBlock}.normalized(),
		ctx:    ctx,
		cancel: cancel,
		sendCh: make(chan outboundMessage, 4),
	}

	h.Register(slow)
	h.Register(fast)

	done := make(chan struct{})
	go func() {
		failed := h.Broadcast(MessageText, []byte("hello"))
		if failed != 1 {
			t.Errorf("failed=%d want 1", failed)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked on slow peer")
	}

	if len(fast.sendCh) != 1 {
		t.Fatalf("fast queue len=%d want 1", len(fast.sendCh))
	}
}

func TestHubLen(t *testing.T) {
	h := NewHub()
	c := &Conn{id: "a"}
	h.Register(c)
	if h.Len() != 1 {
		t.Fatalf("len=%d", h.Len())
	}
	h.Unregister(c)
	if h.Len() != 0 {
		t.Fatalf("len=%d", h.Len())
	}
}

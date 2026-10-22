package websocket

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestConfigNormalizedBackpressureAlias(t *testing.T) {
	cfg := Config{DropOnBackpressure: true}.normalized()
	if cfg.Backpressure != BackpressureError {
		t.Fatalf("want BackpressureError, got %v", cfg.Backpressure)
	}
}

func TestConnMetadata(t *testing.T) {
	c := &Conn{}
	c.Set("user", "u1")
	v, ok := c.Get("user")
	if !ok || v != "u1" {
		t.Fatalf("Get=%v ok=%v", v, ok)
	}
	c.Delete("user")
	if _, ok = c.Get("user"); ok {
		t.Fatal("expected deleted")
	}
}

func TestSendQueueFull(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := Config{
		SendQueueSize: 1,
		Backpressure:  BackpressureError,
		WriteTimeout:  time.Second,
		PingInterval:  time.Hour,
	}.normalized()

	c := &Conn{
		cfg:       cfg,
		ctx:       ctx,
		cancel:    cancel,
		sendCh:    make(chan outboundMessage, 1),
		wakeDrain: make(chan struct{}),
	}
	if err := c.Send(MessageText, []byte("a")); err != nil {
		t.Fatal(err)
	}
	err := c.Send(MessageText, []byte("b"))
	if !errors.Is(err, ErrSendQueueFull) {
		t.Fatalf("want ErrSendQueueFull, got %v", err)
	}
}

func TestCloseNilConn(t *testing.T) {
	var c *Conn
	if err := c.Close(); !errors.Is(err, ErrConnNil) {
		t.Fatalf("got %v", err)
	}
}

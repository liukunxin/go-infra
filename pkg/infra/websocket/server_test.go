package websocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cws "github.com/coder/websocket"
)

func TestServerEchoAndMetadata(t *testing.T) {
	var connected atomic.Bool
	srv := NewServer(
		Config{
			PingInterval:    time.Hour,
			ReadTimeout:     5 * time.Second,
			WriteTimeout:    2 * time.Second,
			SendQueueSize:   16,
			MaxMessageBytes: 1 << 20,
		},
		Handlers{
			OnConnect: func(c *Conn) {
				c.Set("role", "echo")
				connected.Store(true)
			},
			OnMessage: func(c *Conn, m Message) {
				if v, _ := c.Get("role"); v != "echo" {
					t.Errorf("metadata missing: %v", v)
				}
				_ = c.Send(m.Type, m.Data)
			},
		},
		nil,
		func(*http.Request) bool { return true },
	)

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	raw, _, err := cws.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer raw.Close(cws.StatusNormalClosure, "")

	waitUntil(t, time.Second, connected.Load)

	payload := []byte("hello-binary")
	if err = raw.Write(ctx, cws.MessageBinary, payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	typ, data, err := raw.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if typ != cws.MessageBinary || string(data) != string(payload) {
		t.Fatalf("echo got type=%v data=%q", typ, data)
	}
}

func TestServerOnTick(t *testing.T) {
	var ticks atomic.Int64
	srv := NewServer(
		Config{
			PingInterval:  40 * time.Millisecond,
			ReadTimeout:   2 * time.Second,
			WriteTimeout:  time.Second,
			SendQueueSize: 8,
		},
		Handlers{
			OnTick: func(*Conn) { ticks.Add(1) },
		},
		nil,
		func(*http.Request) bool { return true },
	)

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	raw, _, err := cws.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer raw.Close(cws.StatusNormalClosure, "")

	// Peer must keep reading so Ping/Pong can complete; otherwise the server
	// write loop closes after the first ping timeout and OnTick barely fires.
	go func() {
		for {
			if _, _, err := raw.Read(ctx); err != nil {
				return
			}
		}
	}()

	waitUntil(t, 2*time.Second, func() bool { return ticks.Load() >= 2 })
}

func TestServerShutdown(t *testing.T) {
	srv := NewServer(
		Config{
			PingInterval:  time.Hour,
			ReadTimeout:   5 * time.Second,
			WriteTimeout:  time.Second,
			SendQueueSize: 8,
		},
		Handlers{},
		nil,
		func(*http.Request) bool { return true },
	)

	httpSrv := httptest.NewServer(srv)
	defer httpSrv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	wsURL := "ws" + strings.TrimPrefix(httpSrv.URL, "http")
	raw, _, err := cws.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer raw.Close(cws.StatusNormalClosure, "")

	waitUntil(t, time.Second, func() bool { return srv.ActiveConnections() == 1 })

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutCancel()
	if err = srv.Shutdown(shutCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	if n := srv.ActiveConnections(); n != 0 {
		t.Fatalf("active=%d after shutdown", n)
	}

	// New upgrades should be rejected.
	_, resp, err := cws.Dial(ctx, wsURL, nil)
	if err == nil {
		t.Fatal("expected dial failure after shutdown")
	}
	if resp != nil && resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status=%d want 503", resp.StatusCode)
	}
}

func waitUntil(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timeout waiting for condition")
}

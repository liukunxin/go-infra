package core_test

import (
	"bytes"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/liukunxin/go-infra/pkg/base/log/core"
)

type memProvider struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (p *memProvider) WriteLine(b []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = p.buf.Write(b)
}

func (p *memProvider) Close() error { return nil }

func (p *memProvider) String() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.buf.String()
}

func TestLoggerIdleDoesNotSpin(t *testing.T) {
	p := &memProvider{}
	l := core.NewLogger(core.LevelInfo, p, &core.JSONFormatter{}, 64)
	defer l.Close()

	// Give consumer time to park; if it busy-polled at 1µs this would burn CPU.
	time.Sleep(50 * time.Millisecond)
	l.Log(core.LevelInfo, "hello", "", "", nil)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if bytes.Contains([]byte(p.String()), []byte("hello")) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("log not flushed: %q", p.String())
}

func TestLoggerConcurrentAndClose(t *testing.T) {
	p := &memProvider{}
	l := core.NewLogger(core.LevelDebug, p, &core.TxtLineFormatter{}, 128)

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				l.Log(core.LevelInfo, "x", "", "", nil)
			}
		}(i)
	}
	wg.Wait()
	l.Close()

	if got := p.String(); len(got) == 0 {
		t.Fatal("expected logs")
	}
}

func TestEnabledSkipsWork(t *testing.T) {
	p := &memProvider{}
	l := core.NewLogger(core.LevelError, p, &core.JSONFormatter{}, 32)
	defer l.Close()

	if l.Enabled(core.LevelInfo) {
		t.Fatal("info should be disabled")
	}
	l.Output(1, core.LevelDebug, "should-not-appear %d", 1)
	time.Sleep(20 * time.Millisecond)
	if bytes.Contains([]byte(p.String()), []byte("should-not-appear")) {
		t.Fatalf("filtered log appeared: %q", p.String())
	}
}

func TestJSONFormatterOutput(t *testing.T) {
	var buf bytes.Buffer
	p := &writerProvider{w: &buf}
	l := core.NewLogger(core.LevelInfo, p, &core.JSONFormatter{}, 16)
	l.Log(core.LevelInfo, "ok", "tid", "sid", map[string]interface{}{"k": "v"})
	l.Close()
	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"ok"`)) {
		t.Fatalf("bad json: %s", buf.String())
	}
}

type writerProvider struct{ w io.Writer }

func (p *writerProvider) WriteLine(b []byte) { _, _ = p.w.Write(b) }
func (p *writerProvider) Close() error       { return nil }

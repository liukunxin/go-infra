package sse

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Writer provides concurrent-safe Server-Sent Event writing over a Gin response.
type Writer struct {
	ctx  *gin.Context
	mu   sync.Mutex
	done chan struct{}
}

// NewWriter sets SSE response headers and returns a ready-to-use Writer.
func NewWriter(ctx *gin.Context) *Writer {
	ctx.Writer.Header().Set("Content-Type", "text/event-stream")
	ctx.Writer.Header().Set("Cache-Control", "no-cache")
	ctx.Writer.Header().Set("Connection", "keep-alive")
	ctx.Writer.Header().Set("X-Accel-Buffering", "no")
	ctx.Writer.Flush()
	return &Writer{ctx: ctx, done: make(chan struct{})}
}

// SendJSON writes a data-only SSE frame with JSON-encoded payload.
func (w *Writer) SendJSON(v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.ctx.Writer, "data: %s\n\n", payload)
	w.ctx.Writer.Flush()
	return nil
}

// SendEvent writes a named SSE frame (event + data).
func (w *Writer) SendEvent(event string, v interface{}) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	fmt.Fprintf(w.ctx.Writer, "event: %s\ndata: %s\n\n", event, payload)
	w.ctx.Writer.Flush()
	return nil
}

// SendRaw writes pre-formatted bytes directly to the SSE stream.
// Useful for proxying upstream SSE content without re-encoding.
func (w *Writer) SendRaw(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.ctx.Writer.Write(p)
	w.ctx.Writer.Flush()
	return n, err
}

// StartHeartbeat sends the given payload as a data frame at the specified interval.
// The goroutine exits when Stop is called or the client disconnects.
func (w *Writer) StartHeartbeat(interval time.Duration, payload interface{}) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.done:
				return
			case <-w.ctx.Request.Context().Done():
				return
			case <-ticker.C:
				_ = w.SendJSON(payload)
			}
		}
	}()
}

// Done returns a channel that is closed when Stop is called.
func (w *Writer) Done() <-chan struct{} {
	return w.done
}

// Stop terminates the heartbeat goroutine. Safe to call multiple times.
func (w *Writer) Stop() {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
}

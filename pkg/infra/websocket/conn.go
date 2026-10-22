package websocket

import (
	"context"
	"sync"
	"sync/atomic"

	cws "github.com/coder/websocket"
)

type outboundMessage struct {
	msgType cws.MessageType
	data    []byte
}

// Conn wraps a WebSocket connection with a send queue, lifecycle context,
// and opaque application metadata.
//
// Metadata is for caller-owned session state (user id, room, role, …).
// The SDK never interprets these values.
type Conn struct {
	id   string
	role string
	cfg  Config
	raw  *cws.Conn

	ctx    context.Context
	cancel context.CancelFunc
	sendCh chan outboundMessage

	values sync.Map

	// draining: Send rejects new messages; write loop flushes the queue then closes.
	draining atomic.Bool
	// closed marks force/finished close so Send/Read fail fast.
	closed atomic.Bool
	// wakeDrain is closed once when beginDrain runs so writeLoop wakes immediately.
	wakeDrain chan struct{}
	drainOnce sync.Once

	closeOnce sync.Once
	writeMu   sync.Mutex
}

func newConn(id, role string, raw *cws.Conn, cfg Config) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	cfg = cfg.normalized()
	return &Conn{
		id:        id,
		role:      role,
		cfg:       cfg,
		raw:       raw,
		ctx:       WithConnectionID(ctx, id),
		cancel:    cancel,
		sendCh:    make(chan outboundMessage, cfg.SendQueueSize),
		wakeDrain: make(chan struct{}),
	}
}

// ID returns the SDK-assigned connection identifier.
func (c *Conn) ID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// Context is canceled when the connection is fully closed.
func (c *Conn) Context() context.Context {
	if c == nil {
		return context.Background()
	}
	return c.ctx
}

// Raw exposes the underlying coder/websocket connection for advanced use.
// Prefer Send / Close for normal traffic.
func (c *Conn) Raw() *cws.Conn {
	if c == nil {
		return nil
	}
	return c.raw
}

// Set stores an opaque value on the connection (safe for concurrent use).
func (c *Conn) Set(key string, value any) {
	if c == nil || key == "" {
		return
	}
	c.values.Store(key, value)
}

// Get loads a value previously stored with Set.
func (c *Conn) Get(key string) (any, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	return c.values.Load(key)
}

// Delete removes a metadata entry.
func (c *Conn) Delete(key string) {
	if c == nil || key == "" {
		return
	}
	c.values.Delete(key)
}

// Send enqueues a data frame for the write loop.
//
// With BackpressureBlock (default), Send waits until the queue has space or
// the connection is closed. With BackpressureError, a full queue returns
// ErrSendQueueFull without dropping the message.
func (c *Conn) Send(msgType cws.MessageType, data []byte) error {
	if c == nil {
		return ErrConnNil
	}
	if c.closed.Load() || c.draining.Load() {
		return ErrConnClosed
	}
	msg := outboundMessage{
		msgType: msgType,
		data:    append([]byte(nil), data...),
	}
	if c.cfg.Backpressure == BackpressureError {
		return c.enqueueNonBlocking(msg)
	}
	select {
	case <-c.ctx.Done():
		return ErrConnClosed
	case c.sendCh <- msg:
		return nil
	}
}

// TrySend enqueues without blocking. Always returns ErrSendQueueFull when the
// queue is full. Used by Hub broadcast so one slow peer cannot stall others.
func (c *Conn) TrySend(msgType cws.MessageType, data []byte) error {
	if c == nil {
		return ErrConnNil
	}
	if c.closed.Load() || c.draining.Load() {
		return ErrConnClosed
	}
	return c.enqueueNonBlocking(outboundMessage{
		msgType: msgType,
		data:    append([]byte(nil), data...),
	})
}

func (c *Conn) enqueueNonBlocking(msg outboundMessage) error {
	select {
	case <-c.ctx.Done():
		return ErrConnClosed
	case c.sendCh <- msg:
		return nil
	default:
		return ErrSendQueueFull
	}
}

// SendText enqueues a text frame.
func (c *Conn) SendText(text string) error {
	return c.Send(MessageText, []byte(text))
}

// Read blocks until the next data frame or the connection closes.
//
// When ReadTimeout > 0, each Read is bounded by that duration. Prefer
// ReadTimeout=0 (default after normalize when unset) for push-heavy servers and
// rely on PingInterval for dead-peer detection — a per-read deadline will tear
// down idle connections that still pass ping/pong.
func (c *Conn) Read() (Message, error) {
	if c == nil {
		return Message{}, ErrConnNil
	}
	if c.raw == nil {
		return Message{}, ErrConnClosed
	}
	ctx := c.ctx
	var cancel context.CancelFunc
	if c.cfg.ReadTimeout > 0 {
		ctx, cancel = context.WithTimeout(c.ctx, c.cfg.ReadTimeout)
		defer cancel()
	}
	msgType, data, err := c.raw.Read(ctx)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: msgType, Data: data}, nil
}

// beginDrain stops accepting new Send traffic and asks the write loop to flush
// then close. Used by Server.Shutdown.
func (c *Conn) beginDrain() {
	if c == nil {
		return
	}
	c.draining.Store(true)
	c.drainOnce.Do(func() { close(c.wakeDrain) })
}

func (c *Conn) isDraining() bool {
	return c != nil && c.draining.Load()
}

func (c *Conn) queueLen() int {
	if c == nil {
		return 0
	}
	return len(c.sendCh)
}

// Close force-closes the connection immediately. Outbound queue frames that
// have not been written yet are discarded. Prefer Server.Shutdown for a
// best-effort drain before close.
func (c *Conn) Close() error {
	if c == nil {
		return ErrConnNil
	}
	var closeErr error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.draining.Store(true)
		if c.cancel != nil {
			c.cancel()
		}
		if c.raw != nil {
			if err := c.raw.Close(cws.StatusNormalClosure, "bye"); err != nil {
				closeErr = err
			}
		}
	})
	return closeErr
}

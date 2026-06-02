package websocket

import (
	"context"
	"errors"
	"sync"

	cws "github.com/coder/websocket"
)

type outboundMessage struct {
	msgType cws.MessageType
	data    []byte
}

// Conn wraps websocket.Conn with a send queue and lifecycle context.
type Conn struct {
	id   string
	role string
	cfg  Config
	raw  *cws.Conn

	ctx    context.Context
	cancel context.CancelFunc
	sendCh chan outboundMessage

	closeOnce sync.Once
	writeMu   sync.Mutex
}

func newConn(id, role string, raw *cws.Conn, cfg Config) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	cfg = cfg.normalized()
	return &Conn{
		id:     id,
		role:   role,
		cfg:    cfg,
		raw:    raw,
		ctx:    WithConnectionID(ctx, id),
		cancel: cancel,
		sendCh: make(chan outboundMessage, cfg.SendQueueSize),
	}
}

func (c *Conn) ID() string { return c.id }

func (c *Conn) Context() context.Context { return c.ctx }

func (c *Conn) Raw() *cws.Conn { return c.raw }

func (c *Conn) Send(msgType cws.MessageType, data []byte) error {
	if c == nil {
		return errors.New("websocket: connection is nil")
	}
	msg := outboundMessage{
		msgType: msgType,
		data:    append([]byte(nil), data...),
	}
	if c.cfg.DropOnBackpressure {
		select {
		case c.sendCh <- msg:
			return nil
		default:
			return errors.New("websocket: backpressure queue is full")
		}
	}

	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	case c.sendCh <- msg:
		return nil
	}
}

func (c *Conn) SendText(text string) error {
	return c.Send(MessageText, []byte(text))
}

func (c *Conn) Read() (Message, error) {
	if c == nil {
		return Message{}, errors.New("websocket: connection is nil")
	}
	ctx, cancel := context.WithTimeout(c.ctx, c.cfg.ReadTimeout)
	defer cancel()
	msgType, data, err := c.raw.Read(ctx)
	if err != nil {
		return Message{}, err
	}
	return Message{Type: msgType, Data: data}, nil
}

func (c *Conn) Close() error {
	var closeErr error
	c.closeOnce.Do(func() {
		c.cancel()
		if err := c.raw.Close(cws.StatusNormalClosure, "bye"); err != nil {
			closeErr = err
		}
	})
	return closeErr
}

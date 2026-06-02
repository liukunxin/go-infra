package websocket

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	cws "github.com/coder/websocket"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

type Client struct {
	cfg      ClientConfig
	handlers ClientHandlers

	mu        sync.RWMutex
	conn      *Conn
	closeOnce sync.Once
	closed    chan struct{}
}

func NewClient(cfg ClientConfig, handlers ClientHandlers) *Client {
	cfg = cfg.normalized()
	return &Client{
		cfg:      cfg,
		handlers: handlers,
		closed:   make(chan struct{}),
	}
}

// Run blocks and keeps the websocket connection alive. When Reconnect=true, it
// reconnects automatically until ctx is canceled.
func (c *Client) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return nil
		default:
		}

		err := c.connectAndServe(ctx)
		if err == nil || !c.cfg.Reconnect {
			return err
		}

		recordReconnect(ctx)
		attempt++
		if c.handlers.OnReconnectAttempt != nil {
			c.handlers.OnReconnectAttempt(attempt)
		}
		sleep := c.backoff(attempt)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.closed:
			return nil
		case <-time.After(sleep):
		}
	}
}

func (c *Client) Send(msgType cws.MessageType, data []byte) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return errors.New("websocket: client is not connected")
	}
	return conn.Send(msgType, data)
}

func (c *Client) SendText(text string) error {
	return c.Send(MessageText, []byte(text))
}

func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conn != nil {
			err = c.conn.Close()
			c.conn = nil
		}
	})
	return err
}

func (c *Client) connectAndServe(ctx context.Context) error {
	dialCtx, cancelDial := context.WithTimeout(ctx, c.cfg.ConnectTimeout)
	defer cancelDial()
	rawConn, _, err := cws.Dial(dialCtx, c.cfg.URL, &cws.DialOptions{
		HTTPHeader:      c.cfg.Headers,
		CompressionMode: c.cfg.compressionMode(),
	})
	if err != nil {
		recordError(ctx, "client")
		if c.handlers.OnError != nil {
			c.handlers.OnError(err)
		}
		return err
	}

	conn := newConn("client", "client", rawConn, c.cfg.Config)
	recordConnectionDelta(conn.Context(), 1, "client")
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()

	if c.handlers.OnConnect != nil {
		c.handlers.OnConnect()
	}
	if c.cfg.MaxMessageBytes > 0 {
		rawConn.SetReadLimit(c.cfg.MaxMessageBytes)
	}

	errCh := make(chan error, 2)
	go c.clientWriteLoop(conn, errCh)
	go c.clientReadLoop(conn, errCh)

	err = <-errCh
	err = normalizeCloseErr(err)
	recordConnectionDelta(conn.Context(), -1, "client")
	if c.handlers.OnDisconnect != nil {
		c.handlers.OnDisconnect(err)
	}
	_ = conn.Close()
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
	}
	c.mu.Unlock()
	return err
}

func (c *Client) clientReadLoop(conn *Conn, errCh chan<- error) {
	for {
		msg, err := conn.Read()
		if err != nil {
			errCh <- err
			return
		}
		recordMessage(conn.Context(), "client", "in", int(msg.Type), len(msg.Data))
		if c.handlers.OnMessage != nil {
			c.handlers.OnMessage(msg)
		}
	}
}

func (c *Client) clientWriteLoop(conn *Conn, errCh chan<- error) {
	ticker := time.NewTicker(c.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-conn.Context().Done():
			errCh <- conn.Context().Err()
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(conn.Context(), c.cfg.WriteTimeout)
			err := conn.Raw().Ping(pingCtx)
			cancel()
			if err != nil {
				errCh <- err
				return
			}
		case msg, ok := <-conn.sendCh:
			if !ok {
				errCh <- nil
				return
			}
			if err := c.write(conn, msg.msgType, msg.data); err != nil {
				errCh <- err
				return
			}
			recordMessage(conn.Context(), "client", "out", int(msg.msgType), len(msg.data))
		}
	}
}

func (c *Client) write(conn *Conn, msgType cws.MessageType, payload []byte) error {
	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()

	ctx, cancel := context.WithTimeout(conn.Context(), c.cfg.WriteTimeout)
	defer cancel()
	if err := conn.Raw().Write(ctx, msgType, payload); err != nil {
		recordError(conn.Context(), "client")
		log.WithContext(conn.Context()).Warn("websocket client write failed, err=%v", err)
		return err
	}
	return nil
}

func (c *Client) backoff(attempt int) time.Duration {
	base := c.cfg.ReconnectBaseBackoff
	maxBackoff := c.cfg.ReconnectMaxBackoff
	for i := 1; i < attempt; i++ {
		base *= 2
		if base >= maxBackoff {
			base = maxBackoff
			break
		}
	}
	if c.cfg.BackoffJitterRatio <= 0 {
		return base
	}
	factor := 1 + ((rand.Float64()*2 - 1) * c.cfg.BackoffJitterRatio)
	if factor < 0 {
		factor = 0
	}
	return time.Duration(float64(base) * factor)
}

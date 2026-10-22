package websocket

import (
	"errors"

	cws "github.com/coder/websocket"
)

var (
	// ErrSendQueueFull is returned when a non-blocking enqueue finds no free slot.
	// The SDK never silently drops the caller's message.
	ErrSendQueueFull = errors.New("websocket: send queue full")

	// ErrServerClosed is returned when the server has begun Shutdown and rejects
	// new upgrades (also used as the HTTP 503 body).
	ErrServerClosed = errors.New("websocket: server closed")

	// ErrConnNil is returned when a method is called on a nil *Conn.
	ErrConnNil = errors.New("websocket: connection is nil")

	// ErrClientNotConnected is returned by Client.Send when no dial is active.
	ErrClientNotConnected = errors.New("websocket: client is not connected")

	// ErrConnClosed is returned by Send after the connection has started closing
	// (graceful drain or force close).
	ErrConnClosed = errors.New("websocket: connection closed")
)

func normalizeCloseErr(err error) error {
	if err == nil {
		return nil
	}
	code := cws.CloseStatus(err)
	switch code {
	case cws.StatusNormalClosure, cws.StatusGoingAway:
		return nil
	default:
		return err
	}
}

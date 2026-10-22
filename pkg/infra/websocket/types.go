package websocket

import (
	"context"
	"net/http"

	cws "github.com/coder/websocket"
)

const (
	MessageText   = cws.MessageText
	MessageBinary = cws.MessageBinary
)

// Message is one WebSocket data frame delivered to application handlers.
type Message struct {
	Type cws.MessageType
	Data []byte
}

// HandshakeAuth runs after Origin checks and before Accept.
// Return a non-nil error to reject the upgrade with HTTP 401.
type HandshakeAuth func(r *http.Request) error

// Handlers are server-side lifecycle callbacks.
//
// Boundary: handlers receive raw frames. Application codecs, ACK protocols,
// leases, and room semantics stay outside this package.
type Handlers struct {
	OnConnect func(*Conn)
	OnMessage func(*Conn, Message)
	OnClose   func(*Conn, error)
	OnError   func(*Conn, error)

	// OnTick runs on each PingInterval, on the connection write loop, before
	// the WebSocket ping. Keep it short (renew a lease, touch a session).
	// Closing the connection from OnTick is allowed; the loop then exits.
	// A nil OnTick is a no-op — the SDK still sends pings.
	OnTick func(*Conn)
}

// ClientHandlers are client-side lifecycle callbacks.
type ClientHandlers struct {
	OnConnect          func()
	OnMessage          func(Message)
	OnDisconnect       func(error)
	OnReconnectAttempt func(int)
	OnError            func(error)

	// OnTick mirrors Handlers.OnTick for the client write loop.
	OnTick func()
}

type contextKey string

const connectionIDKey contextKey = "websocket_connection_id"

// WithConnectionID stores the connection id in ctx (used by metrics and logging).
func WithConnectionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, connectionIDKey, id)
}

// ConnectionIDFromContext returns the id set by WithConnectionID, or "".
func ConnectionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(connectionIDKey).(string)
	return v
}

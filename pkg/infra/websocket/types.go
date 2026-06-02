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

type Message struct {
	Type cws.MessageType
	Data []byte
}

type HandshakeAuth func(r *http.Request) error

type Handlers struct {
	OnConnect func(*Conn)
	OnMessage func(*Conn, Message)
	OnClose   func(*Conn, error)
	OnError   func(*Conn, error)
}

type ClientHandlers struct {
	OnConnect          func()
	OnMessage          func(Message)
	OnDisconnect       func(error)
	OnReconnectAttempt func(int)
	OnError            func(error)
}

type contextKey string

const connectionIDKey contextKey = "websocket_connection_id"

func WithConnectionID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, connectionIDKey, id)
}

func ConnectionIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(connectionIDKey).(string)
	return v
}

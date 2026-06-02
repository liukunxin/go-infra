package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	cws "github.com/coder/websocket"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

type Server struct {
	cfg         Config
	handlers    Handlers
	auth        HandshakeAuth
	checkOrigin func(*http.Request) bool
	connSeq     uint64
}

func NewServer(cfg Config, handlers Handlers, auth HandshakeAuth, checkOrigin func(*http.Request) bool) *Server {
	cfg = cfg.normalized()
	return &Server{
		cfg:         cfg,
		handlers:    handlers,
		auth:        auth,
		checkOrigin: checkOrigin,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.checkOrigin != nil && !s.checkOrigin(r) {
		http.Error(w, "forbidden origin", http.StatusForbidden)
		return
	}
	if s.auth != nil {
		if err := s.auth(r); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.HandshakeTimeout)
	defer cancel()

	rawConn, err := cws.Accept(w, r.WithContext(ctx), &cws.AcceptOptions{
		CompressionMode: s.cfg.compressionMode(),
	})
	if err != nil {
		return
	}

	connID := fmt.Sprintf("ws-%d", atomic.AddUint64(&s.connSeq, 1))
	conn := newConn(connID, "server", rawConn, s.cfg)
	recordConnectionDelta(conn.Context(), 1, "server")
	if s.handlers.OnConnect != nil {
		s.handlers.OnConnect(conn)
	}

	if s.cfg.MaxMessageBytes > 0 {
		rawConn.SetReadLimit(s.cfg.MaxMessageBytes)
	}

	go s.writeLoop(conn)
	s.readLoop(conn)
}

func (s *Server) readLoop(conn *Conn) {
	var closeErr error
	defer func() {
		recordConnectionDelta(conn.Context(), -1, "server")
		if s.handlers.OnClose != nil {
			s.handlers.OnClose(conn, closeErr)
		}
		_ = conn.Close()
	}()

	for {
		msg, err := conn.Read()
		if err != nil {
			closeErr = normalizeCloseErr(err)
			return
		}

		recordMessage(conn.Context(), "server", "in", int(msg.Type), len(msg.Data))
		if s.handlers.OnMessage != nil {
			s.handlers.OnMessage(conn, msg)
		}
	}
}

func (s *Server) writeLoop(conn *Conn) {
	ticker := time.NewTicker(s.cfg.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-conn.Context().Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(conn.Context(), s.cfg.WriteTimeout)
			err := conn.Raw().Ping(pingCtx)
			cancel()
			if err != nil {
				recordError(conn.Context(), "server")
				if s.handlers.OnError != nil {
					s.handlers.OnError(conn, err)
				}
				_ = conn.Close()
				return
			}
		case msg, ok := <-conn.sendCh:
			if !ok {
				return
			}
			if err := s.write(conn, msg.msgType, msg.data); err != nil {
				recordError(conn.Context(), "server")
				if s.handlers.OnError != nil {
					s.handlers.OnError(conn, err)
				}
				_ = conn.Close()
				return
			}
			recordMessage(conn.Context(), "server", "out", int(msg.msgType), len(msg.data))
		}
	}
}

func (s *Server) write(conn *Conn, msgType cws.MessageType, payload []byte) error {
	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()

	ctx, cancel := context.WithTimeout(conn.Context(), s.cfg.WriteTimeout)
	defer cancel()
	if err := conn.Raw().Write(ctx, msgType, payload); err != nil {
		log.WithContext(conn.Context()).Warn("websocket server write failed, conn=%s err=%v", conn.ID(), err)
		return err
	}
	return nil
}

package websocket

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	cws "github.com/coder/websocket"
	"github.com/liukunxin/go-infra/pkg/base/log"
)

// Server is an http.Handler that upgrades requests and runs connection loops.
type Server struct {
	cfg         Config
	handlers    Handlers
	auth        HandshakeAuth
	checkOrigin func(*http.Request) bool

	connSeq uint64

	mu       sync.Mutex
	conns    map[string]*Conn
	shutting atomic.Bool
}

// NewServer constructs a Server. checkOrigin may be nil (no Origin check).
// auth may be nil (no handshake auth); prefer authenticating in middleware
// when the host framework already owns identity.
func NewServer(cfg Config, handlers Handlers, auth HandshakeAuth, checkOrigin func(*http.Request) bool) *Server {
	cfg = cfg.normalized()
	return &Server{
		cfg:         cfg,
		handlers:    handlers,
		auth:        auth,
		checkOrigin: checkOrigin,
		conns:       make(map[string]*Conn),
	}
}

// ServeHTTP upgrades the connection and blocks until it closes.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.shutting.Load() {
		http.Error(w, ErrServerClosed.Error(), http.StatusServiceUnavailable)
		return
	}
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
	if !s.tryTrack(conn) {
		_ = rawConn.Close(cws.StatusGoingAway, "server shutting down")
		return
	}
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

// Shutdown stops accepting new upgrades, best-effort drains outbound queues,
// force-closes remaining connections when ctx expires, and waits until all
// tracked connections finish (or ctx is done).
//
// Drain semantics: each connection rejects new Send traffic, the write loop
// flushes frames already queued, then closes the socket. It is not a hard
// guarantee that every enqueued frame reaches the wire if the peer is stuck
// or ctx times out.
func (s *Server) Shutdown(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	s.mu.Lock()
	s.shutting.Store(true)
	list := make([]*Conn, 0, len(s.conns))
	for _, c := range s.conns {
		list = append(list, c)
	}
	s.mu.Unlock()

	for _, c := range list {
		c.beginDrain()
	}

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		n := len(s.conns)
		s.mu.Unlock()
		if n == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			s.mu.Lock()
			left := make([]*Conn, 0, len(s.conns))
			for _, c := range s.conns {
				left = append(left, c)
			}
			s.mu.Unlock()
			for _, c := range left {
				_ = c.Close()
			}
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ActiveConnections returns the number of connections still tracked.
func (s *Server) ActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.conns)
}

// tryTrack registers conn unless Shutdown has started. Must be used instead of
// a bare track after Accept to close the race where Shutdown snapshots an empty
// map and returns while a new connection is still being registered.
func (s *Server) tryTrack(conn *Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shutting.Load() {
		return false
	}
	s.conns[conn.ID()] = conn
	return true
}

func (s *Server) untrack(conn *Conn) {
	s.mu.Lock()
	delete(s.conns, conn.ID())
	s.mu.Unlock()
}

func (s *Server) readLoop(conn *Conn) {
	var closeErr error
	defer func() {
		recordConnectionDelta(conn.Context(), -1, "server")
		if s.handlers.OnClose != nil {
			s.handlers.OnClose(conn, closeErr)
		}
		_ = conn.Close()
		s.untrack(conn)
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

	finishIfDrained := func() bool {
		if conn.isDraining() && conn.queueLen() == 0 {
			_ = conn.Close()
			return true
		}
		return false
	}

	for {
		select {
		case <-conn.Context().Done():
			return
		case <-conn.wakeDrain:
			if finishIfDrained() {
				return
			}
		case <-ticker.C:
			if conn.isDraining() {
				if finishIfDrained() {
					return
				}
				continue
			}
			if s.handlers.OnTick != nil {
				s.handlers.OnTick(conn)
			}
			if conn.Context().Err() != nil {
				return
			}
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
			if finishIfDrained() {
				return
			}
		}
	}
}

func (s *Server) write(conn *Conn, msgType cws.MessageType, payload []byte) error {
	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()

	ctx, cancel := context.WithTimeout(conn.Context(), s.cfg.WriteTimeout)
	defer cancel()
	if err := conn.Raw().Write(ctx, msgType, payload); err != nil {
		log.WithContext(conn.Context()).Warnf("websocket server write failed, conn=%s err=%v", conn.ID(), err)
		return err
	}
	return nil
}

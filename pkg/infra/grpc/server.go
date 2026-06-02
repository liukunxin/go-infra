package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Registrar registers protobuf services into the server.
type Registrar func(*ggrpc.Server)

// Server wraps grpc.Server with startup and graceful shutdown helpers.
type Server struct {
	cfg      ServerConfig
	listener net.Listener
	server   *ggrpc.Server
}

// NewServer creates a configured gRPC server and binds the configured address.
func NewServer(cfg ServerConfig, register Registrar) (*Server, error) {
	cfg = cfg.normalized()
	if register == nil {
		return nil, fmt.Errorf("grpc: register callback is required")
	}

	opts := make([]ggrpc.ServerOption, 0, 16)
	opts = append(opts, ggrpc.MaxRecvMsgSize(cfg.MaxRecvMessageSize))
	opts = append(opts, ggrpc.MaxSendMsgSize(cfg.MaxSendMessageSize))
	opts = append(opts, ggrpc.KeepaliveParams(cfg.keepaliveParams()))

	if cfg.TLSConfig != nil {
		opts = append(opts, ggrpc.Creds(credentials.NewTLS(cfg.TLSConfig)))
	}

	unaryInterceptors := make([]ggrpc.UnaryServerInterceptor, 0, len(cfg.UnaryInterceptors)+1)
	unaryInterceptors = append(unaryInterceptors, unaryServerInterceptor(cfg))
	unaryInterceptors = append(unaryInterceptors, cfg.UnaryInterceptors...)
	opts = append(opts, ggrpc.ChainUnaryInterceptor(unaryInterceptors...))

	streamInterceptors := make([]ggrpc.StreamServerInterceptor, 0, len(cfg.StreamInterceptors)+1)
	streamInterceptors = append(streamInterceptors, streamServerInterceptor(cfg))
	streamInterceptors = append(streamInterceptors, cfg.StreamInterceptors...)
	opts = append(opts, ggrpc.ChainStreamInterceptor(streamInterceptors...))

	opts = append(opts, cfg.ExtraServerOptions...)
	srv := ggrpc.NewServer(opts...)
	register(srv)

	if cfg.RegisterHealthService {
		RegisterHealthServer(srv)
	}
	if cfg.EnableReflection {
		reflection.Register(srv)
	}

	ln, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		return nil, fmt.Errorf("grpc: listen %s failed: %w", cfg.Address, err)
	}
	return &Server{
		cfg:      cfg,
		listener: ln,
		server:   srv,
	}, nil
}

func (s *Server) Start() error {
	if s == nil || s.server == nil {
		return fmt.Errorf("grpc: server is nil")
	}
	return s.server.Serve(s.listener)
}

func (s *Server) GracefulStop(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.server.GracefulStop()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		s.server.Stop()
		return ctx.Err()
	}
}

func (s *Server) Stop() {
	if s == nil || s.server == nil {
		return
	}
	s.server.Stop()
}

func (s *Server) Addr() net.Addr {
	if s == nil || s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

func (s *Server) GRPCServer() *ggrpc.Server {
	if s == nil {
		return nil
	}
	return s.server
}

func ShutdownWithTimeout(s *Server, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.GracefulStop(ctx)
}

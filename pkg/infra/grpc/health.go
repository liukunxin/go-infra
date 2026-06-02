package grpc

import (
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"

	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/health"
)

// RegisterHealthServer registers grpc health service and returns the instance.
func RegisterHealthServer(server *ggrpc.Server) *health.Server {
	hs := health.NewServer()
	healthgrpc.RegisterHealthServer(server, hs)
	return hs
}

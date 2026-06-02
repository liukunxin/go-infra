package grpc

import (
	"encoding/json"
	"fmt"

	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClientConn creates a gRPC client connection with unified defaults.
func NewClientConn(cfg ClientConfig) (*ggrpc.ClientConn, error) {
	cfg = cfg.normalized()
	if cfg.Target == "" {
		return nil, fmt.Errorf("grpc: target is required")
	}

	dialOpts := make([]ggrpc.DialOption, 0, 16)
	if cfg.Insecure || cfg.TLSConfig == nil {
		dialOpts = append(dialOpts, ggrpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, ggrpc.WithTransportCredentials(credentials.NewTLS(cfg.TLSConfig)))
	}

	dialOpts = append(dialOpts, ggrpc.WithKeepaliveParams(cfg.keepaliveParams()))
	if cfg.InitialWindowSize > 0 {
		dialOpts = append(dialOpts, ggrpc.WithInitialWindowSize(cfg.InitialWindowSize))
	}
	if cfg.InitialConnWindowSize > 0 {
		dialOpts = append(dialOpts, ggrpc.WithInitialConnWindowSize(cfg.InitialConnWindowSize))
	}

	unaryInterceptors := make([]ggrpc.UnaryClientInterceptor, 0, len(cfg.UnaryInterceptors)+2)
	unaryInterceptors = append(unaryInterceptors, unaryClientInterceptor(cfg))
	unaryInterceptors = append(unaryInterceptors, cfg.UnaryInterceptors...)
	dialOpts = append(dialOpts, ggrpc.WithChainUnaryInterceptor(unaryInterceptors...))

	streamInterceptors := make([]ggrpc.StreamClientInterceptor, 0, len(cfg.StreamInterceptors)+2)
	streamInterceptors = append(streamInterceptors, streamClientInterceptor(cfg))
	streamInterceptors = append(streamInterceptors, cfg.StreamInterceptors...)
	dialOpts = append(dialOpts, ggrpc.WithChainStreamInterceptor(streamInterceptors...))

	callOpts := make([]ggrpc.CallOption, 0, 2)
	if cfg.MaxCallRecvMessageSize > 0 {
		callOpts = append(callOpts, ggrpc.MaxCallRecvMsgSize(cfg.MaxCallRecvMessageSize))
	}
	if cfg.MaxCallSendMessageSize > 0 {
		callOpts = append(callOpts, ggrpc.MaxCallSendMsgSize(cfg.MaxCallSendMessageSize))
	}
	if len(callOpts) > 0 {
		dialOpts = append(dialOpts, ggrpc.WithDefaultCallOptions(callOpts...))
	}

	if cfg.EnableRetry {
		if sc := buildRetryServiceConfig(cfg); sc != "" {
			dialOpts = append(dialOpts, ggrpc.WithDefaultServiceConfig(sc))
		}
	}

	dialOpts = append(dialOpts, cfg.ExtraDialOptions...)
	return ggrpc.Dial(cfg.Target, dialOpts...)
}

func buildRetryServiceConfig(cfg ClientConfig) string {
	codeNames := make([]string, 0, len(cfg.RetryableStatusCode))
	for _, c := range cfg.RetryableStatusCode {
		codeNames = append(codeNames, c.String())
	}

	// gRPC retry policy is configured in service config.
	policy := map[string]any{
		"methodConfig": []map[string]any{
			{
				"name": []map[string]string{{"service": ""}},
				"retryPolicy": map[string]any{
					"maxAttempts":          cfg.MaxRetryAttempts,
					"initialBackoff":       cfg.RetryBackoffBase.String(),
					"maxBackoff":           cfg.RetryBackoffMax.String(),
					"backoffMultiplier":    2.0,
					"retryableStatusCodes": codeNames,
				},
			},
		},
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		return ""
	}
	return string(raw)
}

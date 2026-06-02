package grpc

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// SetOutgoingMetadata merges key-values into gRPC outgoing metadata.
func SetOutgoingMetadata(ctx context.Context, kv map[string]string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	for k, v := range kv {
		md.Set(k, v)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

// GetIncomingMetadata returns the first value by key from incoming metadata.
func GetIncomingMetadata(ctx context.Context, key string) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

// InjectTraceMetadata injects current trace context into outgoing metadata.
func InjectTraceMetadata(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	carrier := metadataCarrier(md)
	otel.GetTextMapPropagator().Inject(ctx, carrier)
	return metadata.NewOutgoingContext(ctx, metadata.MD(carrier))
}

func extractContextFromIncoming(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	md, _ := metadata.FromIncomingContext(ctx)
	return otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md.Copy()))
}

func injectContextToOutgoing(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	otel.GetTextMapPropagator().Inject(ctx, metadataCarrier(md))
	return metadata.NewOutgoingContext(ctx, md)
}

type metadataCarrier metadata.MD

func (c metadataCarrier) Get(key string) string {
	v := metadata.MD(c).Get(key)
	if len(v) == 0 {
		return ""
	}
	return v[0]
}

func (c metadataCarrier) Set(key, value string) {
	metadata.MD(c).Set(key, value)
}

func (c metadataCarrier) Keys() []string {
	md := metadata.MD(c)
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	return keys
}

// UnaryClientTracePropagator injects trace context into outgoing metadata.
func UnaryClientTracePropagator() ggrpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *ggrpc.ClientConn, invoker ggrpc.UnaryInvoker, opts ...ggrpc.CallOption) error {
		ctx = injectContextToOutgoing(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientTracePropagator injects trace context into outgoing metadata.
func StreamClientTracePropagator() ggrpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *ggrpc.StreamDesc, cc *ggrpc.ClientConn, method string, streamer ggrpc.Streamer, opts ...ggrpc.CallOption) (ggrpc.ClientStream, error) {
		ctx = injectContextToOutgoing(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

var _ propagation.TextMapCarrier = metadataCarrier{}

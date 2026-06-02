package grpc

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/liukunxin/go-infra/pkg/base/log"
	"github.com/liukunxin/go-infra/pkg/infra/traffic"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcMetricInitOnce sync.Once

	grpcServerRequestsTotal metric.Int64Counter
	grpcServerDuration      metric.Float64Histogram
	grpcClientRequestsTotal metric.Int64Counter
	grpcClientDuration      metric.Float64Histogram
)

func ensureGRPCMetrics() {
	grpcMetricInitOnce.Do(func() {
		meter := otel.Meter("grpc")
		grpcServerRequestsTotal, _ = meter.Int64Counter("grpc_server_requests_total")
		grpcServerDuration, _ = meter.Float64Histogram("grpc_server_duration_seconds", metric.WithUnit("s"))
		grpcClientRequestsTotal, _ = meter.Int64Counter("grpc_client_requests_total")
		grpcClientDuration, _ = meter.Float64Histogram("grpc_client_duration_seconds", metric.WithUnit("s"))
	})
}

func unaryServerInterceptor(cfg ServerConfig) grpc.UnaryServerInterceptor {
	cfg = cfg.normalized()
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		ctx = extractContextFromIncoming(ctx)
		ctx, span := otel.Tracer("go-infra/grpc-server").Start(ctx, info.FullMethod)
		defer span.End()

		resource := buildServerTrafficResource(cfg.TrafficResourcePrefix, info.FullMethod)
		pass, blockErr := tryTrafficPass(cfg.EnableTrafficInterceptor, resource, traffic.TrafficTypeInbound)
		if blockErr != nil {
			err := status.Errorf(httpStatusToCode(429), blockErr.BlockMsg())
			span.RecordError(err)
			span.SetStatus(codes.Error, blockErr.BlockMsg())
			recordServerMetric(ctx, info.FullMethod, time.Since(start), err)
			return nil, err
		}
		if pass != nil {
			defer pass.Done()
		}

		resp, err := handler(ctx, req)
		if err != nil {
			if pass != nil {
				pass.Error(err)
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.WithContext(ctx).Error("grpc unary server failed, method=%s err=%v", info.FullMethod, err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		recordServerMetric(ctx, info.FullMethod, time.Since(start), err)
		return resp, err
	}
}

func unaryClientInterceptor(cfg ClientConfig) grpc.UnaryClientInterceptor {
	cfg = cfg.normalized()
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()
		ctx, cancel := applyDefaultTimeout(ctx, cfg.DefaultTimeout)
		defer cancel()
		ctx = injectContextToOutgoing(ctx)
		ctx, span := otel.Tracer("go-infra/grpc-client").Start(ctx, method)
		defer span.End()

		resource := buildClientTrafficResource(cfg.TrafficResource, method)
		pass, blockErr := tryTrafficPass(cfg.EnableTrafficInterceptor, resource, traffic.TrafficTypeOutbound)
		if blockErr != nil {
			err := status.Errorf(httpStatusToCode(429), blockErr.BlockMsg())
			span.RecordError(err)
			span.SetStatus(codes.Error, blockErr.BlockMsg())
			recordClientMetric(ctx, method, time.Since(start), err)
			return err
		}
		if pass != nil {
			defer pass.Done()
		}

		err := invoker(ctx, method, req, reply, cc, opts...)
		if err != nil {
			if pass != nil {
				pass.Error(err)
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			log.WithContext(ctx).Error("grpc unary client failed, method=%s err=%v", method, err)
		} else {
			span.SetStatus(codes.Ok, "")
		}
		recordClientMetric(ctx, method, time.Since(start), err)
		return err
	}
}

func streamServerInterceptor(cfg ServerConfig) grpc.StreamServerInterceptor {
	cfg = cfg.normalized()
	return func(srv any, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := extractContextFromIncoming(ss.Context())
		ctx, span := otel.Tracer("go-infra/grpc-server").Start(ctx, info.FullMethod)
		defer span.End()

		resource := buildServerTrafficResource(cfg.TrafficResourcePrefix, info.FullMethod)
		pass, blockErr := tryTrafficPass(cfg.EnableTrafficInterceptor, resource, traffic.TrafficTypeInbound)
		if blockErr != nil {
			err := status.Errorf(httpStatusToCode(429), blockErr.BlockMsg())
			span.RecordError(err)
			span.SetStatus(codes.Error, blockErr.BlockMsg())
			recordServerMetric(ctx, info.FullMethod, time.Since(start), err)
			return err
		}
		if pass != nil {
			defer pass.Done()
		}

		err := handler(srv, &wrappedServerStream{ServerStream: ss, ctx: ctx})
		if err != nil {
			if pass != nil {
				pass.Error(err)
			}
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else {
			span.SetStatus(codes.Ok, "")
		}
		recordServerMetric(ctx, info.FullMethod, time.Since(start), err)
		return err
	}
}

func streamClientInterceptor(cfg ClientConfig) grpc.StreamClientInterceptor {
	cfg = cfg.normalized()
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Stream call timeout should be controlled by caller context to avoid
		// canceling long-lived streams unexpectedly.
		if ctx == nil {
			ctx = context.Background()
		}
		ctx = injectContextToOutgoing(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

func recordServerMetric(ctx context.Context, method string, d time.Duration, err error) {
	ensureGRPCMetrics()
	code := status.Code(err).String()
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("code", code),
	}
	grpcServerRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	grpcServerDuration.Record(ctx, d.Seconds(), metric.WithAttributes(attrs...))
}

func recordClientMetric(ctx context.Context, method string, d time.Duration, err error) {
	ensureGRPCMetrics()
	code := status.Code(err).String()
	attrs := []attribute.KeyValue{
		attribute.String("method", method),
		attribute.String("code", code),
	}
	grpcClientRequestsTotal.Add(ctx, 1, metric.WithAttributes(attrs...))
	grpcClientDuration.Record(ctx, d.Seconds(), metric.WithAttributes(attrs...))
}

func applyDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, has := ctx.Deadline(); has || timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func buildServerTrafficResource(prefix, fullMethod string) string {
	if prefix == "" {
		return "grpc:server:" + fullMethod
	}
	return strings.TrimSuffix(prefix, ":") + ":" + fullMethod
}

func buildClientTrafficResource(resource, fullMethod string) string {
	if resource != "" {
		return resource
	}
	return "grpc:client:" + fullMethod
}

func tryTrafficPass(enabled bool, resource string, typ traffic.TrafficType) (traffic.Pass, traffic.BlockError) {
	if !enabled {
		return nil, nil
	}
	return traffic.GetController().TryPass(
		resource,
		traffic.NewTryPassOptions().WithTrafficType(typ),
	)
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *wrappedServerStream) Context() context.Context { return s.ctx }

# gRPC - 统一客户端/服务端封装

`pkg/infra/grpc` 面向“默认好用”，提供开箱即用的 gRPC 统一能力：

- Client / Server 工厂（默认超时、keepalive、基础重试）
- 内置 unary + stream 拦截器（trace、metrics）
- 可选 traffic 集成（限流/熔断）
- metadata 透传工具
- HTTP 与 gRPC 错误码映射
- health service 注册

底层依赖使用 gRPC 官方实现 `google.golang.org/grpc`。

## 1. 服务端接入（最小示例）

```go
import (
	"context"
	"time"

	infraGrpc "github.com/liukunxin/go-infra/pkg/infra/grpc"
	pb "your/proto/package"
	ggrpc "google.golang.org/grpc"
)

func startGRPC() (*infraGrpc.Server, error) {
	return infraGrpc.NewServer(
		infraGrpc.ServerConfig{
			Address:               ":9090",
			RegisterHealthService: true,
			EnableReflection:      true,
		},
		func(gs *ggrpc.Server) {
			pb.RegisterUserServiceServer(gs, &UserService{})
		},
	)
}

func stopGRPC(srv *infraGrpc.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return srv.GracefulStop(ctx)
}
```

## 2. 客户端接入（最小示例）

```go
conn, err := infraGrpc.NewClientConn(infraGrpc.ClientConfig{
	Target:         "127.0.0.1:9090",
	Insecure:       true,
	DefaultTimeout: 2 * time.Second,
	EnableRetry:    true,
})
if err != nil {
	return err
}
defer conn.Close()

cli := pb.NewUserServiceClient(conn)
resp, err := cli.GetUser(context.Background(), &pb.GetUserReq{Id: 1})
_ = resp
```

## 3. 启用 traffic（可选）

```go
// 客户端或服务端都可单独开启
cfg := infraGrpc.ClientConfig{
	Target:                   "127.0.0.1:9090",
	Insecure:                 true,
	EnableTrafficInterceptor: true,
	TrafficResource:          "user-service",
}
```

## 4. metadata 透传

```go
ctx := context.Background()
ctx = infraGrpc.SetOutgoingMetadata(ctx, map[string]string{
	"x-tenant-id": "tenant-a",
})
ctx = infraGrpc.InjectTraceMetadata(ctx)
```

## 5. 错误码映射

```go
grpcCode := infraGrpc.GRPCCodeFromHTTPStatus(429)          // codes.ResourceExhausted
httpCode := infraGrpc.HTTPStatusFromGRPCCode(grpcCode)     // 429
```

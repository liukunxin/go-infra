package grpc

import (
	"crypto/tls"
	"time"

	ggrpc "google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
)

// ClientConfig defines client-side gRPC defaults.
type ClientConfig struct {
	Target string `yaml:"target" json:"target"`

	// Transport
	Insecure  bool        `yaml:"insecure" json:"insecure"`
	TLSConfig *tls.Config `yaml:"-" json:"-"`

	// Resilience
	DefaultTimeout      time.Duration `yaml:"default_timeout" json:"default_timeout"`
	EnableRetry         bool          `yaml:"enable_retry" json:"enable_retry"`
	MaxRetryAttempts    int           `yaml:"max_retry_attempts" json:"max_retry_attempts"`
	RetryBackoffBase    time.Duration `yaml:"retry_backoff_base" json:"retry_backoff_base"`
	RetryBackoffMax     time.Duration `yaml:"retry_backoff_max" json:"retry_backoff_max"`
	RetryableStatusCode []codes.Code  `yaml:"-" json:"-"`

	// Keepalive
	KeepaliveTime             time.Duration `yaml:"keepalive_time" json:"keepalive_time"`
	KeepaliveTimeout          time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`
	PermitWithoutStream       bool          `yaml:"permit_without_stream" json:"permit_without_stream"`
	InitialWindowSize         int32         `yaml:"initial_window_size" json:"initial_window_size"`
	InitialConnWindowSize     int32         `yaml:"initial_conn_window_size" json:"initial_conn_window_size"`
	MaxCallRecvMessageSize    int           `yaml:"max_call_recv_message_size" json:"max_call_recv_message_size"`
	MaxCallSendMessageSize    int           `yaml:"max_call_send_message_size" json:"max_call_send_message_size"`
	EnableTrafficInterceptor  bool          `yaml:"enable_traffic_interceptor" json:"enable_traffic_interceptor"`
	TrafficResource           string        `yaml:"traffic_resource" json:"traffic_resource"`
	UnaryInterceptors         []ggrpc.UnaryClientInterceptor
	StreamInterceptors        []ggrpc.StreamClientInterceptor
	ExtraDialOptions          []ggrpc.DialOption
}

func (c ClientConfig) normalized() ClientConfig {
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 3 * time.Second
	}
	if c.MaxRetryAttempts <= 0 {
		c.MaxRetryAttempts = 3
	}
	if c.RetryBackoffBase <= 0 {
		c.RetryBackoffBase = 100 * time.Millisecond
	}
	if c.RetryBackoffMax <= 0 {
		c.RetryBackoffMax = 2 * time.Second
	}
	if c.KeepaliveTime <= 0 {
		c.KeepaliveTime = 30 * time.Second
	}
	if c.KeepaliveTimeout <= 0 {
		c.KeepaliveTimeout = 10 * time.Second
	}
	if len(c.RetryableStatusCode) == 0 {
		c.RetryableStatusCode = []codes.Code{
			codes.Unavailable,
			codes.DeadlineExceeded,
			codes.ResourceExhausted,
		}
	}
	return c
}

func (c ClientConfig) keepaliveParams() keepalive.ClientParameters {
	c = c.normalized()
	return keepalive.ClientParameters{
		Time:                c.KeepaliveTime,
		Timeout:             c.KeepaliveTimeout,
		PermitWithoutStream: c.PermitWithoutStream,
	}
}

// ServerConfig defines server-side gRPC defaults.
type ServerConfig struct {
	Address string `yaml:"address" json:"address"`

	// Transport
	TLSConfig *tls.Config `yaml:"-" json:"-"`

	// Limits
	MaxRecvMessageSize int `yaml:"max_recv_message_size" json:"max_recv_message_size"`
	MaxSendMessageSize int `yaml:"max_send_message_size" json:"max_send_message_size"`

	// Keepalive
	MaxConnectionIdle     time.Duration `yaml:"max_connection_idle" json:"max_connection_idle"`
	MaxConnectionAge      time.Duration `yaml:"max_connection_age" json:"max_connection_age"`
	MaxConnectionAgeGrace time.Duration `yaml:"max_connection_age_grace" json:"max_connection_age_grace"`
	Time                  time.Duration `yaml:"keepalive_time" json:"keepalive_time"`
	Timeout               time.Duration `yaml:"keepalive_timeout" json:"keepalive_timeout"`

	EnableTrafficInterceptor bool `yaml:"enable_traffic_interceptor" json:"enable_traffic_interceptor"`
	TrafficResourcePrefix    string `yaml:"traffic_resource_prefix" json:"traffic_resource_prefix"`
	EnableReflection         bool   `yaml:"enable_reflection" json:"enable_reflection"`
	RegisterHealthService    bool   `yaml:"register_health_service" json:"register_health_service"`

	UnaryInterceptors  []ggrpc.UnaryServerInterceptor
	StreamInterceptors []ggrpc.StreamServerInterceptor
	ExtraServerOptions []ggrpc.ServerOption
}

func (c ServerConfig) normalized() ServerConfig {
	if c.Address == "" {
		c.Address = ":9090"
	}
	if c.MaxRecvMessageSize <= 0 {
		c.MaxRecvMessageSize = 4 << 20
	}
	if c.MaxSendMessageSize <= 0 {
		c.MaxSendMessageSize = 4 << 20
	}
	if c.Time <= 0 {
		c.Time = 2 * time.Hour
	}
	if c.Timeout <= 0 {
		c.Timeout = 20 * time.Second
	}
	return c
}

func (c ServerConfig) keepaliveParams() keepalive.ServerParameters {
	c = c.normalized()
	return keepalive.ServerParameters{
		MaxConnectionIdle:     c.MaxConnectionIdle,
		MaxConnectionAge:      c.MaxConnectionAge,
		MaxConnectionAgeGrace: c.MaxConnectionAgeGrace,
		Time:                  c.Time,
		Timeout:               c.Timeout,
	}
}

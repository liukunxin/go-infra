package websocket

import (
	"net/http"
	"time"

	cws "github.com/coder/websocket"
)

// BackpressurePolicy controls Conn.Send when the outbound queue is full.
type BackpressurePolicy int

const (
	// BackpressureBlock waits until a slot is free or the connection is closed.
	// This is the default and the safest choice for point-to-point Send.
	BackpressureBlock BackpressurePolicy = iota
	// BackpressureError returns ErrSendQueueFull immediately when the queue is full.
	// Prefer this when the application can signal the peer. The SDK never
	// silently drops messages on Send. Hub broadcast always uses non-blocking
	// enqueue regardless of this setting.
	BackpressureError
)

// Config holds shared server/client transport settings.
//
// This package owns connection lifecycle and byte transport only.
// Application protocols (framing, ACK, leases, rooms) belong in the caller.
type Config struct {
	HandshakeTimeout time.Duration `yaml:"handshake_timeout" json:"handshake_timeout"`
	// ReadTimeout bounds a single Read call. Zero (default) means wait until
	// the connection context ends — recommended for push-heavy servers; use
	// PingInterval for liveness. A positive value will disconnect peers that
	// send no application messages within the window even if pings succeed.
	ReadTimeout  time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`
	// PingInterval drives both WebSocket ping and Handlers.OnTick / ClientHandlers.OnTick.
	PingInterval    time.Duration `yaml:"ping_interval" json:"ping_interval"`
	MaxMessageBytes int64         `yaml:"max_message_bytes" json:"max_message_bytes"`
	SendQueueSize   int           `yaml:"send_queue_size" json:"send_queue_size"`

	// Backpressure selects Send behavior when the outbound queue is full.
	// Default: BackpressureBlock.
	Backpressure BackpressurePolicy `yaml:"backpressure" json:"backpressure"`

	// DropOnBackpressure is a deprecated alias for BackpressureError.
	// Prefer Backpressure. When true and Backpressure is the zero value (Block),
	// normalized() upgrades the policy to BackpressureError.
	DropOnBackpressure bool `yaml:"drop_on_backpressure" json:"drop_on_backpressure"`

	EnableCompression bool `yaml:"enable_compression" json:"enable_compression"`
}

func (c Config) normalized() Config {
	if c.HandshakeTimeout <= 0 {
		c.HandshakeTimeout = 5 * time.Second
	}
	// ReadTimeout: keep zero as "no per-read deadline". Do not invent a default
	// that kills idle push connections.
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10 * time.Second
	}
	if c.PingInterval <= 0 {
		c.PingInterval = 20 * time.Second
	}
	if c.MaxMessageBytes <= 0 {
		c.MaxMessageBytes = 4 << 20
	}
	if c.SendQueueSize <= 0 {
		c.SendQueueSize = 128
	}
	if c.DropOnBackpressure && c.Backpressure == BackpressureBlock {
		c.Backpressure = BackpressureError
	}
	return c
}

func (c Config) compressionMode() cws.CompressionMode {
	if c.EnableCompression {
		return cws.CompressionContextTakeover
	}
	return cws.CompressionDisabled
}

// ClientConfig extends Config with dial and reconnect options.
type ClientConfig struct {
	URL                  string        `yaml:"url" json:"url"`
	Headers              http.Header   `yaml:"-" json:"-"`
	ConnectTimeout       time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	Reconnect            bool          `yaml:"reconnect" json:"reconnect"`
	ReconnectBaseBackoff time.Duration `yaml:"reconnect_base_backoff" json:"reconnect_base_backoff"`
	ReconnectMaxBackoff  time.Duration `yaml:"reconnect_max_backoff" json:"reconnect_max_backoff"`
	BackoffJitterRatio   float64       `yaml:"backoff_jitter_ratio" json:"backoff_jitter_ratio"`
	Config
}

func (c ClientConfig) normalized() ClientConfig {
	c.Config = c.Config.normalized()
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	if c.ReconnectBaseBackoff <= 0 {
		c.ReconnectBaseBackoff = 200 * time.Millisecond
	}
	if c.ReconnectMaxBackoff <= 0 {
		c.ReconnectMaxBackoff = 5 * time.Second
	}
	if c.BackoffJitterRatio < 0 {
		c.BackoffJitterRatio = 0
	}
	if c.BackoffJitterRatio > 1 {
		c.BackoffJitterRatio = 1
	}
	return c
}

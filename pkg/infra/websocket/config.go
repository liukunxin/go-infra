package websocket

import (
	"net/http"
	"time"

	cws "github.com/coder/websocket"
)

type Config struct {
	HandshakeTimeout  time.Duration `yaml:"handshake_timeout" json:"handshake_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout" json:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout" json:"write_timeout"`
	PingInterval      time.Duration `yaml:"ping_interval" json:"ping_interval"`
	MaxMessageBytes   int64         `yaml:"max_message_bytes" json:"max_message_bytes"`
	SendQueueSize     int           `yaml:"send_queue_size" json:"send_queue_size"`
	DropOnBackpressure bool         `yaml:"drop_on_backpressure" json:"drop_on_backpressure"`
	EnableCompression bool          `yaml:"enable_compression" json:"enable_compression"`
}

func (c Config) normalized() Config {
	if c.HandshakeTimeout <= 0 {
		c.HandshakeTimeout = 5 * time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 60 * time.Second
	}
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
	return c
}

func (c Config) compressionMode() cws.CompressionMode {
	if c.EnableCompression {
		return cws.CompressionContextTakeover
	}
	return cws.CompressionDisabled
}

type ClientConfig struct {
	URL               string      `yaml:"url" json:"url"`
	Headers           http.Header `yaml:"-" json:"-"`
	ConnectTimeout      time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
	Reconnect           bool          `yaml:"reconnect" json:"reconnect"`
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

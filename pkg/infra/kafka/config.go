package kafka

import (
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Config holds cluster-level connection parameters shared by producers and consumers.
type Config struct {
	// Brokers is the list of Kafka broker addresses, e.g. ["kafka:9092"].
	Brokers []string `yaml:"brokers" json:"brokers"`

	// ClientID is an optional identifier included in Kafka requests for debugging.
	ClientID string `yaml:"client_id" json:"client_id"`

	// SASL configures authentication. Leave nil to skip authentication.
	SASL *SASLConfig `yaml:"sasl" json:"sasl"`

	// TLS enables TLS. Leave nil for plain-text connections.
	TLS *TLSConfig `yaml:"tls" json:"tls"`
}

// SASLConfig configures SASL authentication.
type SASLConfig struct {
	// Mechanism: "PLAIN", "SCRAM-SHA-256", or "SCRAM-SHA-512".
	Mechanism string `yaml:"mechanism" json:"mechanism"`
	Username  string `yaml:"username"  json:"username"`
	Password  string `yaml:"password"  json:"password"`
}

// TLSConfig enables TLS for Kafka connections.
type TLSConfig struct {
	// CACert is the PEM-encoded CA certificate used to verify the broker certificate.
	CACert []byte `yaml:"ca_cert" json:"ca_cert"`
	// InsecureSkipVerify disables server certificate verification. Never use in production.
	InsecureSkipVerify bool `yaml:"insecure_skip_verify" json:"insecure_skip_verify"`
}

// ProducerConfig controls producer behavior.
type ProducerConfig struct {
	// BatchSize is the maximum number of messages batched in a single Kafka request.
	// Default: 100
	BatchSize int `yaml:"batch_size" json:"batch_size"`

	// BatchTimeout is the maximum time to wait before flushing an incomplete batch.
	// Default: 10ms
	BatchTimeout time.Duration `yaml:"batch_timeout" json:"batch_timeout"`

	// WriteTimeout is the per-request write deadline. Default: 10s
	WriteTimeout time.Duration `yaml:"write_timeout" json:"write_timeout"`

	// MaxAttempts is the number of times to retry a failed write. Default: 3
	MaxAttempts int `yaml:"max_attempts" json:"max_attempts"`

	// RequiredAcks controls the durability guarantee.
	//   -1 (RequireAll)   — all in-sync replicas must ack (safest, default)
	//    0 (RequireNone)  — fire-and-forget (fastest, no guarantee)
	//    1 (RequireOne)   — leader ack only
	RequiredAcks kafkago.RequiredAcks `yaml:"required_acks" json:"required_acks"`

	// Compression algorithm. Default: no compression.
	// Options: kafka.Gzip, kafka.Snappy, kafka.Lz4, kafka.Zstd
	Compression kafkago.Compression `yaml:"compression" json:"compression"`

	// AllowAutoTopicCreation creates the topic on first write if it does not exist.
	// Requires broker auto.create.topics.enable=true. Default: false
	AllowAutoTopicCreation bool `yaml:"allow_auto_topic_creation" json:"allow_auto_topic_creation"`
}

func (c ProducerConfig) withDefaults() ProducerConfig {
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.BatchTimeout <= 0 {
		c.BatchTimeout = 10 * time.Millisecond
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 10 * time.Second
	}
	if c.MaxAttempts <= 0 {
		c.MaxAttempts = 3
	}
	if c.RequiredAcks == 0 {
		c.RequiredAcks = kafkago.RequireAll
	}
	return c
}

// ConsumerConfig controls consumer group behavior.
type ConsumerConfig struct {
	// GroupID is the Kafka consumer group name. Required.
	GroupID string `yaml:"group_id" json:"group_id"`

	// StartOffset controls where to start reading when no committed offset exists.
	//   kafka.FirstOffset (-2) — read from the beginning of the topic
	//   kafka.LastOffset  (-1) — read only new messages (default)
	StartOffset int64 `yaml:"start_offset" json:"start_offset"`

	// MinBytes / MaxBytes control fetch request sizes.
	// Default: MinBytes=1, MaxBytes=10MB
	MinBytes int `yaml:"min_bytes" json:"min_bytes"`
	MaxBytes int `yaml:"max_bytes" json:"max_bytes"`

	// MaxWait is the maximum time to block on a fetch request. Default: 1s
	MaxWait time.Duration `yaml:"max_wait" json:"max_wait"`

	// CommitInterval is how often offsets are auto-committed to Kafka.
	// 0 disables auto-commit (offsets are committed manually after each handler).
	// Default: 0 (manual commit for proper at-least-once delivery)
	CommitInterval time.Duration `yaml:"commit_interval" json:"commit_interval"`

	// MaxRetries is the number of times a failing handler is retried per message.
	// Default: 3
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// RetryBackoff is the initial backoff duration between handler retries.
	// Each retry doubles the backoff (exponential). Default: 200ms
	RetryBackoff time.Duration `yaml:"retry_backoff" json:"retry_backoff"`
}

func (c ConsumerConfig) withDefaults() ConsumerConfig {
	if c.StartOffset == 0 {
		c.StartOffset = kafkago.LastOffset
	}
	if c.MinBytes <= 0 {
		c.MinBytes = 1
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = 10 << 20 // 10 MB
	}
	if c.MaxWait <= 0 {
		c.MaxWait = time.Second
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.RetryBackoff <= 0 {
		c.RetryBackoff = 200 * time.Millisecond
	}
	return c
}

package kafka

import (
	"context"
	"errors"
	"fmt"
	"io"

	kafkago "github.com/segmentio/kafka-go"
)

// Producer sends messages to Kafka. It is safe for concurrent use.
// Topic can be set per-message, making one Producer usable across multiple topics.
type Producer struct {
	writer *kafkago.Writer
}

// NewProducer creates a Producer connected to the brokers in cfg.
// The producer does NOT fix a topic: callers set Message.Topic on each send,
// allowing one Producer instance to write to multiple topics.
func NewProducer(cfg Config, pcfg ProducerConfig) (*Producer, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka: at least one broker address is required")
	}
	pcfg = pcfg.withDefaults()

	transport, err := newTransport(cfg)
	if err != nil {
		return nil, err
	}

	w := &kafkago.Writer{
		Addr:                   kafkago.TCP(cfg.Brokers...),
		Balancer:               &kafkago.LeastBytes{},
		BatchSize:              pcfg.BatchSize,
		BatchTimeout:           pcfg.BatchTimeout,
		WriteTimeout:           pcfg.WriteTimeout,
		MaxAttempts:            pcfg.MaxAttempts,
		RequiredAcks:           pcfg.RequiredAcks,
		Compression:            pcfg.Compression,
		AllowAutoTopicCreation: pcfg.AllowAutoTopicCreation,
		ErrorLogger:            kafkago.LoggerFunc(func(s string, a ...interface{}) {}), // suppress internal noise; errors surface via WriteMessages
	}
	if transport != nil {
		w.Transport = transport
	}

	return &Producer{writer: w}, nil
}

// Send publishes one or more messages. All messages in a single call are sent
// in one batched request when possible.
//
// Message.Topic must be set (the producer is not bound to a fixed topic).
// Message.Time is set to the current time if zero.
func (p *Producer) Send(ctx context.Context, msgs ...Message) error {
	if len(msgs) == 0 {
		return nil
	}
	km := make([]kafkago.Message, 0, len(msgs))
	for _, m := range msgs {
		if m.Topic == "" {
			return errors.New("kafka: Message.Topic must not be empty")
		}
		km = append(km, toKafkaMessage(m))
	}
	if err := p.writer.WriteMessages(ctx, km...); err != nil {
		return fmt.Errorf("kafka: send %d message(s): %w", len(msgs), err)
	}
	return nil
}

// Stats returns writer statistics for monitoring (throughput, errors, latency).
func (p *Producer) Stats() kafkago.WriterStats {
	return p.writer.Stats()
}

// Close flushes all buffered messages and releases resources.
// Must be called when the producer is no longer needed.
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("kafka: close producer: %w", err)
	}
	return nil
}

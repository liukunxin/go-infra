package kafka

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// ── Global producer ───────────────────────────────────────────────────────────

var (
	globalProducer     atomic.Pointer[Producer]
	producerInitOnce   sync.Once
)

// InitProducer initializes the global Kafka producer. Only the first call takes effect.
// Returns an error so the caller can decide to fatal, panic, or fall back.
func InitProducer(cfg Config, pcfg ProducerConfig) error {
	var initErr error
	producerInitOnce.Do(func() {
		p, err := NewProducer(cfg, pcfg)
		if err != nil {
			initErr = fmt.Errorf("kafka: init producer: %w", err)
			return
		}
		globalProducer.Store(p)
	})
	return initErr
}

// GetProducer returns the global producer.
// Panics if InitProducer has not been called — programming error.
func GetProducer() *Producer {
	p := globalProducer.Load()
	if p == nil {
		panic("kafka: producer not initialized, call InitProducer first")
	}
	return p
}

// ── Global consumer group ─────────────────────────────────────────────────────

var (
	globalConsumer    atomic.Pointer[ConsumerGroup]
	consumerInitOnce  sync.Once
	consumerInitErrP  atomic.Pointer[error] // persists the first-call error for subsequent callers
)

// InitConsumer initializes the global Kafka consumer group. Only the first call takes effect.
// If the first call fails, every subsequent call returns the same error.
// Use Subscribe on the returned group to register handlers, then call Start.
func InitConsumer(cfg Config, ccfg ConsumerConfig) (*ConsumerGroup, error) {
	consumerInitOnce.Do(func() {
		g, err := NewConsumerGroup(cfg, ccfg)
		if err != nil {
			e := fmt.Errorf("kafka: init consumer: %w", err)
			consumerInitErrP.Store(&e)
			return
		}
		globalConsumer.Store(g)
	})
	if ep := consumerInitErrP.Load(); ep != nil {
		return nil, *ep
	}
	g := globalConsumer.Load()
	if g == nil {
		return nil, errors.New("kafka: consumer not initialized")
	}
	return g, nil
}

// GetConsumer returns the global consumer group.
// Panics if InitConsumer has not been called — programming error.
func GetConsumer() *ConsumerGroup {
	g := globalConsumer.Load()
	if g == nil {
		panic("kafka: consumer not initialized, call InitConsumer first")
	}
	return g
}

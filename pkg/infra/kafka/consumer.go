package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Handler is the callback invoked for every message received from a topic.
// Returning a non-nil error triggers the retry policy defined in ConsumerConfig.
// If all retries are exhausted, OnError (if set) is called and the message is skipped.
type Handler func(ctx context.Context, msg Message) error

// ConsumerGroup manages one kafka.Reader per subscribed topic, all sharing the same
// consumer group ID. Offset commits are manual (per-message) to guarantee at-least-once
// delivery: a message is committed only after the handler succeeds (or exhausts retries).
type ConsumerGroup struct {
	cfg  Config
	ccfg ConsumerConfig

	subsMu sync.Mutex
	subs   map[string]Handler // topic → handler, registered before Start

	startMu sync.Mutex        // protects cancel to avoid Start/Close data race
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// OnError is called when a message exceeds MaxRetries.
	// Use it to send to a dead-letter queue or emit an alert.
	// If nil, the error is logged and the message is skipped.
	OnError func(ctx context.Context, msg Message, err error)
}

// NewConsumerGroup creates a ConsumerGroup. Call Subscribe to register handlers,
// then Start to begin processing.
func NewConsumerGroup(cfg Config, ccfg ConsumerConfig) (*ConsumerGroup, error) {
	if len(cfg.Brokers) == 0 {
		return nil, errors.New("kafka: at least one broker address is required")
	}
	if ccfg.GroupID == "" {
		return nil, errors.New("kafka: ConsumerConfig.GroupID must not be empty")
	}
	return &ConsumerGroup{
		cfg:  cfg,
		ccfg: ccfg.withDefaults(),
		subs: make(map[string]Handler),
	}, nil
}

// Subscribe registers a handler for topic. Must be called before Start.
// Calling Subscribe after Start is a no-op.
func (g *ConsumerGroup) Subscribe(topic string, handler Handler) {
	g.subsMu.Lock()
	defer g.subsMu.Unlock()
	g.subs[topic] = handler
}

// Start launches one consumer goroutine per subscribed topic and blocks until
// ctx is cancelled (or Close is called). Returns an error if no topics have
// been subscribed or if the dialer cannot be built.
func (g *ConsumerGroup) Start(ctx context.Context) error {
	g.subsMu.Lock()
	if len(g.subs) == 0 {
		g.subsMu.Unlock()
		return errors.New("kafka: no topics subscribed; call Subscribe before Start")
	}
	// Snapshot subscriptions so Start is safe even if Subscribe is called concurrently.
	topics := make(map[string]Handler, len(g.subs))
	for t, h := range g.subs {
		topics[t] = h
	}
	g.subsMu.Unlock()

	dialer, err := newDialer(g.cfg)
	if err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(ctx)

	// Store cancel under startMu so Close() cannot race with this assignment.
	g.startMu.Lock()
	g.cancel = cancel
	g.startMu.Unlock()

	for topic, handler := range topics {
		tc := &topicConsumer{
			reader:  g.newReader(topic, dialer),
			handler: handler,
			cfg:     g.ccfg,
			onError: g.OnError,
		}
		g.wg.Add(1)
		go tc.run(runCtx, &g.wg)
	}

	// Block until the context is cancelled.
	<-runCtx.Done()
	return nil
}

// Close stops all consumer goroutines and waits for them to finish.
// It is safe to call Close concurrently with Start.
func (g *ConsumerGroup) Close() {
	g.startMu.Lock()
	cancel := g.cancel
	g.startMu.Unlock()
	if cancel != nil {
		cancel()
	}
	g.wg.Wait()
}

func (g *ConsumerGroup) newReader(topic string, dialer *kafkago.Dialer) *kafkago.Reader {
	rcfg := kafkago.ReaderConfig{
		Brokers:        g.cfg.Brokers,
		GroupID:        g.ccfg.GroupID,
		Topic:          topic,
		MinBytes:       g.ccfg.MinBytes,
		MaxBytes:       g.ccfg.MaxBytes,
		MaxWait:        g.ccfg.MaxWait,
		CommitInterval: g.ccfg.CommitInterval, // 0 = manual commit
		StartOffset:    g.ccfg.StartOffset,
	}
	if dialer != nil {
		rcfg.Dialer = dialer
	}
	return kafkago.NewReader(rcfg)
}

// ── per-topic consumer goroutine ──────────────────────────────────────────────

type topicConsumer struct {
	reader  *kafkago.Reader
	handler Handler
	cfg     ConsumerConfig
	onError func(context.Context, Message, error)
}

const (
	fetchErrInitBackoff = 100 * time.Millisecond
	fetchErrMaxBackoff  = 5 * time.Second
)

func (tc *topicConsumer) run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() {
		if err := tc.reader.Close(); err != nil {
			log.Printf("kafka: close reader %s: %v", tc.reader.Config().Topic, err)
		}
	}()

	fetchBackoff := fetchErrInitBackoff
	for {
		km, err := tc.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return // normal shutdown
			}
			log.Printf("kafka: fetch %s: %v", tc.reader.Config().Topic, err)
			// Exponential backoff to avoid tight-looping on persistent errors.
			select {
			case <-time.After(fetchBackoff):
			case <-ctx.Done():
				return
			}
			if fetchBackoff < fetchErrMaxBackoff {
				fetchBackoff *= 2
			}
			continue
		}
		fetchBackoff = fetchErrInitBackoff // reset on success

		msg := fromKafkaMessage(km)
		handlerErr := tc.invokeWithRetry(ctx, msg)

		if handlerErr != nil {
			if tc.onError != nil {
				tc.onError(ctx, msg, handlerErr)
			} else {
				log.Printf("kafka: handler %s offset=%d exhausted retries: %v",
					msg.Topic, msg.Offset, handlerErr)
			}
			// Skip the message — commit it so it is not replayed indefinitely.
		}

		// Commit regardless of handler outcome after retries.
		// With CommitInterval=0 (manual), this is the only commit path.
		if commitErr := tc.reader.CommitMessages(ctx, km); commitErr != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("kafka: commit %s offset=%d: %v", msg.Topic, msg.Offset, commitErr)
		}
	}
}

// invokeWithRetry calls the handler up to MaxRetries+1 times with exponential backoff.
// Returns the last error if all attempts fail, or nil on success.
func (tc *topicConsumer) invokeWithRetry(ctx context.Context, msg Message) error {
	var lastErr error
	for attempt := 0; attempt <= tc.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := tc.cfg.RetryBackoff * time.Duration(1<<uint(attempt-1))
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return fmt.Errorf("kafka: context cancelled during retry: %w", ctx.Err())
			}
		}
		if lastErr = tc.handler(ctx, msg); lastErr == nil {
			return nil
		}
	}
	return lastErr
}

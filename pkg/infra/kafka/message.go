package kafka

import (
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

// Header is a key-value pair attached to a Kafka message.
// Value is raw bytes; encode/decode is the caller's responsibility.
type Header struct {
	Key   string
	Value []byte
}

// Message is a unified type used for both producing and consuming.
//
// When producing: Topic, Key, Value, and Headers are used.
// When consuming: all fields are populated by the broker.
type Message struct {
	Topic     string
	Partition int
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   []Header
	Time      time.Time
}

// toKafkaMessage converts a Message to the kafka-go wire type.
func toKafkaMessage(m Message) kafkago.Message {
	km := kafkago.Message{
		Topic: m.Topic,
		Key:   m.Key,
		Value: m.Value,
		Time:  m.Time,
	}
	for _, h := range m.Headers {
		km.Headers = append(km.Headers, kafkago.Header{Key: h.Key, Value: h.Value})
	}
	return km
}

// fromKafkaMessage converts a kafka-go wire message to our Message type.
func fromKafkaMessage(km kafkago.Message) Message {
	m := Message{
		Topic:     km.Topic,
		Partition: km.Partition,
		Offset:    km.Offset,
		Key:       km.Key,
		Value:     km.Value,
		Time:      km.Time,
	}
	for _, h := range km.Headers {
		m.Headers = append(m.Headers, Header{Key: h.Key, Value: h.Value})
	}
	return m
}

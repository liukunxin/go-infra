package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// newTransport builds a *kafka.Transport for use by kafka.Writer (producer).
// Returns nil when no auth/TLS is configured (kafka-go uses a default transport).
func newTransport(cfg Config) (*kafkago.Transport, error) {
	if cfg.SASL == nil && cfg.TLS == nil && cfg.ClientID == "" {
		return nil, nil
	}

	t := &kafkago.Transport{}

	if cfg.ClientID != "" {
		t.ClientID = cfg.ClientID
	}
	if cfg.SASL != nil {
		m, err := buildSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		t.SASL = m
	}
	if cfg.TLS != nil {
		tlsCfg, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		t.TLS = tlsCfg
	}
	return t, nil
}

// newDialer builds a *kafka.Dialer for use by kafka.Reader (consumer).
// Returns nil (default dialer) when no auth/TLS is configured.
func newDialer(cfg Config) (*kafkago.Dialer, error) {
	if cfg.SASL == nil && cfg.TLS == nil {
		return nil, nil
	}

	d := *kafkago.DefaultDialer // copy defaults (timeout, keepalive, etc.)

	if cfg.SASL != nil {
		m, err := buildSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		d.SASLMechanism = m
	}
	if cfg.TLS != nil {
		tlsCfg, err := buildTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		d.TLS = tlsCfg
	}
	return &d, nil
}

func buildSASLMechanism(cfg *SASLConfig) (sasl.Mechanism, error) {
	switch cfg.Mechanism {
	case "PLAIN", "plain":
		return plain.Mechanism{Username: cfg.Username, Password: cfg.Password}, nil
	case "SCRAM-SHA-256":
		m, err := scram.Mechanism(scram.SHA256, cfg.Username, cfg.Password)
		if err != nil {
			return nil, fmt.Errorf("kafka: SCRAM-SHA-256 mechanism: %w", err)
		}
		return m, nil
	case "SCRAM-SHA-512":
		m, err := scram.Mechanism(scram.SHA512, cfg.Username, cfg.Password)
		if err != nil {
			return nil, fmt.Errorf("kafka: SCRAM-SHA-512 mechanism: %w", err)
		}
		return m, nil
	case "":
		return nil, errors.New("kafka: SASL.Mechanism must not be empty")
	default:
		return nil, fmt.Errorf("kafka: unsupported SASL mechanism %q (want PLAIN, SCRAM-SHA-256, SCRAM-SHA-512)", cfg.Mechanism)
	}
}

func buildTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsCfg := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec
	}
	if len(cfg.CACert) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(cfg.CACert) {
			return nil, errors.New("kafka: failed to parse CA certificate")
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

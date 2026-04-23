package es

import "time"

// Config holds Elasticsearch connection parameters.
// All Duration fields use standard Go notation (e.g. "30s", "1m").
type Config struct {
	// Addresses is a list of Elasticsearch node URLs, e.g. ["http://localhost:9200"].
	Addresses []string `yaml:"addresses" json:"addresses"`

	// Username / Password for HTTP Basic Auth (optional).
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"`

	// APIKey for API-key based authentication (optional; takes priority over Basic Auth).
	APIKey string `yaml:"api_key" json:"api_key"`

	// MaxRetries controls how many times a failed request is retried. Default 3.
	MaxRetries int `yaml:"max_retries" json:"max_retries"`

	// RetryBackoff is the initial backoff duration between retries. Default 100ms.
	RetryBackoff time.Duration `yaml:"retry_backoff" json:"retry_backoff"`

	// EnableTLS enables TLS for connections to Elasticsearch.
	EnableTLS bool `yaml:"enable_tls" json:"enable_tls"`

	// CACert is a PEM-encoded certificate authority cert used to verify the server cert.
	// Only relevant when EnableTLS is true.
	CACert []byte `yaml:"ca_cert" json:"ca_cert"`
}

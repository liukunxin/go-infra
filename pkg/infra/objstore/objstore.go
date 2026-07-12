package objstore

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Client wraps an S3-compatible client with presign capabilities.
type Client struct {
	s3           *s3.Client
	presign      *s3.PresignClient
	bucket       string
	endpoint     string
	usePathStyle bool
}

type clientHolder struct {
	c *Client
}

var (
	globalClient atomic.Pointer[clientHolder]
	initOnce     sync.Once
)

// Init initializes the global objstore client. Only the first call takes effect.
func Init(cfg *Config) error {
	if cfg == nil {
		return errors.New("objstore: config must not be nil")
	}

	var initErr error
	initOnce.Do(func() {
		c, err := NewClient(cfg)
		if err != nil {
			initErr = fmt.Errorf("objstore: init failed: %w", err)
			return
		}
		globalClient.Store(&clientHolder{c: c})
	})
	return initErr
}

// GetClient returns the global client.
// Panics if Init has not been called.
func GetClient() *Client {
	h := globalClient.Load()
	if h == nil {
		panic("objstore: not initialized, call Init first")
	}
	return h.c
}

// NewClient creates a new objstore Client without touching global state.
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("objstore: config must not be nil")
	}
	cfg.Normalize()
	if cfg.Endpoint == "" {
		return nil, errors.New("objstore: endpoint is required")
	}
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("objstore: access_key and secret_key are required")
	}

	region := cfg.Region

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	opts := func(o *s3.Options) {
		o.BaseEndpoint = aws.String(serviceBaseEndpoint(cfg.Endpoint))
		o.Region = region
		o.Credentials = credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")
		o.UsePathStyle = cfg.UsePathStyle
		o.HTTPClient = &http.Client{Transport: transport}
	}

	s3Client := s3.New(s3.Options{}, opts)
	presignClient := s3.NewPresignClient(s3Client)

	return &Client{
		s3:           s3Client,
		presign:      presignClient,
		bucket:       cfg.Bucket,
		endpoint:     cfg.Endpoint,
		usePathStyle: cfg.UsePathStyle,
	}, nil
}

// DefaultBucket returns the bucket configured at client creation time.
func (c *Client) DefaultBucket() string {
	return c.bucket
}

// resolveBucket returns the explicit bucket if non-empty, otherwise falls back to
// the client's default bucket configured at init time.
func (c *Client) resolveBucket(bucket string) string {
	if bucket != "" {
		return bucket
	}
	return c.bucket
}

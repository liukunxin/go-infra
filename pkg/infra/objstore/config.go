package objstore

import (
	"strings"
	"time"
)

const (
	// DefaultRegion is used when Config.Region is empty.
	DefaultRegion = "us-east-1"
	// DefaultPresignGetTTL is the fallback TTL for presigned GET URLs.
	DefaultPresignGetTTL = 5 * time.Minute
	// DefaultPresignPutTTL is the fallback TTL for presigned PUT URLs.
	DefaultPresignPutTTL = time.Hour
	// MaxPresignTTL is the maximum allowed presigned URL lifetime.
	MaxPresignTTL = 7 * 24 * time.Hour
)

// Config holds S3-compatible object storage connection parameters.
// Works with any S3-compatible provider (AWS S3, Aliyun OSS, Huawei OBS, KS3, MinIO, etc.)
// by setting the appropriate Endpoint.
type Config struct {
	Endpoint     string `yaml:"endpoint" json:"endpoint"`
	Region       string `yaml:"region" json:"region"`
	AccessKey    string `yaml:"access_key" json:"access_key"`
	SecretKey    string `yaml:"secret_key" json:"secret_key"`
	Bucket       string `yaml:"bucket" json:"bucket"`
	UsePathStyle bool   `yaml:"use_path_style" json:"use_path_style"`
}

// Normalize trims whitespace and removes URL schemes from Endpoint.
// Call before NewClient when loading config from files or env.
func (c *Config) Normalize() {
	if c == nil {
		return
	}
	c.Endpoint = normalizeEndpoint(c.Endpoint)
	c.Region = strings.TrimSpace(c.Region)
	c.AccessKey = strings.TrimSpace(c.AccessKey)
	c.SecretKey = strings.TrimSpace(c.SecretKey)
	c.Bucket = strings.TrimSpace(c.Bucket)
	if c.Region == "" {
		c.Region = DefaultRegion
	}
}

// PutOptions carries optional parameters for PutObject.
type PutOptions struct {
	ContentType   string
	ContentLength int64
	Expires       time.Time
	Metadata      map[string]string
	ACL           string
}

// PresignPutOptions carries optional parameters for presigned PUT URLs.
// When ACL or ContentType is set, the uploader must send the same headers on PUT.
type PresignPutOptions struct {
	ACL         string
	ContentType string
}

// ObjectMeta carries object metadata from HeadObject.
type ObjectMeta struct {
	Size        int64
	ContentType string
	ETag        string
}

// PresignedRequest describes a presigned HTTP request.
// RequiredHeaders lists headers the HTTP client must include exactly as signed.
type PresignedRequest struct {
	URL             string
	Method          string
	RequiredHeaders map[string]string
}

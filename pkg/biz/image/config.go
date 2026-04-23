package image

import "time"

// Config holds all configuration for the image service.
type Config struct {
	// Bucket is the object storage bucket name.
	Bucket string `yaml:"bucket" json:"bucket"`

	// KeyPrefix is the path prefix prepended to every stored object key.
	// Defaults to "images". Final key format: {KeyPrefix}/{imageID}
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`

	// SignSecret is the HMAC key used to sign and verify access tokens.
	// Must be non-empty; use a cryptographically random string of 32+ bytes.
	SignSecret string `yaml:"sign_secret" json:"sign_secret"`

	// DefaultTokenTTL is the default lifetime assigned to new access tokens.
	// Defaults to 30 minutes.
	DefaultTokenTTL time.Duration `yaml:"default_token_ttl" json:"default_token_ttl"`

	// PresignTTL is the lifetime of the storage-layer presigned URL returned
	// by ResolveAccessToken. Defaults to 5 minutes.
	// Keeping this shorter than DefaultTokenTTL limits the blast radius of a
	// leaked presigned URL while still allowing the business token to remain valid.
	PresignTTL time.Duration `yaml:"presign_ttl" json:"presign_ttl"`

	// AllowedMimeTypes whitelists accepted MIME types.
	// Defaults to all "image/*" types when empty.
	AllowedMimeTypes []string `yaml:"allowed_mime_types" json:"allowed_mime_types"`
}

func (c Config) keyPrefix() string {
	if c.KeyPrefix == "" {
		return "images"
	}
	return c.KeyPrefix
}

func (c Config) defaultTokenTTL() time.Duration {
	if c.DefaultTokenTTL <= 0 {
		return 30 * time.Minute
	}
	return c.DefaultTokenTTL
}

func (c Config) presignTTL() time.Duration {
	if c.PresignTTL <= 0 {
		return 5 * time.Minute
	}
	return c.PresignTTL
}

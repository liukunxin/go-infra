package objstore

import (
	"fmt"
	"net/url"
	"strings"
)

func normalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")
	return strings.TrimSuffix(endpoint, "/")
}

func serviceBaseEndpoint(endpoint string) string {
	endpoint = normalizeEndpoint(endpoint)
	if endpoint == "" {
		return ""
	}
	return "https://" + endpoint
}

func encodeObjectKey(key string) string {
	key = strings.TrimPrefix(strings.TrimSpace(key), "/")
	if key == "" {
		return ""
	}
	parts := strings.Split(key, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func buildObjectURL(endpoint, bucket, key string, usePathStyle bool) string {
	endpoint = normalizeEndpoint(endpoint)
	bucket = strings.TrimSpace(bucket)
	key = encodeObjectKey(key)
	if endpoint == "" || bucket == "" || key == "" {
		return ""
	}
	if usePathStyle {
		return fmt.Sprintf("https://%s/%s/%s", endpoint, bucket, key)
	}
	return fmt.Sprintf("https://%s.%s/%s", bucket, endpoint, key)
}

func buildPublicURL(publicBaseURL, endpoint, bucket, key string, usePathStyle bool) string {
	key = encodeObjectKey(key)
	if key == "" {
		return ""
	}
	if publicBaseURL != "" {
		return publicBaseURL + "/" + key
	}
	return buildObjectURL(endpoint, bucket, key, usePathStyle)
}

// JoinKey joins an optional prefix with path segments into a single object key.
func JoinKey(prefix string, parts ...string) string {
	segments := make([]string, 0, len(parts)+1)
	if p := strings.Trim(strings.TrimSpace(prefix), "/"); p != "" {
		segments = append(segments, p)
	}
	for _, part := range parts {
		if s := strings.Trim(strings.TrimSpace(part), "/"); s != "" {
			segments = append(segments, s)
		}
	}
	return strings.Join(segments, "/")
}

// ObjectURL builds a direct object URL without query-string auth.
// For private objects this URL is not anonymously accessible; use PresignGet instead.
func (c *Client) ObjectURL(bucket, key string) string {
	bucket = c.resolveBucket(bucket)
	return buildObjectURL(c.endpoint, bucket, key, c.usePathStyle)
}

// PublicURL builds the externally reachable object URL.
// When PublicBaseURL is configured it is used; otherwise falls back to ObjectURL.
func (c *Client) PublicURL(bucket, key string) string {
	bucket = c.resolveBucket(bucket)
	return buildPublicURL(c.publicBaseURL, c.endpoint, bucket, key, c.usePathStyle)
}

package objstore

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func clampTTL(ttl, fallback time.Duration) time.Duration {
	if ttl <= 0 || ttl > MaxPresignTTL {
		return fallback
	}
	return ttl
}

// PresignGet generates a presigned GET request for downloading an object.
func (c *Client) PresignGet(ctx context.Context, bucket, key string, ttl time.Duration) (*PresignedRequest, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return nil, err
	}

	ttl = clampTTL(ttl, DefaultPresignGetTTL)
	resp, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("objstore: presign get %q: %w", key, err)
	}
	return &PresignedRequest{
		URL:    resp.URL,
		Method: http.MethodGet,
	}, nil
}

// PresignPut generates a presigned PUT request for uploading an object.
// RequiredHeaders must be sent by the HTTP client when non-empty.
func (c *Client) PresignPut(ctx context.Context, bucket, key string, ttl time.Duration, opts PresignPutOptions) (*PresignedRequest, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return nil, err
	}

	ttl = clampTTL(ttl, DefaultPresignPutTTL)
	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}
	if acl, ok := resolvePresignPutACL(opts); ok {
		input.ACL = acl
	}
	if contentType := strings.TrimSpace(opts.ContentType); contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	resp, err := c.presign.PresignPutObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("objstore: presign put %q: %w", key, err)
	}
	return &PresignedRequest{
		URL:             resp.URL,
		Method:          http.MethodPut,
		RequiredHeaders: presignPutRequiredHeaders(opts),
	}, nil
}

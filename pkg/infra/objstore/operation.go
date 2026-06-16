package objstore

import (
	"context"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetObject downloads an object from the bucket.
// The caller must close the returned ReadCloser when finished.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.resolveBucket(bucket)),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

// PutObject uploads an object to the bucket.
// body is read until EOF; the caller is responsible for closing it.
func (c *Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, opts PutOptions) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(c.resolveBucket(bucket)),
		Key:    aws.String(key),
		Body:   body,
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if opts.ContentLength > 0 {
		input.ContentLength = aws.Int64(opts.ContentLength)
	}
	if !opts.Expires.IsZero() {
		input.Expires = aws.Time(opts.Expires)
	}
	if len(opts.Metadata) > 0 {
		input.Metadata = opts.Metadata
	}
	_, err := c.s3.PutObject(ctx, input)
	return err
}

// DeleteObject removes an object from the bucket.
// No error is returned when the object does not exist (S3 standard behavior).
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.resolveBucket(bucket)),
		Key:    aws.String(key),
	})
	return err
}

// PresignGetURL generates a presigned URL for downloading an object.
// ttl controls the URL validity duration; clamped to [1s, 7d].
func (c *Client) PresignGetURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	ttl = clampTTL(ttl, 300*time.Second)
	resp, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.resolveBucket(bucket)),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return resp.URL, nil
}

// PresignPutURL generates a presigned URL for uploading an object.
// ttl controls the URL validity duration; clamped to [1s, 7d].
func (c *Client) PresignPutURL(ctx context.Context, bucket, key string, ttl time.Duration) (string, error) {
	ttl = clampTTL(ttl, 3600*time.Second)
	resp, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.resolveBucket(bucket)),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", err
	}
	return resp.URL, nil
}

const maxTTL = 7 * 24 * time.Hour // S3 presign max: 7 days

func clampTTL(ttl, fallback time.Duration) time.Duration {
	if ttl <= 0 || ttl > maxTTL {
		return fallback
	}
	return ttl
}

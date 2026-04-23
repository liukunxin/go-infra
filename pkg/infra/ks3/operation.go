package ks3

import (
	"context"
	"io"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// PutOptions carries optional parameters for PutObject.
type PutOptions struct {
	ContentType string
	Size        int64
}

// PutObject uploads an object to the specified bucket.
// body is read until EOF; the caller is responsible for closing it.
func PutObject(ctx context.Context, bucket, key string, body io.Reader, opts PutOptions) error {
	input := &s3.PutReaderInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if opts.ContentType != "" {
		input.ContentType = aws.String(opts.ContentType)
	}
	if opts.Size > 0 {
		input.ContentLength = aws.Long(opts.Size)
	}
	_, err := client.PutReaderWithContext(ctx, input)
	return err
}

// DeleteObject removes an object from the specified bucket.
// No error is returned when the object does not exist.
func DeleteObject(ctx context.Context, bucket, key string) error {
	_, err := client.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	return err
}

// PresignGetURL generates a read-only presigned URL for the specified object.
// ttl must be between 1 s and 7 d (604800 s); out-of-range values are clamped to 300 s.
func PresignGetURL(_ context.Context, bucket, key string, ttl time.Duration) (string, error) {
	seconds := int64(ttl.Seconds())
	if seconds <= 0 || seconds > 604800 {
		seconds = 300
	}
	return client.GeneratePresignedUrl(&s3.GeneratePresignedUrlInput{
		Bucket:     aws.String(bucket),
		Key:        aws.String(key),
		HTTPMethod: s3.HTTPMethod("GET"),
		Expires:    seconds,
	})
}

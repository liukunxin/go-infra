package objstore

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetObject downloads an object from the bucket.
// The caller must close the returned ReadCloser when finished.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return nil, err
	}

	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("objstore: get object %q: %w", key, err)
	}
	return resp.Body, nil
}

// PutObject uploads an object to the bucket.
// body is read until EOF; the caller is responsible for closing it when applicable.
func (c *Client) PutObject(ctx context.Context, bucket, key string, body io.Reader, opts PutOptions) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return err
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(bucket),
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
	if acl, ok := resolvePresignPutACL(PresignPutOptions{ACL: opts.ACL}); ok {
		input.ACL = acl
	}

	if _, err := c.s3.PutObject(ctx, input); err != nil {
		return fmt.Errorf("objstore: put object %q: %w", key, err)
	}
	return nil
}

// DeleteObject removes an object from the bucket.
func (c *Client) DeleteObject(ctx context.Context, bucket, key string) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return err
	}

	if _, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		return fmt.Errorf("objstore: delete object %q: %w", key, err)
	}
	return nil
}

// HeadObject returns object metadata without downloading the body.
func (c *Client) HeadObject(ctx context.Context, bucket, key string) (*ObjectMeta, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	bucket, err := c.resolveBucketOrError(bucket)
	if err != nil {
		return nil, err
	}

	resp, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("objstore: head object %q: %w", key, err)
	}

	meta := &ObjectMeta{}
	if resp.ContentLength != nil {
		meta.Size = *resp.ContentLength
	}
	if resp.ContentType != nil {
		meta.ContentType = *resp.ContentType
	}
	if resp.ETag != nil {
		meta.ETag = strings.Trim(*resp.ETag, `"`)
	}
	return meta, nil
}

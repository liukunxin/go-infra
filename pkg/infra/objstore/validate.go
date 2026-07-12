package objstore

import (
	"fmt"
	"strings"
)

func validateObjectKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("%w: object key is required", ErrInvalidArgument)
	}
	return nil
}

func (c *Client) resolveBucketOrError(bucket string) (string, error) {
	bucket = c.resolveBucket(bucket)
	if strings.TrimSpace(bucket) == "" {
		return "", fmt.Errorf("%w: bucket is required", ErrInvalidArgument)
	}
	return bucket, nil
}

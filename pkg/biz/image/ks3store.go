package image

import (
	"context"
	"io"
	"time"

	infraks3 "github.com/liukunxin/go-infra/pkg/infra/ks3"
)

// KS3Store is an ObjectStore backed by KS3.
// The global KS3 client must be initialized via ks3.Init before any method is called.
type KS3Store struct {
	bucket string
}

// NewKS3Store creates a KS3-backed ObjectStore for the given bucket.
// Pass the result to NewService as the ObjectStore parameter.
//
//	ks3.Init(&ks3.Config{...})
//	store := image.NewKS3Store("my-image-bucket")
//	svc, _ := image.NewService(cfg, store, myMetaStore)
func NewKS3Store(bucket string) *KS3Store {
	return &KS3Store{bucket: bucket}
}

func (k *KS3Store) Put(ctx context.Context, key string, body io.Reader, opts PutOptions) error {
	return infraks3.PutObject(ctx, k.bucket, key, body, infraks3.PutOptions{
		ContentType: opts.ContentType,
		Size:        opts.Size,
	})
}

func (k *KS3Store) Delete(ctx context.Context, key string) error {
	return infraks3.DeleteObject(ctx, k.bucket, key)
}

func (k *KS3Store) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	return infraks3.PresignGetURL(ctx, k.bucket, key, ttl)
}

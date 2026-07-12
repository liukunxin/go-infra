package image

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofrs/uuid"
	"github.com/liukunxin/go-infra/pkg/infra/objstore"
)

// Service is the image business service.
//
// Construct with NewService. The objstore client must be initialized via objstore.Init
// before calling any method, or pass a custom *objstore.Client via WithClient option.
type Service struct {
	cfg    Config
	store  *objstore.Client
	meta   MetaStorage // nil disables metadata persistence
	signer *signer
}

// NewService creates an image service.
// cfg.SignSecret and cfg.Bucket must not be empty.
// If no WithClient option is provided, the global objstore.GetClient() is used.
func NewService(cfg Config, meta MetaStorage, opts ...ServiceOption) (*Service, error) {
	sg, err := newSigner(cfg.SignSecret)
	if err != nil {
		return nil, err
	}
	if cfg.Bucket == "" {
		return nil, errors.New("image: bucket must not be empty")
	}

	s := &Service{cfg: cfg, meta: meta, signer: sg}
	for _, opt := range opts {
		opt(s)
	}
	if s.store == nil {
		s.store = objstore.GetClient()
	}
	return s, nil
}

// ServiceOption configures the image Service.
type ServiceOption func(*Service)

// WithClient injects a custom objstore.Client instead of the global one.
func WithClient(c *objstore.Client) ServiceOption {
	return func(s *Service) { s.store = c }
}

// Upload stores an image and returns its metadata.
//
// MIME handling:
//   - If MimeType is provided it is validated against Config.AllowedMimeTypes.
//   - If MimeType is empty up to 3072 bytes of Body are peeked to auto-detect the type;
//     the full body (including the peeked bytes) is still forwarded to storage.
//
// When UploadRequest.MaxBytes > 0, reads exceeding that limit abort with ErrFileTooLarge.
// On metadata-save failure the object is deleted to avoid orphaned storage.
func (s *Service) Upload(ctx context.Context, req *UploadRequest) (*Image, error) {
	body := io.Reader(req.Body)
	if req.MaxBytes > 0 {
		body = &limitedReader{r: body, n: req.MaxBytes}
	}

	mime := req.MimeType
	if mime == "" {
		// TeeReader copies the peeked bytes into buf so they can be prepended back.
		var buf bytes.Buffer
		tr := io.TeeReader(body, &buf)
		detected, peekErr := mimetype.DetectReader(tr)
		if peekErr != nil {
			return nil, fmt.Errorf("image: detect mime type: %w", peekErr)
		}
		body = io.MultiReader(&buf, body)
		mime = detected.String()
	}
	if err := s.validateMime(mime); err != nil {
		return nil, err
	}

	id := newImageID()
	key := s.objectKey(id)

	if err := s.store.PutObject(ctx, s.cfg.Bucket, key, body, objstore.PutOptions{
		ContentType:   mime,
		ContentLength: req.Size,
	}); err != nil {
		return nil, fmt.Errorf("image: put object: %w", err)
	}

	img := &Image{
		ID:        id,
		MimeType:  mime,
		Size:      req.Size,
		OwnerID:   req.OwnerID,
		CreatedAt: time.Now(),
	}
	if s.meta != nil {
		if err := s.meta.Save(ctx, img); err != nil {
			_ = s.store.DeleteObject(context.Background(), s.cfg.Bucket, key)
			return nil, fmt.Errorf("image: save metadata: %w", err)
		}
	}
	return img, nil
}

// TokenOption is a functional option for BuildAccessToken.
type TokenOption func(*tokenBuildOpts)

type tokenBuildOpts struct{ ttl time.Duration }

// WithTokenTTL overrides the default token lifetime for a single call.
func WithTokenTTL(d time.Duration) TokenOption {
	return func(o *tokenBuildOpts) { o.ttl = d }
}

// BuildAccessToken generates a signed, expiring access token for imageID.
//
// viewerID binds the token to a specific user (private mode): only that user
// can later call ResolveAccessToken successfully.
// Pass an empty viewerID to create a public token usable by anyone.
func (s *Service) BuildAccessToken(imageID, viewerID string, opts ...TokenOption) (string, error) {
	o := &tokenBuildOpts{ttl: s.cfg.defaultTokenTTL()}
	for _, opt := range opts {
		opt(o)
	}
	return s.signer.sign(imageID, viewerID, o.ttl)
}

// ResolveAccessToken verifies a token and returns a short-lived storage presigned URL
// intended for HTTP 302 redirection. The presigned URL lifetime is Config.PresignTTL
// (default 5 min), deliberately shorter than the business token lifetime to limit
// the impact of a leaked storage URL.
//
// viewerID is checked against the token's bound user when the token is in private mode.
// Pass an empty viewerID for public tokens.
func (s *Service) ResolveAccessToken(ctx context.Context, token, viewerID string) (string, error) {
	claims, err := s.signer.verify(token)
	if err != nil {
		return "", err
	}
	if claims.OwnerID != "" && claims.OwnerID != viewerID {
		return "", ErrTokenUnauthorized
	}
	presigned, err := s.store.PresignGet(ctx, s.cfg.Bucket, s.objectKey(claims.ImageID), s.cfg.presignTTL())
	if err != nil {
		return "", fmt.Errorf("image: presign url: %w", err)
	}
	return presigned.URL, nil
}

// Delete removes the image object from storage and its metadata record.
// Both operations are always attempted; all errors are joined and returned together
// so that a storage failure does not silently leave an orphaned metadata record.
func (s *Service) Delete(ctx context.Context, imageID string) error {
	var errs []error
	if err := s.store.DeleteObject(ctx, s.cfg.Bucket, s.objectKey(imageID)); err != nil {
		errs = append(errs, fmt.Errorf("image: delete object: %w", err))
	}
	if s.meta != nil {
		if err := s.meta.Delete(ctx, imageID); err != nil {
			errs = append(errs, fmt.Errorf("image: delete metadata: %w", err))
		}
	}
	return errors.Join(errs...)
}

// objectKey maps an imageID to a deterministic object storage key.
// Using a flat prefix/id layout keeps the key predictable without DB lookups.
func (s *Service) objectKey(imageID string) string {
	return s.cfg.keyPrefix() + "/" + imageID
}

func (s *Service) validateMime(mime string) error {
	if !strings.HasPrefix(mime, "image/") {
		return ErrInvalidMimeType
	}
	if len(s.cfg.AllowedMimeTypes) == 0 {
		return nil
	}
	for _, allowed := range s.cfg.AllowedMimeTypes {
		if mime == allowed {
			return nil
		}
	}
	return ErrInvalidMimeType
}

func newImageID() string {
	// UUID v7 is time-ordered which improves B-tree DB index performance.
	if id, err := uuid.NewV7(); err == nil {
		return id.String()
	}
	return uuid.Must(uuid.NewV4()).String()
}

// limitedReader wraps an io.Reader and returns ErrFileTooLarge when the underlying
// stream exceeds n bytes. It reads up to n+1 bytes at a time so it can distinguish
// "file is exactly n bytes" (returns io.EOF normally) from "file exceeds n bytes".
type limitedReader struct {
	r io.Reader
	n int64 // remaining bytes allowed
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	// Cap to n+1 so we can detect data beyond the limit in a single Read call.
	cap := l.n + 1
	if int64(len(p)) > cap {
		p = p[:cap]
	}
	n, err := l.r.Read(p)
	if int64(n) > l.n {
		// Successfully read past the limit → file is too large.
		return 0, ErrFileTooLarge
	}
	l.n -= int64(n)
	return n, err
}

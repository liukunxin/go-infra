package objstore

import "errors"

var (
	// ErrInvalidArgument indicates caller supplied invalid bucket/key/options.
	ErrInvalidArgument = errors.New("objstore: invalid argument")
	// ErrNotConfigured indicates the package global client was not initialized.
	ErrNotConfigured = errors.New("objstore: not configured")
)

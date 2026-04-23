package image

import "errors"

var (
	// ErrInvalidMimeType is returned when the MIME type is not in the allowed list.
	ErrInvalidMimeType = errors.New("image: unsupported MIME type")
	// ErrFileTooLarge is returned when the upload exceeds MaxBytes.
	ErrFileTooLarge = errors.New("image: file exceeds maximum size limit")
	// ErrNotFound is returned when the image ID does not exist in MetaStorage.
	ErrNotFound = errors.New("image: not found")
	// ErrTokenExpired is returned when the access token has passed its expiry time.
	ErrTokenExpired = errors.New("image: access token expired")
	// ErrTokenInvalid is returned when the token is malformed or the signature does not match.
	ErrTokenInvalid = errors.New("image: access token invalid")
	// ErrTokenUnauthorized is returned when the caller's viewer ID does not match the token's bound user.
	ErrTokenUnauthorized = errors.New("image: viewer not authorized for this token")
	// ErrMissingSecret is returned when Config.SignSecret is empty.
	ErrMissingSecret = errors.New("image: sign secret must not be empty")
)

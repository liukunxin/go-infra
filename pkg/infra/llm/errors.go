package llm

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	// ErrNoProviders indicates no provider was registered during client creation.
	ErrNoProviders = errors.New("llm: no providers configured")
	// ErrProviderNotFound indicates the selected provider is not registered.
	ErrProviderNotFound = errors.New("llm: provider not found")
	// ErrProviderRequired indicates provider cannot be resolved from request/options/default.
	ErrProviderRequired = errors.New("llm: provider is required")
	// ErrModelRequired indicates model cannot be resolved from request/options/default.
	ErrModelRequired = errors.New("llm: model is required")
	// ErrInvalidConfig indicates required provider/client config is missing.
	ErrInvalidConfig = errors.New("llm: invalid config")
)

// ProviderError describes upstream provider failures with useful debug metadata.
type ProviderError struct {
	Provider   string
	HTTPStatus int
	RequestID  string
	Code       string
	Type       string
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return ""
	}
	status := e.HTTPStatus
	if status == 0 {
		status = http.StatusInternalServerError
	}
	if e.Code != "" {
		return fmt.Sprintf("llm: provider=%s status=%d code=%s msg=%s", e.Provider, status, e.Code, e.Message)
	}
	return fmt.Sprintf("llm: provider=%s status=%d msg=%s", e.Provider, status, e.Message)
}

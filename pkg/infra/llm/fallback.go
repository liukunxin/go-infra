package llm

import (
	"context"
	"errors"
)

// FallbackTarget defines one fallback provider/model candidate.
type FallbackTarget struct {
	Provider string `yaml:"provider" json:"provider"`
	Model    string `yaml:"model" json:"model"`
}

// key format:
// - provider + "::" + model   => specific route
// - provider + "::*"          => provider-level route for any model
func fallbackKey(provider, model string) string {
	if model == "" {
		return provider + "::*"
	}
	return provider + "::" + model
}

func shouldFallback(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, ErrInvalidConfig) ||
		errors.Is(err, ErrProviderRequired) ||
		errors.Is(err, ErrProviderNotFound) ||
		errors.Is(err, ErrModelRequired) ||
		errors.Is(err, ErrNoProviders) {
		return false
	}
	return true
}

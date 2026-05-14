package llm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/codes"
)

// Client is the unified entrypoint for LLM calls.
type Client struct {
	defaultProvider string
	defaultModel    string
	providers       map[string]Provider
	fallbacks       map[string][]FallbackTarget
}

// New creates an LLM client.
func New(opts ...Option) (*Client, error) {
	c := defaultOptionConfig()
	for _, opt := range opts {
		if err := opt.Apply(c); err != nil {
			return nil, err
		}
	}
	if len(c.providers) == 0 {
		return nil, ErrNoProviders
	}
	if c.defaultProvider == "" && len(c.providers) == 1 {
		for name := range c.providers {
			c.defaultProvider = name
		}
	}
	if c.defaultProvider != "" {
		if _, ok := c.providers[c.defaultProvider]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, c.defaultProvider)
		}
	}
	return &Client{
		defaultProvider: c.defaultProvider,
		defaultModel:    c.defaultModel,
		providers:       c.providers,
		fallbacks:       c.fallbacks,
	}, nil
}

// Generate performs one non-streaming chat completion request.
func (c *Client) Generate(ctx context.Context, req GenerateRequest, opts ...CallOption) (*GenerateResponse, error) {
	attempts, err := c.resolveAttempts(req, opts...)
	if err != nil {
		return nil, err
	}
	var lastErr error
	for idx, attempt := range attempts {
		opCtx, span := startSpan(ctx, "generate", attempt.ProviderName, attempt.Request.Model)
		start := time.Now()
		logAttempt(opCtx, "llm generate attempt", map[string]interface{}{
			"provider": attempt.ProviderName,
			"model":    attempt.Request.Model,
			"attempt":  idx + 1,
			"total":    len(attempts),
		})
		resp, callErr := attempt.Provider.Generate(opCtx, attempt.Request)
		if callErr != nil {
			lastErr = callErr
			recordMetrics(opCtx, attempt.ProviderName, attempt.Request.Model, "generate", "error", time.Since(start))
			span.RecordError(callErr)
			span.SetStatus(codes.Error, callErr.Error())
			span.End()
			if idx < len(attempts)-1 && shouldFallback(callErr) {
				continue
			}
			return nil, callErr
		}
		recordMetrics(opCtx, attempt.ProviderName, attempt.Request.Model, "generate", "ok", time.Since(start))
		span.SetStatus(codes.Ok, "ok")
		span.End()
		if resp != nil {
			resp.Provider = attempt.ProviderName
			if resp.Model == "" {
				resp.Model = attempt.Request.Model
			}
		}
		return resp, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("llm: request failed without error")
}

// GenerateStream performs one streaming chat completion request.
func (c *Client) GenerateStream(ctx context.Context, req GenerateRequest, opts ...CallOption) (<-chan StreamEvent, error) {
	attempts, err := c.resolveAttempts(req, opts...)
	if err != nil {
		return nil, err
	}
	var (
		sourceCh <-chan StreamEvent
		route    resolvedRoute
	)
	lastErr := error(nil)
	for idx, attempt := range attempts {
		opCtx, span := startSpan(ctx, "stream", attempt.ProviderName, attempt.Request.Model)
		start := time.Now()
		logAttempt(opCtx, "llm stream attempt", map[string]interface{}{
			"provider": attempt.ProviderName,
			"model":    attempt.Request.Model,
			"attempt":  idx + 1,
			"total":    len(attempts),
		})
		ch, callErr := attempt.Provider.GenerateStream(opCtx, attempt.Request)
		if callErr != nil {
			lastErr = callErr
			recordMetrics(opCtx, attempt.ProviderName, attempt.Request.Model, "stream", "error", time.Since(start))
			span.RecordError(callErr)
			span.SetStatus(codes.Error, callErr.Error())
			span.End()
			if idx < len(attempts)-1 && shouldFallback(callErr) {
				continue
			}
			return nil, callErr
		}
		recordMetrics(opCtx, attempt.ProviderName, attempt.Request.Model, "stream", "ok", time.Since(start))
		span.SetStatus(codes.Ok, "ok")
		span.End()
		sourceCh = ch
		route = attempt
		break
	}
	if sourceCh == nil {
		if lastErr != nil {
			return nil, lastErr
		}
		return nil, errors.New("llm: stream failed without error")
	}
	out := make(chan StreamEvent)
	go func() {
		defer close(out)
		for ev := range sourceCh {
			ev.Provider = route.ProviderName
			if ev.Model == "" {
				ev.Model = route.Request.Model
			}
			out <- ev
		}
	}()
	return out, nil
}

type resolvedRoute struct {
	ProviderName string
	Provider     Provider
	Request      GenerateRequest
}

type providerDefaultModel interface {
	DefaultModel() string
}

func (c *Client) resolveAttempts(req GenerateRequest, opts ...CallOption) ([]resolvedRoute, error) {
	if c == nil {
		return nil, errors.New("llm: client is nil")
	}
	callCfg := &callOptionConfig{}
	for _, opt := range opts {
		if err := opt.Apply(callCfg); err != nil {
			return nil, err
		}
	}
	if req.Provider == "" {
		req.Provider = callCfg.provider
	}
	if req.Provider == "" {
		req.Provider = c.defaultProvider
	}
	if req.Provider == "" {
		return nil, ErrProviderRequired
	}
	primaryProvider, ok := c.providers[req.Provider]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, req.Provider)
	}
	if req.Model == "" {
		req.Model = callCfg.model
	}
	if req.Model == "" {
		req.Model = c.defaultModel
	}
	if req.Model == "" {
		if dm, ok := primaryProvider.(providerDefaultModel); ok {
			req.Model = dm.DefaultModel()
		}
	}
	if req.Model == "" {
		return nil, ErrModelRequired
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("%w: messages is empty", ErrInvalidConfig)
	}
	attempts := []resolvedRoute{{
		ProviderName: req.Provider,
		Provider:     primaryProvider,
		Request:      req,
	}}

	targets := c.fallbackCandidates(req.Provider, req.Model)
	for _, t := range targets {
		provider, exists := c.providers[t.Provider]
		if !exists {
			continue
		}
		if t.Provider == req.Provider && t.Model == req.Model {
			continue
		}
		attemptReq := req
		attemptReq.Provider = t.Provider
		attemptReq.Model = t.Model
		attempts = append(attempts, resolvedRoute{
			ProviderName: t.Provider,
			Provider:     provider,
			Request:      attemptReq,
		})
	}
	return attempts, nil
}

func (c *Client) fallbackCandidates(provider, model string) []FallbackTarget {
	out := make([]FallbackTarget, 0, 4)
	if c == nil || len(c.fallbacks) == 0 {
		return out
	}
	if targets, ok := c.fallbacks[fallbackKey(provider, model)]; ok {
		out = append(out, targets...)
	}
	if targets, ok := c.fallbacks[fallbackKey(provider, "")]; ok {
		out = append(out, targets...)
	}
	return out
}

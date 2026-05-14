package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type openAICompatibleProvider struct {
	name       string
	baseURL    string
	apiKey     string
	httpClient *http.Client
	headers    map[string]string
	model      string
	retry      RetryConfig
}

// OpenAICompatibleConfig configures an OpenAI-compatible provider.
// Many vendors (OpenAI/DeepSeek/Qwen/Kimi/Volcengine etc.) support this protocol.
type OpenAICompatibleConfig struct {
	BaseURL      string            `yaml:"base_url" json:"base_url"`
	APIKey       string            `yaml:"api_key" json:"api_key"`
	HTTPClient   *http.Client      `yaml:"-" json:"-"`
	HTTPTimeout  time.Duration     `yaml:"http_timeout" json:"http_timeout"`
	Headers      map[string]string `yaml:"headers" json:"headers"`
	DefaultModel string            `yaml:"default_model" json:"default_model"`
	Retry        RetryConfig       `yaml:"retry" json:"retry"`
}

// NewOpenAICompatibleProvider builds a provider from OpenAI-compatible HTTP APIs.
func NewOpenAICompatibleProvider(name string, cfg OpenAICompatibleConfig) (Provider, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: provider name is empty", ErrInvalidConfig)
	}
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.HTTPTimeout
		if timeout <= 0 {
			timeout = 30 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	return &openAICompatibleProvider{
		name:       name,
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		httpClient: httpClient,
		headers:    headers,
		model:      cfg.DefaultModel,
		retry:      cfg.Retry.normalized(),
	}, nil
}

func (p *openAICompatibleProvider) Name() string { return p.name }

func (p *openAICompatibleProvider) DefaultModel() string { return strings.TrimSpace(p.model) }

func (p *openAICompatibleProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(p.model)
	}
	if model == "" {
		return nil, ErrModelRequired
	}
	payload := mapGenerateRequest(req, model, false)
	respBody, status, headers, err := p.doJSON(ctx, http.MethodPost, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}
	if status >= http.StatusBadRequest {
		return nil, p.buildProviderError(status, headers, respBody)
	}
	var resp openAIChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("llm: decode response failed: %w", err)
	}
	out := &GenerateResponse{
		Provider: p.name,
		Model:    resp.Model,
		Usage:    resp.Usage,
	}
	if len(resp.Choices) > 0 {
		out.Content = resp.Choices[0].Message.Content
		out.FinishReason = resp.Choices[0].FinishReason
	}
	if out.Model == "" {
		out.Model = model
	}
	return out, nil
}

func (p *openAICompatibleProvider) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamEvent, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(p.model)
	}
	if model == "" {
		return nil, ErrModelRequired
	}
	payload := mapGenerateRequest(req, model, true)
	resp, err := p.doStreamRequest(ctx, "/chat/completions", payload)
	if err != nil {
		return nil, err
	}

	ch := make(chan StreamEvent)
	go func() {
		defer close(ch)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || !strings.HasPrefix(line, "data:") {
				continue
			}
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "" {
				continue
			}
			if data == "[DONE]" {
				ch <- StreamEvent{
					Provider: p.name,
					Model:    model,
					Done:     true,
				}
				return
			}
			var chunk openAIChatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- StreamEvent{
					Provider: p.name,
					Model:    model,
					Err:      fmt.Errorf("llm: decode stream chunk failed: %w", err),
				}
				return
			}
			eventModel := chunk.Model
			if eventModel == "" {
				eventModel = model
			}
			if len(chunk.Choices) == 0 && chunk.Usage != nil {
				ch <- StreamEvent{
					Provider: p.name,
					Model:    eventModel,
					Usage:    chunk.Usage,
				}
				continue
			}
			for _, choice := range chunk.Choices {
				ch <- StreamEvent{
					Provider:     p.name,
					Model:        eventModel,
					Delta:        choice.Delta.Content,
					FinishReason: choice.FinishReason,
				}
			}
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
			ch <- StreamEvent{
				Provider: p.name,
				Model:    model,
				Err:      err,
			}
		}
	}()

	return ch, nil
}

func (p *openAICompatibleProvider) doJSON(ctx context.Context, method, path string, payload any) ([]byte, int, http.Header, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("llm: encode request failed: %w", err)
	}
	maxAttempts := p.retry.maxAttempts()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		httpReq, reqErr := http.NewRequestWithContext(ctx, method, p.baseURL+path, bytes.NewReader(body))
		if reqErr != nil {
			return nil, 0, nil, reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		p.applyAuthHeaders(httpReq)

		resp, doErr := p.httpClient.Do(httpReq)
		if doErr != nil {
			if attempt < maxAttempts && shouldRetryTransportError(doErr) {
				if !sleepWithContext(ctx, p.retry.backoff(attempt)) {
					return nil, 0, nil, ctx.Err()
				}
				continue
			}
			return nil, 0, nil, doErr
		}
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if readErr != nil {
			if attempt < maxAttempts && shouldRetryTransportError(readErr) {
				if !sleepWithContext(ctx, p.retry.backoff(attempt)) {
					return nil, 0, nil, ctx.Err()
				}
				continue
			}
			return nil, resp.StatusCode, resp.Header, readErr
		}
		if attempt < maxAttempts && p.retry.shouldRetryHTTPStatus(resp.StatusCode) {
			if !sleepWithContext(ctx, p.retry.backoff(attempt)) {
				return nil, 0, nil, ctx.Err()
			}
			continue
		}
		return respBody, resp.StatusCode, resp.Header, nil
	}
	return nil, 0, nil, errors.New("llm: request failed after retries")
}

func (p *openAICompatibleProvider) applyAuthHeaders(req *http.Request) {
	if strings.TrimSpace(p.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
}

func (p *openAICompatibleProvider) doStreamRequest(ctx context.Context, path string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("llm: encode request failed: %w", err)
	}
	maxAttempts := p.retry.maxAttempts()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+path, bytes.NewReader(body))
		if reqErr != nil {
			return nil, reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		p.applyAuthHeaders(httpReq)

		resp, doErr := p.httpClient.Do(httpReq)
		if doErr != nil {
			if attempt < maxAttempts && shouldRetryTransportError(doErr) {
				if !sleepWithContext(ctx, p.retry.backoff(attempt)) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, doErr
		}
		if resp.StatusCode >= http.StatusBadRequest {
			respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
			resp.Body.Close()
			err = p.buildProviderError(resp.StatusCode, resp.Header, respBody)
			if attempt < maxAttempts && p.retry.shouldRetryHTTPStatus(resp.StatusCode) {
				if !sleepWithContext(ctx, p.retry.backoff(attempt)) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}
		return resp, nil
	}
	return nil, errors.New("llm: stream request failed after retries")
}

func sleepWithContext(ctx context.Context, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

func mapGenerateRequest(req GenerateRequest, model string, stream bool) map[string]any {
	out := map[string]any{
		"model":    model,
		"messages": req.Messages,
		"stream":   stream,
	}
	if req.Temperature != nil {
		out["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		out["top_p"] = *req.TopP
	}
	if req.MaxTokens != nil {
		out["max_tokens"] = *req.MaxTokens
	}
	if req.ResponseFormat != nil && req.ResponseFormat.Type != "" {
		out["response_format"] = req.ResponseFormat
	}
	return out
}

func (p *openAICompatibleProvider) buildProviderError(status int, headers http.Header, raw []byte) error {
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    any    `json:"code"`
		} `json:"error"`
	}
	msg := strings.TrimSpace(string(raw))
	var code string
	var typ string
	if err := json.Unmarshal(raw, &parsed); err == nil {
		if parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		typ = parsed.Error.Type
		if parsed.Error.Code != nil {
			code = fmt.Sprintf("%v", parsed.Error.Code)
		}
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &ProviderError{
		Provider:   p.name,
		HTTPStatus: status,
		RequestID:  headers.Get("x-request-id"),
		Code:       code,
		Type:       typ,
		Message:    msg,
	}
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage Usage `json:"usage"`
}

type openAIChatStreamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

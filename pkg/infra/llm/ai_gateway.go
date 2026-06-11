package llm

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func randomHex32() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

const ProviderTypeAIGateway = "ai_gateway"

// AIGatewayConfig configures a provider that speaks the internal AI gateway protocol.
type AIGatewayConfig struct {
	BaseURL         string            `yaml:"base_url" json:"base_url"`
	APIKey          string            `yaml:"api_key" json:"api_key"`
	HTTPClient      *http.Client      `yaml:"-" json:"-"`
	HTTPTimeout     time.Duration     `yaml:"http_timeout" json:"http_timeout"`
	Headers         map[string]string `yaml:"headers" json:"headers"`
	DefaultModel    string            `yaml:"default_model" json:"default_model"`
	Retry           RetryConfig       `yaml:"retry" json:"retry"`
	GatewayProvider string            `yaml:"gateway_provider" json:"gateway_provider"`
	GatewayVersion  string            `yaml:"gateway_version" json:"gateway_version"`
	Path            string            `yaml:"path" json:"path"`

	// AI Gateway 业务鉴权字段
	ProductName   string `yaml:"product_name" json:"product_name"`
	IntentionCode string `yaml:"intention_code" json:"intention_code"`
	DefaultUID    string `yaml:"default_uid" json:"default_uid"`
}

type aiGatewayProvider struct {
	name            string
	baseURL         string
	apiKey          string
	httpClient      *http.Client
	headers         map[string]string
	model           string
	retry           RetryConfig
	gatewayProvider string
	gatewayVersion  string
	path            string
	productName     string
	intentionCode   string
	defaultUID      string
}

// NewAIGatewayProvider builds a provider for the internal AI gateway.
func NewAIGatewayProvider(name string, cfg AIGatewayConfig) (Provider, error) {
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("%w: provider name is empty", ErrInvalidConfig)
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("%w: ai_gateway base_url is required", ErrInvalidConfig)
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.HTTPTimeout
		if timeout <= 0 {
			timeout = 60 * time.Second
		}
		httpClient = &http.Client{Timeout: timeout}
	}
	headers := make(map[string]string, len(cfg.Headers))
	for k, v := range cfg.Headers {
		headers[k] = v
	}
	path := strings.TrimSpace(cfg.Path)
	if path == "" {
		path = "/v1/chat/completions"
	}
	return &aiGatewayProvider{
		name:            name,
		baseURL:         baseURL,
		apiKey:          cfg.APIKey,
		httpClient:      httpClient,
		headers:         headers,
		model:           cfg.DefaultModel,
		retry:           cfg.Retry.normalized(),
		gatewayProvider: cfg.GatewayProvider,
		gatewayVersion:  cfg.GatewayVersion,
		path:            path,
		productName:     cfg.ProductName,
		intentionCode:   cfg.IntentionCode,
		defaultUID:      cfg.DefaultUID,
	}, nil
}

func (p *aiGatewayProvider) Name() string { return p.name }

func (p *aiGatewayProvider) DefaultModel() string { return strings.TrimSpace(p.model) }

func (p *aiGatewayProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(p.model)
	}
	if model == "" {
		return nil, ErrModelRequired
	}
	payload := p.buildPayload(req, model, false)
	respBody, status, headers, err := p.doJSON(ctx, payload)
	if err != nil {
		return nil, err
	}
	if status >= http.StatusBadRequest {
		return nil, p.buildProviderError(status, headers, respBody)
	}
	var resp openAIChatResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("llm: decode ai_gateway response: %w", err)
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

func (p *aiGatewayProvider) GenerateStream(ctx context.Context, req GenerateRequest) (<-chan StreamEvent, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = strings.TrimSpace(p.model)
	}
	if model == "" {
		return nil, ErrModelRequired
	}
	payload := p.buildPayload(req, model, true)
	resp, err := p.doStreamRequest(ctx, payload)
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
				ch <- StreamEvent{Provider: p.name, Model: model, Done: true}
				return
			}
			var chunk openAIChatStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- StreamEvent{Provider: p.name, Model: model, Err: fmt.Errorf("llm: decode stream chunk: %w", err)}
				return
			}
			eventModel := chunk.Model
			if eventModel == "" {
				eventModel = model
			}
			if len(chunk.Choices) == 0 && chunk.Usage != nil {
				ch <- StreamEvent{Provider: p.name, Model: eventModel, Usage: chunk.Usage}
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
			ch <- StreamEvent{Provider: p.name, Model: model, Err: err}
		}
	}()

	return ch, nil
}

// buildPayload 将 GenerateRequest 映射为 AI 网关请求体。
func (p *aiGatewayProvider) buildPayload(req GenerateRequest, model string, stream bool) map[string]any {
	var contextStr string
	var messages []map[string]string

	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			contextStr = msg.Content
			continue
		}
		m := map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Name != "" {
			m["name"] = msg.Name
		}
		messages = append(messages, m)
	}

	payload := map[string]any{
		"stream":   stream,
		"context":  contextStr,
		"messages": messages,
		"provider": p.gatewayProvider,
		"model":    model,
	}

	if p.gatewayVersion != "" {
		payload["version"] = p.gatewayVersion
	}

	baseLLMArgs := map[string]any{}
	if req.Temperature != nil {
		baseLLMArgs["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		baseLLMArgs["top_p"] = *req.TopP
	}
	if req.MaxTokens != nil {
		baseLLMArgs["max_tokens"] = *req.MaxTokens
	}
	if len(baseLLMArgs) > 0 {
		payload["base_llm_arguments"] = baseLLMArgs
	}

	if req.ResponseFormat != nil && req.ResponseFormat.Type == "json_object" {
		baseLLMArgs["response_format"] = map[string]string{"type": "json_object"}
		payload["base_llm_arguments"] = baseLLMArgs
	}

	return payload
}

func (p *aiGatewayProvider) doJSON(ctx context.Context, payload any) ([]byte, int, http.Header, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("llm: encode request: %w", err)
	}
	maxAttempts := p.retry.maxAttempts()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.path, bytes.NewReader(body))
		if reqErr != nil {
			return nil, 0, nil, reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "application/json")
		p.applyHeaders(httpReq)

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
	return nil, 0, nil, errors.New("llm: ai_gateway request failed after retries")
}

func (p *aiGatewayProvider) doStreamRequest(ctx context.Context, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("llm: encode request: %w", err)
	}
	maxAttempts := p.retry.maxAttempts()
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		httpReq, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+p.path, bytes.NewReader(body))
		if reqErr != nil {
			return nil, reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Accept", "text/event-stream")
		p.applyHeaders(httpReq)

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
	return nil, errors.New("llm: ai_gateway stream failed after retries")
}

func (p *aiGatewayProvider) applyHeaders(req *http.Request) {
	if strings.TrimSpace(p.apiKey) != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	if p.productName != "" {
		req.Header.Set("AI-Gateway-Product-Name", p.productName)
	}
	if p.intentionCode != "" {
		req.Header.Set("AI-Gateway-Intention-Code", p.intentionCode)
	}
	if p.defaultUID != "" {
		req.Header.Set("AI-Gateway-Uid", p.defaultUID)
	}
	// X-Action-Id: 每次请求生成 32 位随机十六进制数
	req.Header.Set("X-Action-Id", randomHex32())
	// Client-Request-Id: 复用 uuid 作为全链路追踪 ID
	req.Header.Set("Client-Request-Id", randomHex32())
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
}

func (p *aiGatewayProvider) buildProviderError(status int, headers http.Header, raw []byte) error {
	msg := strings.TrimSpace(string(raw))
	if msg == "" {
		msg = http.StatusText(status)
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    any    `json:"code"`
		} `json:"error"`
		Msg string `json:"msg"`
	}
	var code, typ string
	if err := json.Unmarshal(raw, &parsed); err == nil {
		if parsed.Error.Message != "" {
			msg = parsed.Error.Message
			typ = parsed.Error.Type
			if parsed.Error.Code != nil {
				code = fmt.Sprintf("%v", parsed.Error.Code)
			}
		} else if parsed.Msg != "" {
			msg = parsed.Msg
		}
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

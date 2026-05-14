package llm

import (
	"context"
	"strings"
)

// Ask sends one user prompt and returns the normalized response.
func (c *Client) Ask(ctx context.Context, prompt string, opts ...CallOption) (*GenerateResponse, error) {
	return c.Generate(ctx, GenerateRequest{
		Messages: []Message{UserMessage(prompt)},
	}, opts...)
}

// AskText sends one user prompt and returns plain text content.
func (c *Client) AskText(ctx context.Context, prompt string, opts ...CallOption) (string, error) {
	resp, err := c.Ask(ctx, prompt, opts...)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", nil
	}
	return strings.TrimSpace(resp.Content), nil
}

// AskStream streams a one-prompt request.
func (c *Client) AskStream(ctx context.Context, prompt string, opts ...CallOption) (<-chan StreamEvent, error) {
	return c.GenerateStream(ctx, GenerateRequest{
		Messages: []Message{UserMessage(prompt)},
	}, opts...)
}

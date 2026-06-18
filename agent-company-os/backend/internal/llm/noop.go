package llm

import "context"

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GenerateRequest struct {
	Messages []Message
}

type GenerateResponse struct {
	Text  string                 `json:"text"`
	Usage map[string]interface{} `json:"usage,omitempty"`
}

type Provider interface {
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}

type NoopProvider struct{}

func (NoopProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	return &GenerateResponse{Text: "Phase 0 noop LLM provider: no real model call was made."}, nil
}

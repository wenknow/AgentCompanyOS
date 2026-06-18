package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type DeepSeekConfig struct {
	APIKey          string
	BaseURL         string
	Model           string
	ReasoningEffort string
	Thinking        string
	Timeout         time.Duration
}

type DeepSeekProvider struct {
	cfg    DeepSeekConfig
	client *http.Client
}

func NewDeepSeekProvider(cfg DeepSeekConfig) *DeepSeekProvider {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.deepseek.com"
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Model == "" {
		cfg.Model = "deepseek-v4-pro"
	}
	if cfg.ReasoningEffort == "" {
		cfg.ReasoningEffort = "high"
	}
	if cfg.Thinking == "" {
		cfg.Thinking = "enabled"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 180 * time.Second
	}
	return &DeepSeekProvider{cfg: cfg, client: &http.Client{Timeout: cfg.Timeout}}
}

func (p *DeepSeekProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body := map[string]interface{}{
		"model":            p.cfg.Model,
		"messages":         req.Messages,
		"stream":           false,
		"reasoning_effort": p.cfg.ReasoningEffort,
	}
	if strings.EqualFold(p.cfg.Thinking, "enabled") {
		body["thinking"] = map[string]string{"type": "enabled"}
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &Error{Class: ErrorInvalidResponse, Err: err}
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, &Error{Class: ErrorUnavailable, Err: err}
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &Error{Class: ErrorUnavailable, Err: err}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		io.Copy(io.Discard, resp.Body)
		return nil, &Error{Class: classForStatus(resp.StatusCode), Status: resp.StatusCode}
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]interface{} `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, &Error{Class: ErrorInvalidResponse, Err: err}
	}
	if len(decoded.Choices) == 0 || strings.TrimSpace(decoded.Choices[0].Message.Content) == "" {
		return nil, &Error{Class: ErrorInvalidResponse, Err: fmt.Errorf("missing assistant content")}
	}
	return &GenerateResponse{Text: strings.TrimSpace(decoded.Choices[0].Message.Content), Usage: decoded.Usage}, nil
}

func classForStatus(status int) string {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return ErrorAuthFailed
	case status == http.StatusTooManyRequests:
		return ErrorRateLimited
	case status == http.StatusPaymentRequired || status >= 500:
		return ErrorUnavailable
	default:
		return ErrorInvalidResponse
	}
}

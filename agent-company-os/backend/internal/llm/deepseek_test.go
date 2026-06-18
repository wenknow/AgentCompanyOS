package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeepSeekRequestBodyAndAuthorization(t *testing.T) {
	var gotAuth string
	var gotBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"draft output"}}],"usage":{"prompt_tokens":7,"completion_tokens":3}}`))
	}))
	defer server.Close()

	provider := NewDeepSeekProvider(DeepSeekConfig{APIKey: "test-secret-key", BaseURL: server.URL, Model: "deepseek-v4-pro", ReasoningEffort: "high", Thinking: "enabled", Timeout: time.Second})
	resp, err := provider.Generate(context.Background(), GenerateRequest{Messages: []Message{{Role: "user", Content: SanitizeText("use Bearer abcdefghijklmnop")}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "draft output" || resp.Usage["prompt_tokens"].(float64) != 7 {
		t.Fatalf("unexpected response %#v", resp)
	}
	if gotAuth != "Bearer test-secret-key" {
		t.Fatalf("authorization header not set")
	}
	if gotBody["model"] != "deepseek-v4-pro" || gotBody["stream"] != false || gotBody["reasoning_effort"] != "high" {
		t.Fatalf("unexpected body %#v", gotBody)
	}
	if _, ok := gotBody["thinking"].(map[string]interface{}); !ok {
		t.Fatalf("thinking not enabled in body %#v", gotBody)
	}
	messages := gotBody["messages"].([]interface{})
	content := messages[0].(map[string]interface{})["content"].(string)
	if strings.Contains(content, "abcdefghijklmnop") || !strings.Contains(content, "[REDACTED]") {
		t.Fatalf("content was not sanitized: %q", content)
	}
}

func TestDeepSeekErrorMappingDoesNotLeakAuthorization(t *testing.T) {
	cases := []struct {
		status int
		class  string
	}{
		{http.StatusUnauthorized, ErrorAuthFailed},
		{http.StatusPaymentRequired, ErrorUnavailable},
		{http.StatusTooManyRequests, ErrorRateLimited},
		{http.StatusInternalServerError, ErrorUnavailable},
	}
	for _, tc := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "upstream body with details", tc.status)
		}))
		provider := NewDeepSeekProvider(DeepSeekConfig{APIKey: "must-not-leak", BaseURL: server.URL, Timeout: time.Second})
		_, err := provider.Generate(context.Background(), GenerateRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
		server.Close()
		if ClassifyError(err) != tc.class {
			t.Fatalf("status %d classified as %q: %v", tc.status, ClassifyError(err), err)
		}
		if strings.Contains(err.Error(), "must-not-leak") || strings.Contains(err.Error(), "upstream body") {
			t.Fatalf("error leaked sensitive details: %v", err)
		}
	}
}

func TestDeepSeekInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()
	provider := NewDeepSeekProvider(DeepSeekConfig{APIKey: "key", BaseURL: server.URL, Timeout: time.Second})
	_, err := provider.Generate(context.Background(), GenerateRequest{Messages: []Message{{Role: "user", Content: "hello"}}})
	if ClassifyError(err) != ErrorInvalidResponse {
		t.Fatalf("unexpected error class %q", ClassifyError(err))
	}
}

func TestSanitizeText(t *testing.T) {
	privateKey := "-----BEGIN PRIVATE KEY-----\nsecret\n-----END PRIVATE KEY-----"
	raw := "api_key=abcdef123456 Bearer qwertyuiopasdfgh 0x" + strings.Repeat("a", 64) + " " + privateKey
	got := SanitizeText(raw)
	for _, secret := range []string{"abcdef123456", "qwertyuiopasdfgh", strings.Repeat("a", 64), "BEGIN PRIVATE KEY"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q was not redacted in %q", secret, got)
		}
	}
}

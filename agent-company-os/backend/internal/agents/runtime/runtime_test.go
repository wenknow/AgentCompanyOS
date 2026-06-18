package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
)

type fakeProvider struct {
	resp  *llm.GenerateResponse
	err   error
	calls int
	req   llm.GenerateRequest
}

func (f *fakeProvider) Generate(ctx context.Context, req llm.GenerateRequest) (*llm.GenerateResponse, error) {
	f.calls++
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func TestLLMRuntimeUsesProviderOnSuccess(t *testing.T) {
	provider := &fakeProvider{resp: &llm.GenerateResponse{Text: "llm draft", Usage: map[string]interface{}{"total_tokens": 12}}}
	rt := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	out, err := rt.Run(context.Background(), input("backend", "design a REST API with token=secret123456789"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Summary != "llm draft" || out.Details["fallback_used"] != false || out.Details["provider"] != "deepseek" {
		t.Fatalf("unexpected output %#v", out)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider call, got %d", provider.calls)
	}
	if len(provider.req.Messages) != 2 || provider.req.Messages[0].Role != "system" {
		t.Fatalf("unexpected messages %#v", provider.req.Messages)
	}
	if provider.req.Messages[1].Content == "" || contains(provider.req.Messages[1].Content, "secret123456789") {
		t.Fatalf("task prompt was not sanitized: %#v", provider.req.Messages[1])
	}
}

func TestLLMRuntimeFallsBackOnProviderError(t *testing.T) {
	provider := &fakeProvider{err: &llm.Error{Class: llm.ErrorRateLimited}}
	rt := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	out, err := rt.Run(context.Background(), input("product", "define onboarding story"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Details["fallback_used"] != true || out.Details["error_class"] != llm.ErrorRateLimited {
		t.Fatalf("unexpected fallback details %#v", out.Details)
	}
	if out.Summary == "" || provider.calls != 1 {
		t.Fatalf("fallback did not run as expected")
	}
}

func TestLLMRuntimeFallsBackOnUnknownProviderError(t *testing.T) {
	provider := &fakeProvider{err: errors.New("network down")}
	rt := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	out, err := rt.Run(context.Background(), input("cto", "make a technical plan"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Details["error_class"] != llm.ErrorUnavailable {
		t.Fatalf("unexpected error class %#v", out.Details)
	}
}

func TestLLMRuntimeDoesNotCallProviderForCriticalSensitiveTask(t *testing.T) {
	provider := &fakeProvider{resp: &llm.GenerateResponse{Text: "should not be used"}}
	rt := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	out, err := rt.Run(context.Background(), input("backend", "访问钱包私钥 and move funds"))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 {
		t.Fatalf("provider was called for critical sensitive task")
	}
	if out.Details["fallback_used"] != true || out.Details["error_class"] != "critical_sensitive_task" {
		t.Fatalf("unexpected details %#v", out.Details)
	}
}

func input(agentName, title string) AgentRunInput {
	return AgentRunInput{Agent: model.Agent{Name: agentName, Role: "Test Agent", Description: "Tests runtime behavior."}, Task: model.Task{Title: title}}
}

func contains(s, needle string) bool { return strings.Contains(s, needle) }

func TestRouterRuntimeRoutesCodingToDisabledClaudeAdapter(t *testing.T) {
	provider := &fakeProvider{resp: &llm.GenerateResponse{Text: "deepseek draft"}}
	deep := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	claude := NewClaudeCodeAdapter(ClaudeCodeConfig{Command: "definitely-missing-claude-command", Enabled: false, Workdir: "."})
	rt := NewRouterRuntime(RouterConfig{DeepSeekRuntime: deep, DeepSeekStatus: ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: true, Enabled: true, Available: true}, CodingRuntime: "claude_code_local", ClaudeCode: claude})
	out, err := rt.Run(context.Background(), input("coding", "add a local unit test"))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 {
		t.Fatalf("deepseek provider called for coding task")
	}
	if out.Details["provider"] != "claude_code_local" || out.Details["error_class"] != "coding_runtime_disabled" {
		t.Fatalf("unexpected coding route %#v", out.Details)
	}
	if out.Details["external_execution"] != false {
		t.Fatalf("disabled claude adapter executed externally")
	}
}

func TestRouterRuntimeRoutesOrdinaryTaskToDeepSeek(t *testing.T) {
	provider := &fakeProvider{resp: &llm.GenerateResponse{Text: "deepseek draft"}}
	deep := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	rt := NewRouterRuntime(RouterConfig{DeepSeekRuntime: deep, DeepSeekStatus: ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: true, Enabled: true, Available: true}})
	out, err := rt.Run(context.Background(), input("product", "define onboarding"))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 1 || out.Summary != "deepseek draft" {
		t.Fatalf("ordinary task did not use deepseek: calls=%d out=%#v", provider.calls, out)
	}
}

func TestRouterRuntimeDoesNotCallProvidersForHighRiskCoding(t *testing.T) {
	provider := &fakeProvider{resp: &llm.GenerateResponse{Text: "should not run"}}
	deep := NewLLMRuntime(provider, "deepseek", "deepseek-v4-pro")
	claude := NewClaudeCodeAdapter(ClaudeCodeConfig{Command: "definitely-missing-claude-command", Enabled: true, Workdir: "."})
	rt := NewRouterRuntime(RouterConfig{DeepSeekRuntime: deep, DeepSeekStatus: ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: true, Enabled: true, Available: true}, CodingRuntime: "claude_code_local", ClaudeCode: claude})
	out, err := rt.Run(context.Background(), input("coding", "deploy production code"))
	if err != nil {
		t.Fatal(err)
	}
	if provider.calls != 0 || out.Details["provider"] != "approval_only" {
		t.Fatalf("high-risk coding task was not approval-only: calls=%d details=%#v", provider.calls, out.Details)
	}
}

func TestClaudeCodeAdapterRejectsInvalidWorkdir(t *testing.T) {
	adapter := NewClaudeCodeAdapter(ClaudeCodeConfig{Command: "claude", Workdir: "/", Enabled: true})
	if adapter.Status().ErrorClass != "invalid_workdir" || adapter.Status().Available {
		t.Fatalf("unexpected status %#v", adapter.Status())
	}
}

func TestClaudeCodeAdapterAllowsConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	adapter := NewClaudeCodeAdapter(ClaudeCodeConfig{Command: "definitely-missing-claude-command", Workdir: root, AllowedRoot: root, Enabled: true})
	if adapter.Status().ErrorClass == "invalid_workdir" {
		t.Fatalf("workdir inside configured allowed root was rejected: %#v", adapter.Status())
	}
	if adapter.Status().ErrorClass != "command_unavailable" {
		t.Fatalf("expected command availability check after root validation, got %#v", adapter.Status())
	}
}

func TestClaudeCodeAdapterRejectsPathOutsideConfiguredRoot(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	adapter := NewClaudeCodeAdapter(ClaudeCodeConfig{Command: "definitely-missing-claude-command", Workdir: outside, AllowedRoot: root, Enabled: true})
	if adapter.Status().ErrorClass != "invalid_workdir" || adapter.Status().Available {
		t.Fatalf("workdir outside configured allowed root was not rejected: %#v", adapter.Status())
	}
}

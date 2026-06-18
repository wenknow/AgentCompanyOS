package runtime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/llm"
	"github.com/agentcompany/agent-company-os/backend/internal/model"
	"github.com/agentcompany/agent-company-os/backend/internal/risk"
)

type AgentRunInput struct {
	Agent model.Agent
	Task  model.Task
}

type AgentRunOutput struct {
	Summary string                 `json:"summary"`
	Details map[string]interface{} `json:"details"`
}

type AgentRuntime interface {
	Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error)
	Status() Status
}

type Status struct {
	Provider        string       `json:"provider"`
	Model           string       `json:"model"`
	Configured      bool         `json:"configured"`
	FallbackMode    bool         `json:"fallback_mode"`
	DeepSeek        ToolStatus   `json:"deepseek"`
	ClaudeCodeLocal ToolStatus   `json:"claude_code_local"`
	Tools           []ToolStatus `json:"tools"`
}

type ToolStatus struct {
	Name           string `json:"name"`
	Runtime        string `json:"runtime"`
	Configured     bool   `json:"configured"`
	Enabled        bool   `json:"enabled"`
	Available      bool   `json:"available"`
	FallbackMode   bool   `json:"fallback_mode"`
	ErrorClass     string `json:"error_class,omitempty"`
	Command        string `json:"command,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

type RuleBasedRuntime struct {
	status Status
}

func NewRuleBasedRuntime() *RuleBasedRuntime {
	return &RuleBasedRuntime{status: Status{Provider: "rule_based", Model: "", Configured: true, FallbackMode: true}}
}

func NewRuleBasedFallbackRuntime(provider, model string, configured bool) *RuleBasedRuntime {
	if provider == "" {
		provider = "rule_based"
	}
	return &RuleBasedRuntime{status: Status{Provider: provider, Model: model, Configured: configured, FallbackMode: true}}
}

func (r *RuleBasedRuntime) Status() Status { return r.status }

func (r *RuleBasedRuntime) Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	title := strings.TrimSpace(input.Task.Title)
	var summary string
	switch input.Agent.Name {
	case "backend":
		summary = fmt.Sprintf("Backend plan for %q: define API endpoints, PostgreSQL schema, service/repository boundaries, tests, and a Codex implementation prompt draft. No external code execution is performed.", title)
	case "designer":
		summary = fmt.Sprintf("Design direction for %q: remove generic AI-looking UI, define a refined Apple-inspired visual system, clear hierarchy, restrained color, precise spacing, typography, motion, empty/loading/error states, and design QA checks before frontend implementation.", title)
	case "frontend":
		summary = fmt.Sprintf("Frontend plan for %q: define user flows, component boundaries, accessibility checks, responsive states, and frontend tests. No external code execution is performed.", title)
	case "qa":
		summary = fmt.Sprintf("QA review for %q: define acceptance checks, regression coverage, failure modes, and release-quality notes.", title)
	case "devops":
		summary = fmt.Sprintf("DevOps draft for %q: outline infrastructure, rollout, rollback, monitoring, and approval gates. No deployment is performed.", title)
	case "growth":
		summary = fmt.Sprintf("Growth draft for %q: prepare positioning, channels, experiment hypotheses, metrics, and review gates. No external outreach is performed.", title)
	case "sales":
		summary = fmt.Sprintf("Sales draft for %q: prepare ICP notes, discovery questions, outreach copy, and CRM fields. No message is sent.", title)
	case "finance":
		summary = fmt.Sprintf("Finance draft for %q: outline budget impact, runway considerations, assumptions, and approval needs. No funds are moved.", title)
	case "coding":
		summary = fmt.Sprintf("Coding prompt prepared for %q. Claude Code execution is controlled by the runtime router and remains disabled unless explicitly configured.", title)
	case "content":
		summary = fmt.Sprintf("Content draft for %q prepared as an internal draft only. It will not be published and requires Founder approval before any public action.", title)
	case "compliance":
		summary = fmt.Sprintf("Compliance review for %q: avoid guaranteed returns, direct investment advice, exaggerated claims, and add clear risk notes.", title)
	case "cto":
		summary = fmt.Sprintf("Technical plan for %q: clarify architecture, risks, rollback strategy, testing, and approval gates before production changes.", title)
	case "product":
		summary = fmt.Sprintf("Product breakdown for %q: define user story, acceptance criteria, scope boundaries, and roadmap impact.", title)
	default:
		summary = fmt.Sprintf("Chief of Staff routed %q, identified next steps, blockers, reporting needs, and approval requirements.", title)
	}
	return &AgentRunOutput{Summary: llm.SanitizeText(summary), Details: map[string]interface{}{"phase": "phase_0", "provider": "rule_based", "model": "", "fallback_used": false, "external_execution": false, "tools_used": []string{}}}, nil
}

type LLMRuntime struct {
	provider llm.Provider
	fallback *RuleBasedRuntime
	status   Status
}

func NewLLMRuntime(provider llm.Provider, providerName, model string) *LLMRuntime {
	return &LLMRuntime{provider: provider, fallback: NewRuleBasedRuntime(), status: Status{Provider: providerName, Model: model, Configured: true, FallbackMode: false}}
}

func (r *LLMRuntime) Status() Status { return r.status }

func (r *LLMRuntime) Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error) {
	if isCriticalSensitive(input.Task.Title) {
		return approvalOnlyOutput(input, "critical_sensitive_task"), nil
	}
	resp, err := r.provider.Generate(ctx, llm.GenerateRequest{Messages: []llm.Message{
		{Role: "system", Content: systemPrompt(input.Agent)},
		{Role: "user", Content: userPrompt(input)},
	}})
	if err != nil {
		return r.fallbackWithDetails(ctx, input, llm.ClassifyError(err), nil)
	}
	return &AgentRunOutput{Summary: llm.SanitizeText(resp.Text), Details: map[string]interface{}{
		"phase":              "phase_1",
		"provider":           r.status.Provider,
		"model":              r.status.Model,
		"fallback_used":      false,
		"error_class":        "",
		"usage":              resp.Usage,
		"external_execution": false,
		"tools_used":         []string{},
	}}, nil
}

func (r *LLMRuntime) fallbackWithDetails(ctx context.Context, input AgentRunInput, errorClass string, usage map[string]interface{}) (*AgentRunOutput, error) {
	out, err := r.fallback.Run(ctx, input)
	if err != nil {
		return nil, err
	}
	out.Details["phase"] = "phase_1"
	out.Details["provider"] = r.status.Provider
	out.Details["model"] = r.status.Model
	out.Details["fallback_used"] = true
	out.Details["error_class"] = errorClass
	out.Details["usage"] = usage
	out.Details["external_execution"] = false
	out.Details["tools_used"] = []string{}
	return out, nil
}

type RouterConfig struct {
	DeepSeekRuntime AgentRuntime
	DeepSeekStatus  ToolStatus
	CodingRuntime   string
	ClaudeCode      *ClaudeCodeAdapter
}

type RouterRuntime struct {
	fallback      *RuleBasedRuntime
	deepseek      AgentRuntime
	deepseekTool  ToolStatus
	codingRuntime string
	claude        *ClaudeCodeAdapter
}

func NewRouterRuntime(cfg RouterConfig) *RouterRuntime {
	fallback := NewRuleBasedRuntime()
	deepseek := cfg.DeepSeekRuntime
	if deepseek == nil {
		deepseek = fallback
	}
	return &RouterRuntime{fallback: fallback, deepseek: deepseek, deepseekTool: cfg.DeepSeekStatus, codingRuntime: cfg.CodingRuntime, claude: cfg.ClaudeCode}
}

func (r *RouterRuntime) Status() Status {
	deep := r.deepseekTool
	if deep.Name == "" {
		deep = ToolStatus{Name: "deepseek", Runtime: "deepseek", Configured: false, Available: false, FallbackMode: true}
	}
	claude := ToolStatus{Name: "claude_code_local", Runtime: r.codingRuntime, FallbackMode: true}
	if r.claude != nil {
		claude = r.claude.Status()
	}
	return Status{Provider: "runtime_router", Model: deep.Command, Configured: true, FallbackMode: deep.FallbackMode || !claude.Enabled, DeepSeek: deep, ClaudeCodeLocal: claude, Tools: []ToolStatus{deep, claude}}
}

func (r *RouterRuntime) Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error) {
	riskResult := risk.Detect(input.Task.Title)
	if riskResult.Level == "critical" || (riskResult.Level == "high" && input.Agent.Name == "coding") {
		return approvalOnlyOutput(input, riskResult.Level+"_risk_task"), nil
	}
	if input.Agent.Name == "coding" {
		if r.claude == nil {
			return codingUnavailableOutput(input, "coding_runtime_unavailable", manualPrompt(input)), nil
		}
		return r.claude.Run(ctx, input)
	}
	return r.deepseek.Run(ctx, input)
}

type ClaudeCodeConfig struct {
	Command        string
	Timeout        time.Duration
	Workdir        string
	AllowedRoot    string
	MaxOutputBytes int
	Enabled        bool
}

type ClaudeCodeAdapter struct {
	command        string
	timeout        time.Duration
	workdir        string
	allowedRoot    string
	maxOutputBytes int
	enabled        bool
	available      bool
	errorClass     string
}

func NewClaudeCodeAdapter(cfg ClaudeCodeConfig) *ClaudeCodeAdapter {
	if cfg.Command == "" {
		cfg.Command = "claude"
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 15 * time.Minute
	}
	if cfg.Workdir == "" {
		cfg.Workdir = ".."
	}
	if cfg.MaxOutputBytes <= 0 {
		cfg.MaxOutputBytes = 200000
	}
	abs, err := filepath.Abs(cfg.Workdir)
	allowedRoot, rootErr := resolveAllowedRoot(cfg.AllowedRoot)
	adapter := &ClaudeCodeAdapter{command: cfg.Command, timeout: cfg.Timeout, workdir: abs, allowedRoot: allowedRoot, maxOutputBytes: cfg.MaxOutputBytes, enabled: cfg.Enabled}
	if err != nil || rootErr != nil || !pathAllowed(abs, allowedRoot) {
		adapter.errorClass = "invalid_workdir"
		return adapter
	}
	if _, err := exec.LookPath(cfg.Command); err != nil {
		adapter.errorClass = "command_unavailable"
		return adapter
	}
	adapter.available = true
	return adapter
}

func (a *ClaudeCodeAdapter) Status() ToolStatus {
	return ToolStatus{Name: "claude_code_local", Runtime: "claude_code_local", Configured: true, Enabled: a.enabled, Available: a.available, FallbackMode: !a.enabled || !a.available, ErrorClass: a.errorClass, Command: a.command, TimeoutSeconds: int(a.timeout.Seconds())}
}

func (a *ClaudeCodeAdapter) Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error) {
	prompt := manualPrompt(input)
	if !a.enabled {
		return codingUnavailableOutput(input, "coding_runtime_disabled", prompt), nil
	}
	if !a.available {
		if a.errorClass == "" {
			a.errorClass = "coding_runtime_unavailable"
		}
		return codingUnavailableOutput(input, a.errorClass, prompt), nil
	}
	if !pathAllowed(a.workdir, a.allowedRoot) {
		return codingUnavailableOutput(input, "invalid_workdir", prompt), nil
	}
	ctx, cancel := context.WithTimeout(ctx, a.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, a.command, "--permission-mode", "acceptEdits", "--no-session-persistence", "--append-system-prompt", claudeSystemPrompt(a.workdir), "-p", prompt)
	cmd.Dir = a.workdir
	var stdout, stderr limitedBuffer
	stdout.limit = a.maxOutputBytes
	stderr.limit = a.maxOutputBytes / 10
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	combined := llm.SanitizeText(strings.TrimSpace(stdout.String() + "\n" + stderr.String()))
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return codingUnavailableOutput(input, "timeout", prompt), nil
	}
	if limited, resetAt, retryAfter := detectClaudeLimit(combined); limited {
		out := codingUnavailableOutput(input, "claude_rate_limited", prompt)
		out.Summary = "Claude Code 使用额度已达上限，需要等待 reset 后继续。"
		out.Details["stderr_summary"] = combined
		out.Details["reset_at"] = resetAt
		out.Details["retry_after_seconds"] = retryAfter
		return out, nil
	}
	if err != nil {
		out := codingUnavailableOutput(input, "execution_failed", prompt)
		out.Details["stderr_summary"] = combined
		return out, nil
	}
	text := llm.SanitizeText(stdout.String())
	if strings.TrimSpace(text) == "" {
		text = "Claude Code completed without textual output."
	}
	if limited, resetAt, retryAfter := detectClaudeLimit(text); limited {
		out := codingUnavailableOutput(input, "claude_rate_limited", prompt)
		out.Summary = "Claude Code 使用额度已达上限，需要等待 reset 后继续。"
		out.Details["stderr_summary"] = text
		out.Details["reset_at"] = resetAt
		out.Details["retry_after_seconds"] = retryAfter
		return out, nil
	}
	return &AgentRunOutput{Summary: text, Details: map[string]interface{}{"phase": "phase_1", "provider": "claude_code_local", "model": "claude_code", "fallback_used": false, "error_class": "", "manual_prompt": prompt, "external_execution": true, "tools_used": []string{"claude_code_local"}}}, nil
}

func detectClaudeLimit(text string) (bool, string, int64) {
	lower := strings.ToLower(text)
	if !strings.Contains(lower, "limit") && !strings.Contains(lower, "rate") && !strings.Contains(lower, "usage") {
		return false, "", 0
	}
	if !strings.Contains(lower, "reset") && !strings.Contains(lower, "try again") && !strings.Contains(lower, "too many") && !strings.Contains(lower, "rate") {
		return false, "", 0
	}
	retry := int64(3600)
	reset := time.Now().Add(time.Duration(retry) * time.Second)
	if d, ok := parseRelativeReset(lower); ok {
		retry = int64(d.Seconds())
		reset = time.Now().Add(d)
	} else if t, ok := parseClockReset(lower); ok {
		retry = int64(time.Until(t).Seconds())
		if retry < 60 {
			retry = 3600
			reset = time.Now().Add(time.Hour)
		} else {
			reset = t
		}
	}
	return true, reset.Format(time.RFC3339), retry
}

func parseRelativeReset(text string) (time.Duration, bool) {
	re := regexp.MustCompile(`(?i)(?:in|after)\s+(?:(\d+)\s*h(?:ours?)?)?\s*(?:(\d+)\s*m(?:in(?:ute)?s?)?)?`)
	m := re.FindStringSubmatch(text)
	if len(m) == 0 || (m[1] == "" && m[2] == "") {
		return 0, false
	}
	var d time.Duration
	if m[1] != "" {
		var h int
		_, _ = fmt.Sscanf(m[1], "%d", &h)
		d += time.Duration(h) * time.Hour
	}
	if m[2] != "" {
		var min int
		_, _ = fmt.Sscanf(m[2], "%d", &min)
		d += time.Duration(min) * time.Minute
	}
	if d < time.Minute {
		d = time.Hour
	}
	return d, true
}

func parseClockReset(text string) (time.Time, bool) {
	re := regexp.MustCompile(`(?i)(?:reset|resets|until|at)\D{0,20}(\d{1,2})(?::(\d{2}))?\s*(am|pm)?`)
	m := re.FindStringSubmatch(text)
	if len(m) == 0 {
		return time.Time{}, false
	}
	var hour int
	_, _ = fmt.Sscanf(m[1], "%d", &hour)
	minute := 0
	if m[2] != "" {
		_, _ = fmt.Sscanf(m[2], "%d", &minute)
	}
	ampm := strings.ToLower(m[3])
	if ampm == "pm" && hour < 12 {
		hour += 12
	}
	if ampm == "am" && hour == 12 {
		hour = 0
	}
	now := time.Now()
	reset := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !reset.After(now.Add(time.Minute)) {
		reset = reset.Add(24 * time.Hour)
	}
	return reset, true
}

type limitedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.limit <= 0 || b.Len() >= b.limit {
		return len(p), nil
	}
	remaining := b.limit - b.Len()
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		return len(p), nil
	}
	_, _ = b.Buffer.Write(p)
	return len(p), nil
}

func resolveAllowedRoot(configured string) (string, error) {
	if strings.TrimSpace(configured) != "" {
		return filepath.Abs(configured)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root := filepath.Clean(cwd)
	if filepath.Base(root) == "backend" {
		root = filepath.Dir(root)
	}
	return root, nil
}

func pathAllowed(abs, allowedRoot string) bool {
	clean := filepath.Clean(abs)
	root := filepath.Clean(allowedRoot)
	if !filepath.IsAbs(clean) || !filepath.IsAbs(root) || clean == string(filepath.Separator) || root == string(filepath.Separator) {
		return false
	}
	rel, err := filepath.Rel(root, clean)
	if err != nil {
		return false
	}
	return rel == "." || (!filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func codingUnavailableOutput(input AgentRunInput, errorClass, prompt string) *AgentRunOutput {
	return &AgentRunOutput{Summary: "Claude Code local execution is not available; a manual coding prompt artifact was prepared instead.", Details: map[string]interface{}{"phase": "phase_1", "provider": "claude_code_local", "model": "claude_code", "fallback_used": true, "error_class": errorClass, "manual_prompt": prompt, "external_execution": false, "tools_used": []string{}}}
}

func approvalOnlyOutput(input AgentRunInput, errorClass string) *AgentRunOutput {
	return &AgentRunOutput{Summary: "Approval is required before this task can use model or coding runtime execution. No external action was performed.", Details: map[string]interface{}{"phase": "phase_1", "provider": "approval_only", "model": "", "fallback_used": true, "error_class": errorClass, "external_execution": false, "tools_used": []string{}}}
}

func claudeSystemPrompt(workdir string) string {
	return strings.Join([]string{
		"You are running as the coding worker for AgentCompanyOS.",
		"Work only in this checkout: " + workdir,
		"Do not create or use git worktrees, background agents, branches, pull requests, commits, or sub-agent sessions.",
		"Do not write implementation files under .claude/worktrees or any other temporary worktree directory.",
		"Make edits directly in the current working directory checkout and leave changes in the working tree for the operator to inspect.",
		"If no safe code change is possible, say exactly why instead of creating a separate worktree.",
	}, "\n")
}

func manualPrompt(input AgentRunInput) string {
	return llm.SanitizeText(strings.Join([]string{
		"You are Claude Code working inside AgentCompanyOS.",
		"Implement only the requested low-risk local coding task.",
		"Edit the current checkout directly. Do not create or use .claude/worktrees, git worktrees, background agents, branches, commits, or pull requests.",
		"Do not deploy, publish, trade, move funds, access wallets, or use production credentials.",
		"Before editing, inspect the repository and follow existing project patterns.",
		"Task: " + strings.TrimSpace(input.Task.Title),
	}, "\n"))
}

func systemPrompt(agent model.Agent) string {
	lines := []string{
		"You are " + agent.Role + " in AgentCompanyOS Phase 1.",
		"Role description: " + agent.Description,
		"You produce concise internal draft output only.",
		"Never claim to execute GitHub, Codex, Claude, deployment, publishing, trading, wallet, production, or external tool actions.",
		"For high-risk work, describe approval needs and safe next steps only.",
		"用中文输出。第一句必须是50字以内的结论摘要，后面再给必要要点。",
		"Return practical, structured text suitable for the Founder to review.",
	}
	if agent.Name == "designer" {
		lines = append(lines,
			"你是资深产品设计师，审美参考 Apple Human Interface Guidelines 的克制、清晰、精致和空间感，但不要抄袭 Apple 品牌资产。",
			"输出必须具体到页面布局、信息层级、组件状态、字号层级、间距、颜色策略、动效节奏和前端可执行改造点。",
			"避免 AI 味：不要大面积渐变、玻璃拟态堆叠、空洞营销文案、无意义卡片、过度圆角和单调紫蓝色主题。",
		)
	}
	return strings.Join(lines, "\n")
}

func userPrompt(input AgentRunInput) string {
	return "Task title: " + llm.SanitizeText(strings.TrimSpace(input.Task.Title))
}

func isCriticalSensitive(text string) bool {
	r := risk.Detect(text)
	return r.Level == "critical" && llm.ContainsCriticalSensitiveText(text)
}

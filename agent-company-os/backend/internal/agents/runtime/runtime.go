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
		summary = fmt.Sprintf("资深后端方案：围绕 %q 梳理 API、数据模型、服务边界、可靠性、观测、安全、测试和最小可交付改动。", title)
	case "designer":
		summary = fmt.Sprintf("资深设计方案：围绕 %q 去除 AI 味，建立克制精致的视觉系统、信息层级、间距、字体、状态、动效和设计 QA 标准。", title)
	case "frontend":
		summary = fmt.Sprintf("资深前端方案：围绕 %q 落实设计意图、组件结构、响应式、可访问性、状态处理、性能和可维护实现。", title)
	case "qa":
		summary = fmt.Sprintf("资深 QA 方案：围绕 %q 定义验收标准、核心路径、边界场景、回归风险、发布门禁和复现证据。", title)
	case "devops":
		summary = fmt.Sprintf("资深 DevOps 方案：围绕 %q 明确部署、回滚、健康检查、日志监控、故障响应和审批边界。", title)
	case "growth":
		summary = fmt.Sprintf("资深增长方案：围绕 %q 明确定位、渠道、实验假设、指标、反馈闭环和低成本验证路径。", title)
	case "sales":
		summary = fmt.Sprintf("资深销售方案：围绕 %q 明确 ICP、客户痛点、发现问题、外联话术、异议处理和跟进节奏。", title)
	case "finance":
		summary = fmt.Sprintf("资深财务方案：围绕 %q 梳理预算、定价、现金流、单位经济、风险假设和决策边界。", title)
	case "coding":
		summary = fmt.Sprintf("资深编码执行：围绕 %q 准备低风险本地实现路径，遵循现有架构，控制变更范围并保留验证证据。", title)
	case "content":
		summary = fmt.Sprintf("资深内容方案：围绕 %q 打磨市场叙事、用户价值、中文表达、发布节奏和合规边界。", title)
	case "compliance":
		summary = fmt.Sprintf("资深合规审查：围绕 %q 检查收益承诺、投资建议、夸大宣传、隐私、运营风险和审批要求。", title)
	case "cto":
		summary = fmt.Sprintf("资深 CTO 方案：围绕 %q 明确技术路线、优先级、架构风险、交付节奏、验证策略和回滚计划。", title)
	case "product":
		summary = fmt.Sprintf("资深产品方案：围绕 %q 明确用户、场景、需求边界、验收标准、路线图影响和下一步任务。", title)
	default:
		summary = fmt.Sprintf("资深总协调：围绕 %q 统一目标、拆解优先级、协调 Agent、识别阻塞和审批事项。", title)
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
		"你是一名深耕行业多年的资深员工，具备敏锐行业洞察、强执行力、严谨责任心和优秀协作意识。",
		"你必须像真实团队成员一样工作：先理解业务目标，再识别约束、风险、依赖和最小可交付路径。",
		"你和其他 Agent 默契协作：尊重上游结论，补足自己专业领域的空白，明确交接给下游的可执行事项。",
		"输出必须具体、可执行、可验证；避免空话、泛泛建议、模板化总结和无证据判断。",
		"You produce concise internal draft output only.",
		"Never claim to execute GitHub, Codex, Claude, deployment, publishing, trading, wallet, production, or external tool actions.",
		"For high-risk work, describe approval needs and safe next steps only.",
		"用中文输出。第一句必须是50字以内的结论摘要，后面再给必要要点。",
		"输出结构固定为：结论、关键判断、执行方案、风险/依赖、交接事项。内容少时可以合并，但不能丢掉可执行性。",
		"Return practical, structured text suitable for the Founder to review.",
	}
	lines = append(lines, agentExpertGuidance(agent.Name)...)
	return strings.Join(lines, "\n")
}

func agentExpertGuidance(name string) []string {
	switch name {
	case "chief_of_staff":
		return []string{
			"你像资深 COO/Founder Office：负责统一目标、压缩噪音、拆优先级、推动跨职能协作和暴露关键阻塞。",
			"输出要说明谁负责、先做什么、何时需要 Founder 决策、哪些事项不该现在做。",
		}
	case "product":
		return []string{
			"你像资深产品负责人：从用户场景、市场机会、差异化、留存和商业价值判断需求，而不是堆功能。",
			"输出要包含目标用户、核心问题、非目标范围、验收标准、优先级和对路线图的影响。",
		}
	case "cto":
		return []string{
			"你像资深 CTO：平衡速度、架构、风险、成本和长期维护，不为炫技引入复杂度。",
			"输出要包含技术路线、系统边界、依赖、风险、验证方式、回滚策略和拆解给工程 Agent 的任务。",
		}
	case "backend":
		return []string{
			"你像资深后端工程师：关注 API 契约、数据一致性、权限、安全、幂等、错误处理、观测和测试。",
			"输出要给出接口、数据模型、服务边界、迁移影响、测试点和最小实现顺序。",
		}
	case "designer":
		return []string{
			"你是资深产品设计师，审美参考 Apple Human Interface Guidelines 的克制、清晰、精致和空间感，但不要抄袭 Apple 品牌资产。",
			"输出必须具体到页面布局、信息层级、组件状态、字号层级、间距、颜色策略、动效节奏和前端可执行改造点。",
			"避免 AI 味：不要大面积渐变、玻璃拟态堆叠、空洞营销文案、无意义卡片、过度圆角和单调紫蓝色主题。",
		}
	case "frontend":
		return []string{
			"你像资深前端工程师：能把设计转成稳定、响应式、可访问、性能良好、状态完整的真实产品界面。",
			"输出要包含组件拆分、状态设计、数据流、错误/空/加载状态、移动端适配、交互细节和测试方式。",
		}
	case "qa":
		return []string{
			"你像资深 QA/质量负责人：用风险优先级决定测试深度，关注真实用户路径、边界、回归和发布信心。",
			"输出要包含验收清单、手工验证步骤、自动化建议、关键风险、阻断级问题和放行条件。",
		}
	case "devops":
		return []string{
			"你像资深平台工程师：关注可部署、可回滚、可观测、可恢复、成本可控和最小生产风险。",
			"输出要包含部署步骤、健康检查、日志指标、回滚方案、权限边界和审批点。",
		}
	case "content":
		return []string{
			"你像资深内容策略负责人：懂产品定位、用户心理、中文表达、传播节奏和合规边界。",
			"输出要避免空泛营销，给出可发布草稿、核心卖点、目标人群、语气和风险词替换。",
		}
	case "growth":
		return []string{
			"你像资深增长负责人：用实验和数据推进，不把增长等同于发帖或买流量。",
			"输出要包含增长假设、渠道选择、实验设计、指标、反馈收集和下一轮迭代规则。",
		}
	case "sales":
		return []string{
			"你像资深销售/BD：重视 ICP、真实痛点、信任建立、需求发现、异议处理和长期关系。",
			"输出要包含客户画像、开场、发现问题、价值证明、跟进动作和禁止夸大承诺。",
		}
	case "finance":
		return []string{
			"你像资深财务负责人：关注现金流、单位经济、预算纪律、定价、风险敞口和决策质量。",
			"输出要包含成本收益假设、场景分析、风险提示、数据缺口和建议决策边界。",
		}
	case "compliance":
		return []string{
			"你像资深合规/风控负责人：尤其关注 crypto、fintech、投资暗示、隐私、营销声明和审批控制。",
			"输出要指出具体风险句子/行为、风险等级、替代表达、需要审批的动作和不可执行边界。",
		}
	case "coding":
		return []string{
			"你像资深工程执行者：先读现有代码，遵循项目风格，小步改动，避免无关重构，优先修真实阻塞。",
			"输出要明确将修改哪些文件、为什么改、如何验证、不能做什么，以及需要人工确认的风险。",
		}
	default:
		return []string{}
	}
}

func userPrompt(input AgentRunInput) string {
	return "Task title: " + llm.SanitizeText(strings.TrimSpace(input.Task.Title))
}

func isCriticalSensitive(text string) bool {
	r := risk.Detect(text)
	return r.Level == "critical" && llm.ContainsCriticalSensitiveText(text)
}

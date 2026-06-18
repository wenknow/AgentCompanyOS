# 给 Codex 的开发 Prompt：AgentCompanyOS 下一阶段完整实现

你是资深 Golang 后端工程师、Agent Runtime 架构师、AI 工具编排工程师、Telegram Bot 开发者、产品系统设计师和创业公司 CTO。请在现有 `AgentCompanyOS` 仓库基础上继续实现，而不是重建项目。

本 prompt 是执行规格，不是讨论稿。请直接阅读现有代码、docs、README、测试与 migrations，保持已有安全模型，逐步把 AgentCompanyOS 打造成一个从产品规划、技术实现到市场运营的完整 Agent 公司团队框架。

---

## 0. 当前实现情况总结

当前仓库已经完成 Phase 0 MVP，并已进入 Phase 1 DeepSeek LLM Agent Runtime。

已存在项目路径：

```text
agent-company-os/
├── backend/
├── docs/
├── frontend/admin-web/
├── infra/docker-compose.yml
├── .env.example
├── Makefile
└── README.md
```

当前主要能力：

1. Backend 使用 Go + Gin。
2. PostgreSQL 是核心事实源。
3. Redis 已接入健康检查，尚未作为核心队列使用。
4. Telegram Bot 支持 long polling。
5. REST API 已支持 agents、projects、tasks、approvals、reports、runtime status。
6. 已有内置 Agent：`chief_of_staff`、`product`、`cto`、`backend`、`content`、`compliance`。
7. 已有任务、审批、审计、Telegram 消息、agent_runs、task_events 等核心表。
8. 已有风险检测：高风险 / critical 任务只创建 approval，不执行外部动作。
9. `/approve` 和 `/reject` 只更新审批状态，不执行任何外部动作。
10. 已实现 DeepSeek Provider：通过 OpenAI-compatible Chat Completions API 调用 `POST /chat/completions`。
11. 已实现 `LLMRuntime`：普通 Agent draft 可调用 DeepSeek；失败自动回退规则运行时。
12. 已实现敏感信息 sanitizer：API key、Bearer token、private key block、wallet/private-key-like payload 会被 redacted。
13. critical wallet/private-key/funds 类任务不会发送给 LLM，会直接走规则 fallback 并创建 approval。
14. `agent_runs.output` 已记录 provider、model、fallback_used、error_class、usage 等元数据。
15. `audit_logs.metadata` 已记录 runtime mode 和 fallback，不记录 secret 或完整 HTTP payload。
16. 已新增 `GET /api/v1/runtime/status`，返回 runtime provider/model/configured/fallback 状态。
17. 已更新 `.env.example`，支持：

```env
LLM_PROVIDER=deepseek
DEEPSEEK_API_KEY=
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-pro
DEEPSEEK_REASONING_EFFORT=high
DEEPSEEK_THINKING=enabled
LLM_TIMEOUT_SECONDS=180
```

已通过验证：

```sh
make lint
make test
docker compose -f infra/docker-compose.yml config --quiet
```

注意：当前系统仍然没有真实 GitHub、部署、发布、交易、钱包、生产 MCP 集成。DeepSeek 输出只作为 draft text 保存。

---

## 1. 最终目标

把 AgentCompanyOS 打造成完整的 Agent 公司团队操作系统，让 Founder 可以通过 Telegram、REST API 和后续 Admin Web 指挥一个虚拟公司团队，覆盖完整业务闭环：

1. 产品规划：愿景、战略、路线图、PRD、用户故事、验收标准、优先级。
2. 技术设计：架构、数据库、API、风险评审、任务拆分、实施计划。
3. 编程实现：通过本地安装的 Claude Code 执行代码修改、测试、lint、diff 总结。
4. 内容与市场：内容日历、公告、推文、文章、增长实验、KOL 联系草稿、渠道计划。
5. 合规与风控：内容审查、投资/收益声明风险、生产/发布/资金/钱包/交易风险审批。
6. 项目管理：任务分配、状态追踪、依赖关系、阻塞项、日报、周报、里程碑。
7. 知识沉淀：公司 memory、项目上下文、决策记录、artifact、agent run history。
8. 审批执行：所有外部可见、高风险、生产相关动作都必须先生成 approval，审批后仍需通过显式受控执行通道执行。

一句话目标：

> AgentCompanyOS = Founder Command Center + Product Team + Engineering Team + Claude Code Coding Runtime + Marketing Team + Compliance Team + Approval System + Audit Log + Company Memory。

---

## 2. 模型与工具原则

### 2.1 通用模型必须使用 DeepSeek

所有非编程类 Agent 推理、总结、规划、草稿生成默认使用 DeepSeek：

1. 产品规划。
2. PRD / 用户故事 / 验收标准。
3. 架构建议和技术方案草稿。
4. 内容草稿。
5. 市场运营策略。
6. 合规审查。
7. 日报、周报、状态总结。
8. 任务拆解与路由。

要求：

1. 默认 provider：`deepseek`。
2. 默认 model：`deepseek-v4-pro`。
3. 使用现有 `internal/llm` provider 结构扩展，不引入 SDK，继续使用 `net/http`。
4. 保持 `stream: false`。
5. 支持 timeout、safe error class、usage metadata。
6. 所有请求前必须 sanitize。
7. 任何 LLM 失败必须 fallback 到规则运行时或可解释的 degradation path。
8. 任何 secret、private key、token、credential 不得进入 LLM request、日志、audit、error、Telegram response。

### 2.2 编程相关任务必须使用本地 Claude Code

所有真实代码修改、测试、lint、diff 生成、工程文件编辑、依赖变更等编程执行任务，必须通过本地安装的 Claude Code 完成，而不是 DeepSeek。

编程相关任务包括但不限于：

1. 修改代码。
2. 新增文件。
3. 删除文件。
4. 运行测试并根据失败修复。
5. gofmt / lint / build。
6. 生成 migration。
7. 修改 Docker Compose。
8. 修改前端代码。
9. 生成 PR diff summary。
10. 执行 repo 内工程命令。

要求：

1. 新增 Claude Code adapter，明确区分于 DeepSeek LLM provider。
2. Claude Code 只能在本地 workspace 内执行。
3. Claude Code 不得自动执行生产部署、git push、GitHub merge、发布、交易、钱包、资金操作。
4. Claude Code 每次执行必须创建 `agent_runs` 记录，保存 input、sanitized prompt、output summary、command metadata、status、duration、error_class。
5. Claude Code 执行前必须进行风险检测。
6. 高风险编程任务，例如 production deploy、merge、release、security rule、migration to production，只能创建 approval，不得执行。
7. Claude Code 可在低风险开发任务上执行本地文件修改和本地测试，但必须留下 audit log。
8. Claude Code 输出必须作为 artifact 或 run output 保存，避免丢失上下文。
9. 如果本地未安装 Claude Code，系统必须返回 `coding_runtime_unavailable`，并创建可执行手工 prompt artifact。

不要用 DeepSeek 写代码后直接落盘。DeepSeek 可以生成技术计划、验收标准、Claude Code prompt，但代码实施必须由本地 Claude Code adapter 执行。

---

## 3. 必须保留的安全原则

不可破坏当前安全模型：

1. Founder approval first：高风险和外部可见动作必须先审批。
2. Audit everything：所有重要状态变化、工具调用、审批、agent run 必须审计。
3. Safe by default：默认只读、草稿、模拟、审批流。
4. Draft before execute：先生成计划、草稿、记录和 approval，不直接执行外部动作。
5. No hidden side effects：所有工具调用和状态变化必须可解释、可追踪。
6. No production action without explicit approval：没有显式审批不得执行生产动作。
7. Secrets never leave boundary：secret 不得进入 LLM、日志、audit、Telegram、error response。
8. Coding runtime is local only：Claude Code 只能操作本地 workspace，不得默认联网执行外部生产动作。

下一阶段仍然禁止直接接入或自动执行：

1. GitHub push / merge / release。
2. 生产部署。
3. 自动发布 Telegram 群公告、Twitter/X、文章或任何公开内容。
4. 钱包私钥读取、保存、传输。
5. 交易系统、资金系统、钱包系统。
6. 修改真实生产风控规则。
7. 连接真实生产 MCP 或生产工具。

如需支持这些能力，只能先建 approval 和受控接口，不得默认执行。

---

## 4. 下一阶段建议实现范围

请按小步提交式实现，保持每一步可测试。

### 4.1 Agent 团队扩展

扩展内置 Agent，并保证数据库 seed 可幂等：

1. `chief_of_staff`：理解 Founder 意图，路由任务，协调跨团队计划。
2. `product`：PRD、路线图、用户故事、验收标准。
3. `cto`：架构、技术决策、风险评估、工程拆解。
4. `backend`：后端实现计划、API、DB、测试方案。
5. `frontend`：Admin Web、交互、UI 实现计划。
6. `qa`：测试计划、验收测试、回归风险。
7. `devops`：本地 Docker、CI 草稿、部署计划，但不得真实部署。
8. `content`：公告、文章、推文、产品更新草稿。
9. `growth`：增长实验、渠道策略、KOL outreach 草稿。
10. `sales`：客户画像、销售脚本、demo flow、CRM 草稿。
11. `compliance`：内容、财务、收益声明、隐私与风控审查。
12. `finance`：预算、成本、收入预测草稿，不接真实资金系统。
13. `coding`：本地 Claude Code 编程执行 Agent。

每个 Agent 必须有：

1. role。
2. description。
3. permissions。
4. runtime preference：`deepseek` 或 `claude_code_local`。
5. risk boundary。
6. output contract。

### 4.2 Runtime Router

新增 runtime router，按任务类型选择执行方式：

1. 普通规划、草稿、总结：DeepSeek。
2. 编程执行：本地 Claude Code。
3. 高风险动作：approval only。
4. critical sensitive：rule-based fallback + approval。
5. provider unavailable：fallback + audit。

建议接口：

```go
type RuntimeRouter interface {
    Route(ctx context.Context, input AgentRunInput) (RuntimeDecision, error)
}

type RuntimeDecision struct {
    Runtime string
    Provider string
    Model string
    RequiresApproval bool
    ApprovalType string
    Reason string
}
```

### 4.3 Claude Code Local Adapter

新增包建议：

```text
backend/internal/coding/
backend/internal/tools/claudecode/
```

实现能力：

1. 检测本地 `claude` 或配置的 Claude Code command 是否存在。
2. 支持 `.env` 配置：

```env
CODING_RUNTIME=claude_code_local
CLAUDE_CODE_COMMAND=claude
CLAUDE_CODE_TIMEOUT_SECONDS=900
CLAUDE_CODE_WORKDIR=..
CLAUDE_CODE_MAX_OUTPUT_BYTES=200000
```

3. 输入 Claude Code 前生成明确 prompt：任务目标、允许修改范围、禁止动作、验证命令、输出格式。
4. 执行只允许在 workspace 内。
5. 捕获 stdout/stderr、exit code、duration。
6. 超时返回 safe error。
7. 输出摘要保存到 `agent_runs.output`。
8. 大输出保存到 `artifacts` 或文件引用。
9. 不得记录 secret。

注意：如果当前运行环境无法安全调用 Claude Code，先实现 adapter interface、availability check、mock/fake adapter 和 tests。真实命令执行必须有明确配置开关。

### 4.4 Product 到 Engineering 闭环

实现一条完整链路：

```text
Founder idea
-> chief_of_staff route
-> product PRD
-> cto technical plan
-> backend/frontend task breakdown
-> coding agent creates Claude Code implementation plan or executes local coding task
-> qa test plan and validation result
-> content/growth launch draft
-> compliance review
-> Founder approval gates
```

可以先用同步流程实现，不必引入复杂 workflow engine。建议新增 `workflows` service：

```text
backend/internal/workflow/
```

支持 REST 和 Telegram：

1. `/plan [idea]`：生成产品计划和任务拆解。
2. `/build [task]`：创建 coding task；低风险时可调用 Claude Code local；高风险只建 approval。
3. `/launch [topic]`：生成市场/内容/合规草稿，不发布。
4. `/review [item]`：合规和 QA review。

REST 对应：

1. `POST /api/v1/workflows/plan`
2. `POST /api/v1/workflows/build`
3. `POST /api/v1/workflows/launch`
4. `POST /api/v1/workflows/review`
5. `GET /api/v1/workflows/:id`

如需新增表，必须写 migration；如可先复用 `tasks`、`task_events`、`agent_runs`、`artifacts`，优先复用。

### 4.5 Company Memory 与 Artifact

完善 memory 和 artifacts：

1. 保存 PRD、技术方案、Claude Code prompt、测试报告、市场草稿、合规审查。
2. 支持按 project/task/agent 查询。
3. LLM prompt 应引用必要上下文，但必须控制长度和敏感信息。
4. memory 不得保存 secret。
5. artifact 应保存类型、来源、关联 task/run、content hash、metadata。

### 4.6 Admin Web

在 `frontend/admin-web` 中逐步实现可用的管理界面：

1. Dashboard：项目、任务、审批、运行时状态。
2. Task Board：任务列表、状态、owner agent、risk。
3. Approvals：审批列表、approve/reject。
4. Agent Runs：查看 agent output、fallback、provider、usage、error_class。
5. Artifacts：查看 PRD、技术方案、代码执行摘要、市场草稿。
6. Runtime Status：DeepSeek 和 Claude Code availability。

不要做营销 landing page。Admin Web 是工作台，要信息密度适中、可扫描、实用。

---

## 5. 数据模型要求

优先复用现有表：

1. `agents`
2. `projects`
3. `tasks`
4. `task_events`
5. `approvals`
6. `agent_runs`
7. `audit_logs`
8. `telegram_messages`
9. `company_memory`
10. `tool_connections`
11. `artifacts`

如果现有字段不足，可以新增 migration，但不要随意重建 schema。

建议新增或扩展元数据字段：

1. runtime：`deepseek`、`claude_code_local`、`rule_based`。
2. provider。
3. model。
4. fallback_used。
5. error_class。
6. usage。
7. duration_ms。
8. command_metadata。
9. artifact_ids。
10. approval_gate。

所有 JSONB 中不得保存 API key、token、private key、session cookie、完整敏感 payload。

---

## 6. API 与 Telegram 要求

保持当前命令兼容，不破坏已有接口。

新增 Telegram 命令建议：

1. `/plan [idea]`
2. `/build [task]`
3. `/launch [topic]`
4. `/review [item]`
5. `/runs [task_id]`
6. `/artifacts [task_id]`
7. `/runtime`

新增 API 建议：

1. `GET /api/v1/runtime/status`
2. `GET /api/v1/runtime/tools`
3. `POST /api/v1/workflows/plan`
4. `POST /api/v1/workflows/build`
5. `POST /api/v1/workflows/launch`
6. `POST /api/v1/workflows/review`
7. `GET /api/v1/agent-runs`
8. `GET /api/v1/agent-runs/:id`
9. `GET /api/v1/artifacts`
10. `GET /api/v1/artifacts/:id`

API 返回必须是安全错误，不暴露内部 secret 或完整上游 payload。

---

## 7. 风险与审批策略

保留并扩展 risk detector。

必须识别为 high 或 critical 的动作：

1. deploy、production、release、merge、publish。
2. GitHub push、PR merge、tag release。
3. 发布公告、Twitter/X、文章、邮件群发。
4. contact KOL、external outreach。
5. live trading、交易、资金、wallet、private key、seed phrase、私钥、钱包、资金。
6. 修改风控规则、权限规则、安全策略。
7. 删除数据、迁移生产数据库、改变 billing。
8. 连接外部工具、生产 MCP、CRM、广告平台。

行为要求：

1. high risk：创建 approval；默认不执行。
2. critical sensitive：不发送给 DeepSeek；不发送给 Claude Code；创建 approval；返回安全说明。
3. approval approve/reject：仍只更新状态，除非后续明确实现受控 execution queue；当前阶段不要自动执行外部动作。

---

## 8. 测试要求

必须新增或更新测试：

1. DeepSeek 仍然用于通用 Agent runtime。
2. 编程任务被 route 到 Claude Code local adapter。
3. Claude Code 不可用时返回 `coding_runtime_unavailable` 并创建 artifact/prompt。
4. 高风险 coding task 不调用 Claude Code，只创建 approval。
5. critical sensitive task 不调用 DeepSeek、不调用 Claude Code。
6. sanitizer 覆盖 API key、Bearer token、private key block、wallet/private-key-like payload。
7. agent_runs 正确记录 provider/model/runtime/fallback/error/usage/duration。
8. audit_logs 不泄漏 secret。
9. Telegram `/plan`、`/build`、`/launch`、`/review` 基本路径。
10. REST workflow endpoints 基本路径。
11. `/approve` 和 `/reject` 仍不执行外部动作。
12. 现有 Phase 0/Phase 1 测试必须继续通过。

验证命令：

```sh
make lint
make test
docker compose -f infra/docker-compose.yml config --quiet
```

如果新增前端：

```sh
npm test
npm run lint
npm run build
```

实际命令以项目配置为准。

---

## 9. 实现顺序建议

请按以下顺序推进：

1. 重新阅读现有 README、docs、runtime、llm、task、approval、audit、telegram、api、migrations、tests。
2. 建立 runtime router，不改变现有行为。
3. 扩展 Agent registry 和 built-in agents。
4. 新增 Claude Code local adapter interface、config、status，不先做高风险真实执行。
5. 为 coding tasks 增加 route 和 fake adapter tests。
6. 增加 workflow service，实现 plan/build/launch/review 的最小闭环。
7. 将 DeepSeek 用于通用规划和草稿。
8. 将 Claude Code local 用于低风险本地编程任务。
9. 完善 artifacts 和 agent_runs 查询。
10. 增加 Telegram 命令和 REST endpoints。
11. 更新 docs、README、`.env.example`。
12. 跑完整验证。

每一步都要保持系统可运行、可测试、可回退。

---

## 10. 输出要求

实现完成后，请输出：

1. 改动摘要。
2. 新增/修改文件列表。
3. 新增配置项。
4. 新增 API/Telegram 命令。
5. 安全边界说明。
6. 测试与验证结果。
7. 未完成事项和后续建议。

不要输出任何 secret。不要声称已经执行外部生产动作。不要声称已经发布、部署、merge 或交易。


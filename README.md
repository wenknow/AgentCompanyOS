# AgentCompanyOS 项目设计文档 v0.1

## 1. 项目定位

AgentCompanyOS 是一个面向一人公司 / 小团队的公司级 AI Agent 控制平面。

它不是某个单一业务项目的脚本系统，而是一个可复用的公司操作系统底座。Founder 可以通过 Telegram 指挥多个 Agent，完成产品、研发、运营、内容、增长、社区、BD、合规、财务、项目管理等公司流程。

系统核心目标：

1. Founder 通过 Telegram 管理公司级任务。
2. Agent 可以拆解任务、生成方案、生成 Prompt、生成草案、汇报进度。
3. 所有高风险动作必须进入审批系统。
4. 所有动作必须写入审计日志。
5. 支持多项目、多 Agent、多工具、多审批流。
6. 第一版只实现规则版 Agent，不接真实外部执行工具。
7. 后续可接入 LangGraph、Codex、Claude Code、GitHub、Notion、n8n、Langfuse、LiqForge API 等。

一句话定位：

> AgentCompanyOS = Founder Command Center + Agent Registry + Task Manager + Approval System + Tool Gateway + Audit Log。

---

## 2. 设计原则

### 2.1 控制优先，不追求完全自动化

Phase 0 的目标不是让 Agent 自动干活，而是建立可控闭环：

Founder 指令 → 解析命令 → 路由 Agent → 创建任务 / 草案 / 报告 → 需要时创建审批 → Founder 审批 → 写审计日志。

### 2.2 高风险动作只生成 Approval，不直接执行

禁止 Phase 0 实现以下动作：

1. 自动发布内容。
2. 自动部署生产。
3. 自动 merge 代码。
4. 自动联系外部 KOL。
5. 自动操作资金、钱包、交易。
6. 自动修改风控规则。
7. 自动连接敏感外部工具。

### 2.3 工具网关先抽象，后执行

Phase 0 只实现 Tool Gateway 接口和权限模型，不接真实 GitHub、Codex、Claude Code、Notion、交易系统。

### 2.4 Agent Runtime 先规则化，后 LangGraph 化

Phase 0 使用规则版 Agent：

* 固定 Agent 角色。
* 固定指令模板。
* 固定任务拆解逻辑。
* 固定报告生成逻辑。

Phase 1 再接入 LangGraph 作为 Agent Runtime。

### 2.5 审计日志是核心基础设施

所有 Telegram 命令、Agent Run、审批动作、任务状态变化都必须写入 audit_logs。

---

## 3. 系统边界

### Phase 0 实现范围

必须实现：

1. Telegram Bot。
2. Gin API 服务。
3. PostgreSQL 数据库。
4. Redis 连接。
5. Agent Registry。
6. Task Manager。
7. Approval System。
8. Project Manager。
9. Audit Log。
10. Report Generator。
11. 规则版 Agent。
12. Docker Compose。
13. 数据库 migrations。
14. README 和 docs。
15. 基础单元测试。

### Phase 0 不实现

不要实现：

1. 真实 LLM 调用。
2. 真实 LangGraph 工作流。
3. 真实 GitHub 操作。
4. 真实 Codex / Claude Code 操作。
5. 真实部署系统。
6. 自动发布 Telegram 群公告。
7. 自动发 Twitter / X。
8. 自动 merge 代码。
9. 钱包私钥保存。
10. 交易系统接入。
11. 复杂前端后台。

---

## 4. 总体架构

```text
Founder
  ↓
Telegram Bot
  ↓
Command Parser
  ↓
Command Handler
  ↓
Agent Router
  ↓
Chief of Staff Agent
  ↓
Core Services
  ├── Agent Registry
  ├── Task Manager
  ├── Approval System
  ├── Project Manager
  ├── Report Generator
  ├── Memory Service
  ├── Tool Gateway
  └── Audit Service
  ↓
PostgreSQL / Redis
```

---

## 5. 模块设计

### 5.1 Telegram Bot

职责：

1. 接收 Founder 命令。
2. 解析 Telegram 文本。
3. 调用 Command Handler。
4. 返回任务、审批、状态、报告等结果。
5. 将所有命令写入 telegram_messages。
6. 将所有命令动作写入 audit_logs。

Phase 0 建议使用 long polling。后续生产环境可切换 webhook。

核心命令：

```text
/start
/help
/status
/agents
/projects
/tasks
/assign [agent] [task]
/approvals
/approve [approval_id]
/reject [approval_id] [reason]
/daily
/weekly
```

---

### 5.2 Command Parser

职责：

1. 解析 Telegram 命令。
2. 提取 command、args、raw_text、chat_id、user_id。
3. 校验命令格式。
4. 输出结构化 Command 对象。

Command 对象示例：

```go
type Command struct {
    Name    string
    Args    []string
    RawText string
    ChatID  int64
    UserID  int64
}
```

---

### 5.3 Agent Registry

职责：

1. 管理 Agent 元数据。
2. 内置 Phase 0 六个 Agent。
3. 提供 Agent 查询。
4. 判断 Agent 是否存在。
5. 判断 Agent 是否有权限执行某类动作。

Phase 0 内置 Agent：

1. Chief of Staff Agent
2. Product Agent
3. CTO Agent
4. Backend Agent
5. Content Agent
6. Compliance Agent

Agent 权限建议：

```text
chief_of_staff:
  - create_task
  - assign_task
  - generate_report
  - request_approval

product:
  - create_task
  - generate_prd
  - generate_acceptance_criteria

cto:
  - create_task
  - generate_technical_plan
  - generate_risk_review

backend:
  - create_task
  - generate_api_design
  - generate_db_design
  - generate_code_prompt

content:
  - generate_content_draft
  - request_publish_approval

compliance:
  - review_content
  - review_risk
  - request_revision
```

---

### 5.4 Task Manager

职责：

1. 创建任务。
2. 分配任务给 Agent。
3. 查询任务列表。
4. 修改任务状态。
5. 记录任务事件。
6. 支持项目维度筛选。

任务状态机：

```text
created
→ assigned
→ planning
→ executing
→ waiting_review
→ revision_required
→ approved
→ completed
→ archived
```

特殊状态：

```text
blocked
cancelled
failed
needs_founder_approval
needs_compliance_review
needs_security_review
```

Phase 0 简化规则：

1. `/assign [agent] [task]` 直接创建 assigned 状态任务。
2. 规则版 Agent 同步生成一次 agent_run。
3. 如果任务文本包含高风险关键词，则创建 approval。
4. 普通任务不自动完成，只进入 assigned 或 waiting_review。

高风险关键词示例：

```text
发布
公告
上线
部署
生产
merge
合并
交易
钱包
私钥
资金
KOL
风控
risk rule
live trading
production
deploy
```

---

### 5.5 Approval System

职责：

1. 创建审批事项。
2. 查询审批列表。
3. Founder approve / reject。
4. 保存审批原因。
5. 写入 audit log。

审批类型：

```text
publish_content
merge_code
deploy_staging
deploy_production
change_risk_rule
contact_kol
send_telegram_announcement
enable_live_trading
access_sensitive_data
connect_external_tool
modify_project_roadmap
```

审批状态：

```text
pending
approved
rejected
revised
hold
cancelled
```

Phase 0 规则：

1. 高风险动作只创建 approval。
2. `/approve` 只更新 approval 状态，不执行真实外部动作。
3. `/reject` 只更新 approval 状态和 reason。
4. 所有审批动作写入 audit_logs。

---

### 5.6 Tool Gateway

职责：

1. 统一外部工具抽象。
2. 管理工具连接状态。
3. 管理工具权限。
4. 后续代理 GitHub、Codex、Claude Code、Notion、Telegram Channel、n8n、Langfuse、LiqForge API。

Phase 0 只实现接口和 mock。

接口建议：

```go
type ToolGateway interface {
    ListTools(ctx context.Context) ([]ToolConnection, error)
    GetTool(ctx context.Context, name string) (*ToolConnection, error)
    Execute(ctx context.Context, req ToolExecutionRequest) (*ToolExecutionResult, error)
}
```

Phase 0 的 Execute 永远返回：

```text
not implemented in phase 0
```

---

### 5.7 LLM Provider

职责：

1. 抽象 LLM 调用。
2. 后续支持 OpenAI、Claude、本地模型。
3. Phase 0 不进行真实调用，只实现 NoopProvider。

接口建议：

```go
type LLMProvider interface {
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}
```

Phase 0 NoopProvider 返回固定文本。

---

### 5.8 Agent Runtime Adapter

职责：

1. 屏蔽规则版 Agent 和 LangGraph Agent 的差异。
2. Phase 0 使用 RuleBasedRuntime。
3. Phase 1 接 LangGraphRuntime。

接口建议：

```go
type AgentRuntime interface {
    Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error)
}
```

Phase 0 的 RuleBasedRuntime 根据 agent role 和任务内容生成固定输出。

---

### 5.9 Report Generator

职责：

1. 生成日报。
2. 生成周报。
3. 汇总任务状态。
4. 汇总审批状态。
5. 汇总阻塞项。
6. 汇总 Agent Run。

日报内容：

```text
今日任务总数
新增任务
进行中任务
等待审批
已完成任务
阻塞任务
Agent 动作摘要
风险提醒
下一步建议
```

周报内容：

```text
本周任务概览
本周完成事项
本周审批事项
本周阻塞
项目进展
Agent 工作统计
下周建议
```

---

### 5.10 Audit Log

职责：

1. 记录所有 Founder 命令。
2. 记录所有 Agent Run。
3. 记录所有审批动作。
4. 记录所有任务状态变化。
5. 记录风险等级。

风险等级：

```text
low
medium
high
critical
```

审计日志必须 append-only。Phase 0 不提供删除功能。

---

## 6. 数据库设计

推荐统一使用 UUID 主键。

### 6.1 agents

```sql
CREATE TABLE agents (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.2 projects

```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    current_phase TEXT NOT NULL DEFAULT 'phase_0',
    owner TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.3 tasks

```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    title TEXT NOT NULL,
    description TEXT,
    owner_agent TEXT,
    priority TEXT NOT NULL DEFAULT 'P2',
    status TEXT NOT NULL DEFAULT 'assigned',
    due_date DATE,
    created_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.4 task_events

```sql
CREATE TABLE task_events (
    id UUID PRIMARY KEY,
    task_id UUID REFERENCES tasks(id),
    event_type TEXT NOT NULL,
    actor TEXT NOT NULL,
    message TEXT,
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.5 approvals

```sql
CREATE TABLE approvals (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    approval_type TEXT NOT NULL,
    item_type TEXT,
    item_id UUID,
    requested_by TEXT,
    approval_status TEXT NOT NULL DEFAULT 'pending',
    approved_by TEXT,
    reason TEXT,
    risk_level TEXT NOT NULL DEFAULT 'medium',
    payload JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.6 agent_runs

```sql
CREATE TABLE agent_runs (
    id UUID PRIMARY KEY,
    agent_id UUID REFERENCES agents(id),
    project_id UUID REFERENCES projects(id),
    task_id UUID REFERENCES tasks(id),
    input JSONB NOT NULL DEFAULT '{}',
    output JSONB NOT NULL DEFAULT '{}',
    tools_used JSONB NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'completed',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ
);
```

### 6.7 audit_logs

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    actor TEXT NOT NULL,
    action TEXT NOT NULL,
    target TEXT,
    risk_level TEXT NOT NULL DEFAULT 'low',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.8 telegram_messages

```sql
CREATE TABLE telegram_messages (
    id UUID PRIMARY KEY,
    chat_id BIGINT NOT NULL,
    user_id BIGINT,
    command TEXT,
    raw_text TEXT NOT NULL,
    parsed_intent JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.9 company_memory

```sql
CREATE TABLE company_memory (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    tags JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.10 tool_connections

```sql
CREATE TABLE tool_connections (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    tool_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'disabled',
    permissions JSONB NOT NULL DEFAULT '[]',
    config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### 6.11 artifacts

建议新增 artifacts 表，用于保存 Agent 生成的草案、开发 Prompt、报告、PRD、技术方案。

```sql
CREATE TABLE artifacts (
    id UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id),
    task_id UUID REFERENCES tasks(id),
    agent_id UUID REFERENCES agents(id),
    artifact_type TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    metadata JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

原因：

1. 任务是工作流对象。
2. agent_runs 是执行记录。
3. artifacts 才是可复用产物。
4. 后续 PRD、技术方案、Prompt、公告草案、周报都应该进入 artifacts。

---

## 7. 推荐目录结构

```text
agent-company-os/
├── backend/
│   ├── cmd/
│   │   ├── api/
│   │   │   └── main.go
│   │   ├── bot/
│   │   │   └── main.go
│   │   └── worker/
│   │       └── main.go
│   ├── internal/
│   │   ├── app/
│   │   ├── config/
│   │   ├── logger/
│   │   ├── database/
│   │   ├── redis/
│   │   ├── api/
│   │   ├── telegram/
│   │   ├── command/
│   │   ├── agents/
│   │   │   ├── registry/
│   │   │   ├── runtime/
│   │   │   └── builtin/
│   │   ├── task/
│   │   ├── approval/
│   │   ├── project/
│   │   ├── toolgateway/
│   │   ├── memory/
│   │   ├── report/
│   │   ├── llm/
│   │   ├── artifact/
│   │   └── audit/
│   ├── migrations/
│   ├── tests/
│   ├── go.mod
│   └── go.sum
├── frontend/
│   └── admin-web/
├── infra/
│   └── docker-compose.yml
├── docs/
│   ├── architecture.md
│   ├── agent-roles.md
│   ├── telegram-commands.md
│   ├── permission-model.md
│   ├── approval-flow.md
│   ├── database-schema.md
│   ├── roadmap.md
│   └── phase-0-acceptance.md
├── .env.example
├── Makefile
└── README.md
```

---

## 8. API 设计

Phase 0 REST API 主要用于调试和后续 Admin Web 预留。

### Health

```text
GET /health
```

返回：

```json
{
  "status": "ok",
  "service": "agent-company-os"
}
```

### Agents

```text
GET /api/v1/agents
```

### Projects

```text
GET /api/v1/projects
POST /api/v1/projects
```

### Tasks

```text
GET /api/v1/tasks
POST /api/v1/tasks
GET /api/v1/tasks/:id
PATCH /api/v1/tasks/:id/status
```

### Approvals

```text
GET /api/v1/approvals
POST /api/v1/approvals
POST /api/v1/approvals/:id/approve
POST /api/v1/approvals/:id/reject
```

### Reports

```text
GET /api/v1/reports/daily
GET /api/v1/reports/weekly
```

---

## 9. Telegram 命令行为

### /start

返回：

```text
Welcome to AgentCompanyOS.
You are now connected to Founder Command Bot.
Use /help to see available commands.
```

### /help

返回命令说明。

### /status

返回：

```text
AgentCompanyOS Status

Projects: N
Tasks: N
Pending Approvals: N
Active Agents: N
Blocked Tasks: N
```

### /agents

返回 6 个内置 Agent。

### /projects

如果没有项目，自动创建默认项目：

```text
AgentCompanyOS
```

并返回项目列表。

### /assign [agent] [task]

示例：

```text
/assign backend 设计任务系统数据库 schema
```

行为：

1. 校验 agent 是否存在。
2. 获取默认项目。
3. 创建 task。
4. 创建 task_event。
5. 执行规则版 agent_run。
6. 如果识别为高风险任务，创建 approval。
7. 写入 audit_logs。
8. 返回任务 ID、Agent 输出和是否需要审批。

### /tasks

返回最近 20 个任务。

### /approvals

返回 pending approvals。

### /approve [approval_id]

行为：

1. 更新 approval_status = approved。
2. 写入 audit_logs。
3. 返回确认信息。
4. 不执行真实外部动作。

### /reject [approval_id] [reason]

行为：

1. 更新 approval_status = rejected。
2. 保存 reason。
3. 写入 audit_logs。
4. 返回确认信息。

### /daily

生成日报。

### /weekly

生成周报。

---

## 10. 风险识别策略

Phase 0 使用简单关键词策略。

高风险动作识别：

```go
func DetectRisk(text string) RiskResult {
    // publish / deploy / merge / trading / wallet / funds / risk rule / KOL 等关键词
}
```

风险等级：

```text
low: 普通任务、文档、计划
medium: 涉及外部内容草案、产品路线图
high: 发布、部署、merge、联系外部人
critical: 资金、钱包、交易、风控规则、生产部署
```

处理规则：

1. low：正常创建任务。
2. medium：正常创建任务，可提示注意。
3. high：创建 task + approval。
4. critical：创建 task + approval，并标记 needs_founder_approval。

---

## 11. Phase 0 验收标准

必须满足：

1. `docker compose up` 可以启动 backend、PostgreSQL、Redis。
2. API `/health` 返回 ok。
3. Telegram Bot 可以响应 `/start`。
4. Telegram Bot 可以响应 `/help`。
5. `/assign` 可以创建任务。
6. `/tasks` 可以查看任务。
7. `/status` 可以查看整体状态。
8. `/agents` 可以查看 Agent 列表。
9. `/projects` 可以查看项目列表。
10. `/approvals` 可以查看审批事项。
11. `/approve` 可以批准审批。
12. `/reject` 可以拒绝审批。
13. `/daily` 可以生成日报。
14. `/weekly` 可以生成周报。
15. 所有 Telegram 命令写入 `telegram_messages`。
16. 所有关键动作写入 `audit_logs`。
17. 所有 Agent 动作写入 `agent_runs`。
18. 高风险动作只创建 approval，不直接执行。
19. README 说明如何配置 Telegram Bot Token。
20. 有基础单元测试。
21. 所有外部依赖通过 `.env` 或 `config.yaml` 管理。

---

## 12. Phase 1 方向

Phase 1 接入 LangGraph：

1. Chief of Staff 工作流。
2. 产品 PRD 工作流。
3. 内容生成 + 合规审查工作流。
4. 代码任务拆解工作流。
5. 项目周报工作流。

Phase 1 的核心变化：

```text
RuleBasedRuntime → LangGraphRuntime
```

但 Task Manager、Approval System、Audit Log、Tool Gateway 不变。

---

## 13. Phase 2 方向

Phase 2 接入真实工具：

1. GitHub：读 Issue、创建 PR、读 diff。
2. Codex：生成代码、修复 bug、运行测试。
3. Claude Code：复杂代码任务。
4. Notion：写文档。
5. Telegram Channel：待审批后发布公告。
6. n8n：外部自动化。
7. Langfuse：LLM 观测。
8. LiqForge API：业务数据查询。

所有工具必须走 Tool Gateway + Approval System。

---

## 14. 核心优化结论

相比原始方案，优化点如下：

1. 增加 artifacts 表，保存 Agent 产物。
2. 将审批系统设计为一等公民，而不是任务附属功能。
3. 将风险识别抽成 Risk Policy，避免散落在命令逻辑中。
4. Phase 0 先使用 long polling，降低 Telegram 本地开发复杂度。
5. Agent Runtime 明确抽象，Phase 0 规则版，Phase 1 LangGraph。
6. Tool Gateway Phase 0 只 mock，不接真实工具。
7. 把 AgentCompanyOS 的核心定义为控制平面，而不是执行平面。
8. 所有高风险动作只进入 approval，不执行。
9. 所有动作必须 audit log。
10. 保留多项目能力，但 Phase 0 默认自动创建一个默认项目，降低使用复杂度。


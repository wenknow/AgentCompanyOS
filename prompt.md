# 给 Codex 的开发 Prompt：AgentCompanyOS Phase 0 MVP

你是资深 Golang 后端工程师、系统架构师、Telegram Bot 开发者和创业公司 CTO。请使用 AgentCompany Codex Kit，按公司规范直接实现 AgentCompanyOS Phase 0 MVP。

本 prompt 是执行规格，不是讨论稿。请创建代码、目录、migrations、Docker Compose、README、docs 和测试，使项目可运行、可测试、可通过 Docker Compose 启动。

---

## 0. 必须遵守的 AgentCompany Codex Kit 规范

开发前必须按 AgentCompany Codex Kit 的规则执行，尤其遵守：

1. `agent-company-os-dev`：AgentCompanyOS Phase 0 MVP 开发规则。
2. `company-go-backend`：Go、Gin、PostgreSQL、Redis、Docker Compose、REST API 后端规范。
3. `company-risk-review`：高风险动作识别与审批规范。
4. `company-test-and-lint`：测试、lint、验证流程。
5. 公司政策、工程规范、Go 规范、安全规范、审批策略、风控规范。

不可违背的公司原则：

1. Founder approval first：高风险和外部可见动作必须先审批。
2. Audit everything：所有改变产品状态、用户状态、数据、策略或外部系统意图的动作必须有审计。
3. Safe by default：优先只读、草稿、模拟、审批流。
4. Draft before execute：先生成计划、草稿、记录和 approval，不直接执行外部动作。
5. No hidden side effects：所有工具调用和状态变化必须可解释、可追踪。
6. No production action without explicit approval：没有显式审批不得执行生产动作。

Phase 0 绝对禁止：

1. 接入真实 LLM。
2. 接入真实 GitHub。
3. 接入真实 Codex 或 Claude Code 执行。
4. 执行真实部署、生产发布、代码 merge。
5. 自动发布 Telegram 群公告、Twitter/X、文章或任何公开内容。
6. 保存或处理钱包私钥。
7. 接入交易系统、资金系统、钱包系统。
8. 修改真实风控规则。
9. 连接真实生产工具或生产 MCP。

---

## 1. 项目目标

项目名：`AgentCompanyOS`

AgentCompanyOS 是一个面向一人公司 / 小团队的公司级 Agent 控制平面。Founder 可以通过 Telegram 指挥多个 Agent，进行任务创建、任务分配、审批、状态查看、日报、周报等操作。

Phase 0 只实现框架 MVP：

1. 不接入真实 LLM。
2. 不接入真实 GitHub。
3. 不接入真实 Codex。
4. 不接入 Claude Code。
5. 不接生产部署。
6. 不接交易系统。
7. Agent 行为使用规则版逻辑。
8. 预留 LLM Provider、Tool Gateway、Agent Runtime Adapter 接口。

核心原则：

1. 所有高风险动作只创建 approval，不直接执行。
2. 所有 Telegram 命令必须写入 `telegram_messages`。
3. 所有关键动作必须写入 `audit_logs`。
4. 所有 Agent 动作必须写入 `agent_runs`。
5. 所有任务状态变化必须写入 `task_events` 和 `audit_logs`。
6. 代码必须可运行、可测试、可 Docker Compose 启动。

一句话定位：

> AgentCompanyOS = Founder Command Center + Agent Registry + Task Manager + Approval System + Tool Gateway + Audit Log。

---

## 2. 技术栈与工程规范

使用：

1. Backend：Golang
2. Web Framework：Gin
3. Telegram：Telegram Bot API，Phase 0 使用 long polling
4. Database：PostgreSQL
5. Cache / Queue：Redis
6. Logging：zap
7. Migration：golang-migrate 或等价方案
8. Config：`.env` + `config.yaml`，至少支持 `.env`
9. Deployment：Docker Compose
10. API：REST API
11. Test：Go testing

Go 工程要求：

1. `cmd` 只负责进程启动和依赖装配。
2. transport / handler 放在 `internal/api` 或对应 handler package。
3. 业务逻辑放在 service 层。
4. 持久化放在 repository 层。
5. domain、service、repository、transport 边界清晰。
6. 所有请求级工作使用 `context.Context` 作为第一个参数。
7. 数据库、Redis、外部调用必须透传 context。
8. 错误使用 `%w` 包装，API 返回安全错误，详细错误写结构化日志。
9. PostgreSQL 操作必须使用参数化 SQL、ORM 或安全 query builder，禁止拼接 SQL 注入风险。
10. zap 日志必须结构化，并包含 request、task、project、approval 等相关 ID。
11. 日志、audit、Telegram 返回内容不得输出 token、API key、session cookie、私钥、凭证或敏感 payload。
12. schema 变更必须通过 migrations。
13. operational 命令必须能通过 Makefile、README 或脚本复现。

---

## 3. 目标目录结构

请创建以下目录结构：

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

## 4. 配置与安全

创建 `.env.example`：

```env
APP_ENV=development
APP_NAME=agent-company-os
HTTP_PORT=8080

DATABASE_URL=postgres://agent:agent@postgres:5432/agent_company_os?sslmode=disable
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

TELEGRAM_BOT_TOKEN=
TELEGRAM_ALLOWED_USER_IDS=

LOG_LEVEL=debug
DEFAULT_PROJECT_NAME=AgentCompanyOS
```

要求：

1. 所有配置从环境变量读取。
2. 至少支持 `.env`。
3. `TELEGRAM_ALLOWED_USER_IDS` 为空时允许所有用户。
4. `TELEGRAM_ALLOWED_USER_IDS` 非空时只允许白名单用户。
5. 不得硬编码 token、secret、password、private key。
6. 不得在日志、audit、error response、Telegram response 中输出敏感值。
7. admin、生产、外部工具、敏感数据访问都必须作为审批动作处理；Phase 0 只创建 approval。

---

## 5. 数据库 migrations

创建 PostgreSQL migrations，包含以下表：

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

所有表使用 UUID primary key。`jsonb` 字段默认值必须合法，时间字段使用 `timestamptz`。

### agents

1. `id UUID primary key`
2. `name text unique not null`
3. `role text not null`
4. `description text`
5. `permissions jsonb default []`
6. `status text default active`
7. `created_at timestamptz default now()`
8. `updated_at timestamptz default now()`

### projects

1. `id UUID primary key`
2. `name text unique not null`
3. `description text`
4. `status text default active`
5. `current_phase text default phase_0`
6. `owner text`
7. `created_at timestamptz default now()`
8. `updated_at timestamptz default now()`

### tasks

1. `id UUID primary key`
2. `project_id UUID references projects(id)`
3. `title text not null`
4. `description text`
5. `owner_agent text`
6. `priority text default P2`
7. `status text default assigned`
8. `due_date date`
9. `created_by text`
10. `created_at timestamptz default now()`
11. `updated_at timestamptz default now()`

### task_events

1. `id UUID primary key`
2. `task_id UUID references tasks(id)`
3. `event_type text not null`
4. `actor text not null`
5. `message text`
6. `metadata jsonb default {}`
7. `created_at timestamptz default now()`

### approvals

1. `id UUID primary key`
2. `project_id UUID references projects(id)`
3. `approval_type text not null`
4. `item_type text`
5. `item_id UUID`
6. `requested_by text`
7. `approval_status text default pending`
8. `approved_by text`
9. `reason text`
10. `risk_level text default medium`
11. `payload jsonb default {}`
12. `created_at timestamptz default now()`
13. `updated_at timestamptz default now()`

Approval payload 必须支持这些字段：

1. `action_type`
2. `requester`
3. `project`
4. `environment`
5. `risk_level`
6. `summary`
7. `evidence`
8. `expected_impact`
9. `rollback_or_mitigation_plan`
10. `required_approver`
11. `review_deadline`

Approval payload 不得包含 secrets、private keys、tokens 或完整敏感 payload。

### agent_runs

1. `id UUID primary key`
2. `agent_id UUID references agents(id)`
3. `project_id UUID references projects(id)`
4. `task_id UUID references tasks(id)`
5. `input jsonb default {}`
6. `output jsonb default {}`
7. `tools_used jsonb default []`
8. `status text default completed`
9. `error_message text`
10. `created_at timestamptz default now()`
11. `completed_at timestamptz`

### audit_logs

1. `id UUID primary key`
2. `project_id UUID references projects(id)`
3. `actor text not null`
4. `action text not null`
5. `target text`
6. `risk_level text default low`
7. `metadata jsonb default {}`
8. `created_at timestamptz default now()`

### telegram_messages

1. `id UUID primary key`
2. `chat_id bigint not null`
3. `user_id bigint`
4. `command text`
5. `raw_text text not null`
6. `parsed_intent jsonb default {}`
7. `created_at timestamptz default now()`

### company_memory

1. `id UUID primary key`
2. `project_id UUID references projects(id)`
3. `type text not null`
4. `title text not null`
5. `content text not null`
6. `tags jsonb default []`
7. `created_at timestamptz default now()`
8. `updated_at timestamptz default now()`

### tool_connections

1. `id UUID primary key`
2. `name text unique not null`
3. `tool_type text not null`
4. `status text default disabled`
5. `permissions jsonb default []`
6. `config jsonb default {}`
7. `created_at timestamptz default now()`
8. `updated_at timestamptz default now()`

### artifacts

1. `id UUID primary key`
2. `project_id UUID references projects(id)`
3. `task_id UUID references tasks(id)`
4. `agent_id UUID references agents(id)`
5. `artifact_type text not null`
6. `title text not null`
7. `content text not null`
8. `status text default draft`
9. `metadata jsonb default {}`
10. `created_at timestamptz default now()`
11. `updated_at timestamptz default now()`

---

## 6. 内置 Agent 与 Agent Runtime

启动时 seed 6 个 Agent：

1. `chief_of_staff`
2. `product`
3. `cto`
4. `backend`
5. `content`
6. `compliance`

每个 Agent 需要 `name`、`role`、`description`、`permissions`。

职责：

1. `chief_of_staff`：理解 Founder 指令、拆解任务、分配任务、汇总状态、生成日报/周报、识别阻塞、提醒审批。
2. `product`：写 PRD、拆用户故事、生成验收标准、管理产品路线图、整理用户反馈。
3. `cto`：技术方案设计、架构拆解、技术任务拆分、代码 Review 建议、技术风险识别。
4. `backend`：生成后端开发任务、生成 Codex / Claude Code Prompt 草稿、设计 API、设计数据库、输出测试要求。
5. `content`：生成中文内容草案、推文草稿、公告草稿、周报、产品更新日志；不得自动发布。
6. `compliance`：审查内容风险，避免投资建议、收益承诺、夸大宣传，生成合规修改建议。

实现接口：

```go
type AgentRuntime interface {
    Run(ctx context.Context, input AgentRunInput) (*AgentRunOutput, error)
}
```

Phase 0 实现 `RuleBasedRuntime`：

1. 根据 Agent role 返回固定但有用的文本。
2. 不调用真实 LLM。
3. 不调用真实工具。
4. 每次 Agent Run 必须写入 `agent_runs`。
5. 每次 Agent Run 的关键动作必须写入 `audit_logs`。

示例输出要求：

1. backend agent 输出开发拆解、API 设计、数据库设计、测试要求和 Codex Prompt 草稿。
2. content agent 输出内容草稿，并明确“不会发布，需要 Founder approval”。
3. compliance agent 输出合规审查建议，避免保证收益、直接投资建议、夸大宣传，并建议添加风险提示。

---

## 7. 风险识别与审批

实现 Risk Detector。

风险等级：

1. `low`：本地只读或有限影响动作。
2. `medium`：内部行为变化、非生产集成、可回滚 schema 变化。
3. `high`：部署、merge、公开内容、敏感数据访问、外部集成、路线图变化、风控策略变化。
4. `critical`：生产破坏性动作、资金、钱包、私钥、交易、真实风控规则、不可逆数据损失、大范围公开公告。

高风险关键词包括：

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
publish
announcement
```

规则：

1. 普通任务：创建 task + task_event + agent_run + audit_log。
2. 高风险任务：创建 task + task_event + agent_run + approval + audit_log。
3. critical 任务：`task.status = needs_founder_approval`，只创建 approval，不执行真实外部动作。
4. 所有高风险任务返回 Telegram 提醒：该动作不会直接执行，已创建审批。
5. `/approve` 和 `/reject` 只更新 approval 状态，不执行真实外部动作。

审批类型映射：

1. 发布 / publish → `publish_content`
2. 公告 / announcement → `send_telegram_announcement`
3. deploy / 部署 / 生产 → `deploy_production`
4. merge / 合并 → `merge_code`
5. 交易 / live trading → `enable_live_trading`
6. 钱包 / 私钥 / 资金 → `access_sensitive_data`
7. KOL → `contact_kol`
8. 风控 / risk rule → `change_risk_rule`
9. 外部工具连接 → `connect_external_tool`
10. 路线图重大变化 → `modify_project_roadmap`

---

## 8. Telegram 命令

实现 Telegram Bot，Phase 0 使用 long polling。

所有 Telegram 消息必须先写入 `telegram_messages`，再执行业务处理。所有命令动作必须写入 `audit_logs`。

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

实现命令：

### `/start`

返回欢迎信息。

### `/help`

返回所有可用命令。

### `/status`

返回：

1. projects count
2. tasks count
3. pending approvals count
4. active agents count
5. blocked tasks count

### `/agents`

返回内置 Agent 列表。

### `/projects`

返回项目列表。若没有项目，自动创建默认项目。

### `/assign [agent] [task]`

示例：

```text
/assign backend 设计任务系统数据库 schema
```

行为：

1. 解析 agent。
2. 解析 task。
3. 校验 agent 是否存在。
4. 获取或创建默认项目。
5. 创建 task。
6. 创建 task_event。
7. 调用 RuleBasedRuntime。
8. 写入 agent_runs。
9. 风险检测。
10. 高风险则创建 approval。
11. 写入 audit_logs。
12. 返回结果。

### `/tasks`

返回最近 20 个任务。

### `/approvals`

返回 pending approvals。

### `/approve [approval_id]`

更新 approval 为 approved。Phase 0 不执行真实外部动作。

### `/reject [approval_id] [reason]`

更新 approval 为 rejected，并保存 reason。Phase 0 不执行真实外部动作。

### `/daily`

生成日报。

### `/weekly`

生成周报。

---

## 9. REST API

实现：

```text
GET /health

GET /api/v1/agents
GET /api/v1/projects
POST /api/v1/projects

GET /api/v1/tasks
POST /api/v1/tasks
GET /api/v1/tasks/:id
PATCH /api/v1/tasks/:id/status

GET /api/v1/approvals
POST /api/v1/approvals
POST /api/v1/approvals/:id/approve
POST /api/v1/approvals/:id/reject

GET /api/v1/reports/daily
GET /api/v1/reports/weekly
```

要求：

1. API 可以简单实现，但必须可用。
2. 所有写操作必须走 service 层。
3. 所有关键写操作必须写 `audit_logs`。
4. API 错误不得泄露 secrets 或内部敏感 payload。
5. `/health` 返回 ok，并能检查 API 进程基础可用性。

---

## 10. Docker Compose、Makefile、README、docs

### Docker Compose

创建 `infra/docker-compose.yml`，包含：

1. backend
2. postgres
3. redis

要求：

1. `docker compose -f infra/docker-compose.yml up` 可以启动。
2. backend 依赖 postgres 和 redis。
3. postgres 数据持久化 volume。
4. backend 读取环境变量。
5. 暴露 8080 端口。
6. 不包含真实生产凭证。

### Makefile

创建 Makefile，至少包含：

```makefile
run
test
lint
docker-up
docker-down
migrate-up
migrate-down
```

命令可以调用 backend 目录下 go 命令。`make test` 必须可运行。

### README

README 必须包含：

1. 项目介绍。
2. 架构说明。
3. Phase 0 功能。
4. 如何创建 Telegram Bot Token。
5. 如何配置 `.env`。
6. 如何启动 API。
7. 如何启动 Bot。
8. 如何使用 Docker Compose。
9. Telegram 命令示例。
10. 高风险审批机制说明。
11. Phase 0 不做什么。
12. 后续路线图。

### docs

创建以下文档：

1. `docs/architecture.md`
2. `docs/agent-roles.md`
3. `docs/telegram-commands.md`
4. `docs/permission-model.md`
5. `docs/approval-flow.md`
6. `docs/database-schema.md`
7. `docs/roadmap.md`
8. `docs/phase-0-acceptance.md`

文档不需要过度冗长，但必须清楚、可执行、与代码一致。

---

## 11. 测试要求

至少添加以下测试：

1. Command Parser 测试。
2. Risk Detector 测试。
3. Agent Registry 测试。
4. Approval Service 测试。
5. Task Service 测试。

测试要求：

1. `go test ./...` 可以运行通过。
2. `make test` 可以运行通过。
3. 高风险关键词必须覆盖中英文。
4. `/assign backend xxx` 必须覆盖普通任务。
5. `/assign content 发布公告 xxx` 必须覆盖高风险任务并创建 approval。
6. `/approve` 和 `/reject` 必须验证只更新审批状态，不执行真实外部动作。
7. 配置加载测试不得依赖真实 secrets。

---

## 12. 完成标准

完成后必须满足：

1. 项目可以 `go test ./...`。
2. 项目可以 `make test`。
3. 项目可以 `docker compose -f infra/docker-compose.yml up`。
4. API `/health` 返回 ok。
5. Telegram Bot 可以响应 `/start`。
6. Telegram Bot 可以响应 `/help`。
7. Telegram Bot 可以响应 `/agents`。
8. Telegram Bot 可以响应 `/projects`。
9. Telegram Bot 可以通过 `/assign backend xxx` 创建任务。
10. Telegram Bot 可以通过 `/tasks` 查看任务。
11. Telegram Bot 可以通过 `/approvals` 查看审批。
12. 高风险任务会创建 approval。
13. `/approve` 可以批准审批。
14. `/reject` 可以拒绝审批。
15. `/daily` 可以生成日报。
16. `/weekly` 可以生成周报。
17. 所有 Telegram 命令有 `telegram_messages` 记录。
18. 所有 Agent 动作有 `agent_runs` 记录。
19. 所有关键动作有 `audit_logs` 记录。
20. README 和 docs 完整。
21. 没有真实外部副作用。
22. 没有硬编码 secrets。

最终回复必须报告：

1. 已创建/修改的主要模块。
2. 已运行的测试和结果。
3. 未运行的验证及原因。
4. 任何剩余风险或 Phase 0 明确不支持的能力。

---

## 13. 最终执行指令

请直接开始实现，不要只输出方案。

实现时必须保持 AgentCompany Codex Kit 的公司规范优先级高于局部便利性：安全默认、审批优先、审计完整、无真实外部副作用、可运行、可测试、可复现。

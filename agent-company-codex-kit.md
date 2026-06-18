# Codex 开发任务：创建 AgentCompanyOS Company Codex Plugin Kit

你是资深 Codex Plugin 开发者、Golang 后端工程师、AI Agent 系统架构师、安全工程师和创业公司 CTO。

请创建一个可复用的 Codex Plugin 包，项目名：

`agent-company-codex-kit`

目标：

把公司规范、代码规范、风控规范、项目工作流、MCP 配置、AgentCompanyOS 开发工作流打包为一个可安装、可复用、可扩展的 Codex Plugin / Skills Kit。

## 一、核心要求

这个包不是业务代码，而是公司级 Codex 工作规范包。

它需要包含：

1. Codex plugin manifest。
2. 多个 Codex skills。
3. 公司工程规范文档。
4. 风控和审批规范文档。
5. Go/Gin/PostgreSQL/Redis 后端开发规范。
6. AgentCompanyOS 项目开发 skill。
7. MCP 配置模板。
8. AGENTS.md 模板。
9. README。
10. 本地安装说明。
11. 使用示例。

## 二、创建目录结构

请创建以下目录：

```text
agent-company-codex-kit/
├── .codex-plugin/
│   └── plugin.json
├── skills/
│   ├── company-go-backend/
│   │   └── SKILL.md
│   ├── company-architecture-review/
│   │   └── SKILL.md
│   ├── company-risk-review/
│   │   └── SKILL.md
│   ├── company-prd-to-tasks/
│   │   └── SKILL.md
│   ├── company-code-review/
│   │   └── SKILL.md
│   ├── company-test-and-lint/
│   │   └── SKILL.md
│   └── agent-company-os-dev/
│       └── SKILL.md
├── references/
│   ├── company-policy.md
│   ├── engineering-standard.md
│   ├── golang-standard.md
│   ├── security-standard.md
│   ├── risk-control-standard.md
│   ├── approval-policy.md
│   ├── project-workflow.md
│   ├── mcp-policy.md
│   └── agent-company-os-architecture.md
├── templates/
│   ├── AGENTS.md
│   ├── AGENTS.override.md
│   └── README.template.md
├── hooks/
│   ├── hooks.json
│   ├── session_start.py
│   └── pre_tool_use.py
├── scripts/
│   ├── init_project.py
│   ├── validate_policy.py
│   └── generate_agents_md.py
├── .mcp.json
├── marketplace.json
└── README.md
```

## 三、plugin.json

创建 `.codex-plugin/plugin.json`：

```json
{
  "name": "agent-company-codex-kit",
  "version": "0.1.0",
  "description": "Company-level Codex plugin kit for engineering standards, risk controls, project workflows, MCP configuration, and AgentCompanyOS development.",
  "author": {
    "name": "AgentCompanyOS",
    "email": "team@example.com"
  },
  "license": "MIT",
  "keywords": [
    "codex",
    "agent",
    "company-os",
    "golang",
    "risk-control",
    "mcp",
    "workflow"
  ],
  "skills": "./skills/",
  "mcpServers": "./.mcp.json",
  "hooks": "./hooks/hooks.json",
  "interface": {
    "displayName": "AgentCompany Codex Kit",
    "shortDescription": "Company standards, risk rules, workflows, and MCP config for Codex.",
    "longDescription": "A reusable Codex plugin kit that packages company engineering standards, Go backend conventions, risk-control policies, approval workflows, AgentCompanyOS development rules, and safe MCP server configuration.",
    "developerName": "AgentCompanyOS",
    "category": "Productivity",
    "capabilities": ["Read", "Write"],
    "defaultPrompt": [
      "Use AgentCompany Codex Kit to implement a Go backend feature following company standards.",
      "Use AgentCompany Codex Kit to review this change for risk, security, and approval requirements.",
      "Use AgentCompany Codex Kit to develop AgentCompanyOS Phase 0 MVP."
    ],
    "brandColor": "#10A37F"
  }
}
```

## 四、Skills 内容要求

每个 `SKILL.md` 必须有 YAML frontmatter：

```markdown
---
name: skill-name
description: Clear trigger description.
---
```

### 1. company-go-backend/SKILL.md

用途：当任务涉及 Golang、Gin、PostgreSQL、Redis、Docker Compose、REST API、backend service 时触发。

必须要求 Codex：

1. 使用清晰分层：cmd/internal/config/logger/database/api/service/repository。
2. 不把业务逻辑写进 main.go。
3. 使用 context.Context。
4. 使用 zap logging。
5. 所有配置从环境变量读取。
6. 不硬编码密钥。
7. PostgreSQL 使用 migration。
8. Redis 只作为 cache / queue，不作为核心事实数据源。
9. API 要有错误处理。
10. 变更后运行 gofmt、go test。
11. 重要变更更新 README 或 docs。

### 2. company-architecture-review/SKILL.md

用途：当任务涉及系统设计、模块拆分、架构评审、重构、服务边界时触发。

必须输出：

1. 当前架构理解。
2. 影响范围。
3. 推荐设计。
4. 模块边界。
5. 数据流。
6. 风险点。
7. 不做什么。
8. 分阶段实现计划。

### 3. company-risk-review/SKILL.md

用途：当任务涉及发布、部署、merge、生产、资金、钱包、交易、风控规则、KOL、外部公告、用户数据、敏感数据时触发。

必须遵守：

1. 不自动发布公开内容。
2. 不自动部署生产。
3. 不自动 merge 代码。
4. 不自动操作资金。
5. 不保存钱包私钥。
6. 不直接修改交易或风控规则。
7. 不自动联系外部 KOL。
8. 高风险动作只生成 approval request。
9. 给出风险等级：low / medium / high / critical。
10. 给出审批人和审批理由。

### 4. company-prd-to-tasks/SKILL.md

用途：当输入是产品想法、PRD、需求、功能规划时触发。

必须输出：

1. 产品目标。
2. 用户角色。
3. 核心用户故事。
4. MVP 范围。
5. 非 MVP 范围。
6. API / 数据库 / 前端 / 测试任务拆分。
7. 验收标准。
8. 风险和依赖。
9. 开发阶段计划。

### 5. company-code-review/SKILL.md

用途：当任务涉及 code review、diff review、PR review、bug fix review 时触发。

必须检查：

1. 代码正确性。
2. 架构边界。
3. 错误处理。
4. 测试覆盖。
5. 安全问题。
6. 数据库 migration 风险。
7. 配置和密钥风险。
8. 并发和事务风险。
9. 可维护性。
10. 是否需要审批。

### 6. company-test-and-lint/SKILL.md

用途：当任务涉及测试、lint、CI、验收时触发。

必须要求：

1. 运行 gofmt。
2. 运行 go test ./...
3. 如项目有 Makefile，优先使用 make test。
4. 如果测试失败，先定位失败原因。
5. 不要隐藏失败。
6. 输出测试结果摘要。
7. 给出未覆盖风险。

### 7. agent-company-os-dev/SKILL.md

用途：开发 AgentCompanyOS 项目时触发。

必须遵守项目规则：

1. AgentCompanyOS 是公司级 Agent 控制平面，不是单项目脚本。
2. Founder 通过 Telegram 指挥。
3. 第一版只实现 Phase 0 MVP。
4. Phase 0 不接真实 GitHub、Codex、Claude Code、部署系统、交易系统。
5. 高风险动作只创建 approval，不执行。
6. 所有 Telegram 命令写入 telegram_messages。
7. 所有 Agent 动作写入 agent_runs。
8. 所有关键动作写入 audit_logs。
9. 使用 Golang、Gin、PostgreSQL、Redis、Docker Compose、zap。
10. 预留 Agent Runtime、Tool Gateway、LLM Provider 接口。
11. 内置 Agent：chief_of_staff、product、cto、backend、content、compliance。
12. 必须实现任务系统、审批系统、Agent Registry、项目系统、日报周报。
13. 必须有 migrations、README、docs、Makefile、基础测试。

## 五、references 文档内容要求

创建以下文档，并写入清晰规则。

### company-policy.md

写公司级原则：

1. Founder approval first。
2. Audit everything。
3. Safe by default。
4. Draft before execute。
5. No hidden side effects。
6. No production action without explicit approval。

### engineering-standard.md

写工程规范：

1. 简洁架构。
2. 明确模块边界。
3. 可测试。
4. 可观测。
5. 可配置。
6. 可回滚。
7. 文档同步更新。

### golang-standard.md

写 Go 规范：

1. package 命名。
2. context 使用。
3. error wrapping。
4. repository/service/handler 分层。
5. transaction 处理。
6. logging。
7. testing。
8. migrations。

### security-standard.md

写安全规范：

1. 不硬编码 secret。
2. 不打印 token。
3. 不保存私钥。
4. 最小权限。
5. 输入校验。
6. SQL 注入防护。
7. 敏感操作审批。

### risk-control-standard.md

写风险等级：

low、medium、high、critical。

写高风险关键词：

发布、公告、上线、部署、生产、merge、合并、交易、钱包、私钥、资金、KOL、风控、risk rule、live trading、production、deploy、publish、announcement。

### approval-policy.md

写审批规则：

1. publish_content
2. merge_code
3. deploy_staging
4. deploy_production
5. change_risk_rule
6. contact_kol
7. send_telegram_announcement
8. enable_live_trading
9. access_sensitive_data
10. connect_external_tool
11. modify_project_roadmap

### project-workflow.md

写项目工作流：

1. Idea
2. PRD
3. Architecture
4. Task breakdown
5. Implementation
6. Test
7. Review
8. Approval
9. Release
10. Retrospective

### mcp-policy.md

写 MCP 使用规则：

1. 默认只读。
2. 写工具必须审批。
3. 生产工具默认禁用。
4. 钱包、资金、交易工具默认不接。
5. MCP server 必须声明权限。
6. MCP 工具输出不可直接作为事实，关键动作需要二次验证。

### agent-company-os-architecture.md

写 AgentCompanyOS Phase 0 架构摘要：

Founder → Telegram Bot → Command Parser → Agent Router → Task Manager / Approval System / Agent Registry / Audit Log → PostgreSQL / Redis。

## 六、templates/AGENTS.md

创建项目级 AGENTS.md 模板，包含：

```markdown
# AGENTS.md

## Project Identity

This repository follows AgentCompanyOS company engineering standards.

## Non-negotiable Rules

- Do not hardcode secrets.
- Do not bypass approval.
- Do not deploy production.
- Do not merge code automatically.
- Do not publish public content automatically.
- Do not operate funds, wallets, trading, or risk rules.
- All high-risk actions must create an approval request.
- All significant code changes must include tests or explain why tests are not possible.

## Development Workflow

1. Understand the task.
2. Read relevant docs.
3. Propose a short implementation plan.
4. Make minimal focused changes.
5. Run formatters and tests.
6. Summarize changes, tests, and risks.

## Go Backend Standards

- Use context.Context.
- Keep handler/service/repository boundaries clear.
- Use migrations for schema changes.
- Use zap for structured logging.
- Read configuration from environment variables.
- Never commit secrets.

## Risk Policy

High-risk operations include deploy, publish, merge, trading, wallet, private key, production, KOL contact, and risk rule changes.

For high-risk operations, create an approval request instead of executing the action.
```

## 七、.mcp.json

创建安全默认配置：

```json
{
  "mcp_servers": {
    "filesystem-readonly": {
      "command": "mcp-server-filesystem",
      "args": ["--readonly", "."]
    },
    "docs": {
      "command": "docs-mcp",
      "args": ["--stdio"]
    }
  }
}
```

不要配置任何真实资金、钱包、交易、生产部署 MCP server。

## 八、hooks

创建 `hooks/hooks.json`：

```json
{
  "hooks": {
    "SessionStart": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "python3 ${PLUGIN_ROOT}/hooks/session_start.py",
            "statusMessage": "Loading AgentCompany company standards"
          }
        ]
      }
    ],
    "PreToolUse": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "python3 ${PLUGIN_ROOT}/hooks/pre_tool_use.py",
            "statusMessage": "Checking company risk policy"
          }
        ]
      }
    ]
  }
}
```

创建 `session_start.py`：

功能：

1. 输出公司规范摘要。
2. 提醒高风险动作必须审批。
3. 不读取任何 secret。
4. 不联网。

创建 `pre_tool_use.py`：

功能：

1. 从 stdin 读取 hook payload。
2. 检测危险关键词。
3. 如果发现危险命令，输出警告。
4. 不要破坏正常开发流程。
5. 对明显危险操作建议人工确认。

危险关键词：

```text
rm -rf
drop database
truncate
kubectl delete
terraform destroy
docker system prune
git push --force
npm publish
deploy prod
production deploy
private key
wallet
transfer
withdraw
```

## 九、scripts

### init_project.py

功能：

1. 给目标项目生成 AGENTS.md。
2. 可选复制 README template。
3. 不覆盖已有 AGENTS.md，除非传入 `--force`。

### validate_policy.py

功能：

1. 检查 references 是否存在。
2. 检查 skills 是否都有 SKILL.md。
3. 检查 plugin.json 是否存在。
4. 检查 .mcp.json 是否存在。
5. 输出检查结果。

### generate_agents_md.py

功能：

根据 templates/AGENTS.md 输出项目可用 AGENTS.md。

## 十、marketplace.json

创建本地 marketplace 示例：

```json
{
  "name": "agent-company-local-marketplace",
  "interface": {
    "displayName": "AgentCompany Local Plugins"
  },
  "plugins": [
    {
      "name": "agent-company-codex-kit",
      "source": {
        "source": "local",
        "path": "./agent-company-codex-kit"
      },
      "policy": {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL"
      },
      "category": "Productivity"
    }
  ]
}
```

## 十一、README

README 必须包含：

1. 项目介绍。
2. 为什么要做这个 Codex Plugin。
3. 包含哪些 skills。
4. 包含哪些 references。
5. 如何本地安装。
6. 如何复制到 `~/.codex/plugins`。
7. 如何配置 marketplace。
8. 如何在项目里生成 AGENTS.md。
9. 如何启用 MCP。
10. 风控规则说明。
11. 高风险动作说明。
12. AgentCompanyOS 开发示例。
13. 后续路线图。

## 十二、验收标准

完成后必须满足：

1. 目录结构完整。
2. `.codex-plugin/plugin.json` 存在且 JSON 合法。
3. 每个 skill 都有 SKILL.md。
4. 每个 SKILL.md 都有 name 和 description。
5. references 文档完整。
6. `.mcp.json` 合法。
7. hooks 文件存在。
8. scripts 可运行。
9. README 清楚说明安装和使用。
10. 不包含任何真实 secret。
11. 不包含任何真实钱包、交易、生产部署配置。
12. 这个包可以作为公司内部 Codex Plugin Kit 复用。

请直接创建所有文件和内容，不要只输出方案。


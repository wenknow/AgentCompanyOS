# AgentCompanyOS

AgentCompanyOS 是一个面向一人公司和小团队的 AI Agent 控制平面。它通过 Telegram 和 REST API 管理项目、任务、Agent、审批、审计、工作流、自动开发、部署和反馈闭环。

当前版本已经从早期 Phase 0 设计稿推进到可运行的后端闭环：DeepSeek 负责规划和总结，Claude Code local adapter 负责低风险本地编码任务，Telegram bot 负责项目管理、自动调度、进度汇报、审批和反馈采集。

## 当前能力

- Telegram Bot：Founder 通过 Telegram 管理项目、任务、审批、部署和 autopilot。
- Agent 团队：`chief_of_staff`、`product`、`cto`、`backend`、`frontend`、`qa`、`devops`、`content`、`growth`、`sales`、`finance`、`compliance`、`coding`。
- Runtime Router：普通规划、内容、合规走 DeepSeek；低风险 coding 走 Claude Code local；高风险 coding 进入审批或 fallback。
- Workflow：支持 `plan`、`build`、`launch`、`review`，产出 task、agent run、artifact、audit log。
- Project Config：每个项目可配置 workdir、项目文档、自动提交、自动部署和多个部署服务。
- Autopilot：根据 `project.md`、Telegram 反馈和关键错误日志连续推进开发；遇到 Claude limit 会等待 reset 后继续。
- 部署闭环：支持多服务部署命令；只有检测到主项目代码变化时才 commit / deploy / 创建部署审批。
- Approval：高风险动作仍通过审批系统；支持数字编号和 `/reject all` 批量拒绝。
- 可见性：支持查询 tasks、agent runs、artifacts、runtime tools、approvals。
- 日志反馈：PM2 日志只保留接口 `4xx/5xx`、`error`、`failed`、`panic`、`timeout` 等关键问题，过滤普通 debug/warning 噪音。
- 中文汇报：Telegram 汇报使用中文短摘要，重点反馈规划、编码、代码变化、部署和阻塞原因。

## 目录结构

```text
agent-company-os/
  backend/                 Go 后端服务
    cmd/api                REST API
    cmd/bot                Telegram Bot
    cmd/worker             Worker 入口
    internal/app           应用服务编排、Telegram 命令、autopilot
    internal/agents        Agent registry 和 runtime router
    internal/workflow      plan/build/launch/review workflow
    internal/deployment    本地部署 adapter
    internal/approval      审批系统
    internal/task          task、event、agent run
    internal/artifact      workflow 和配置产物
    internal/llm           DeepSeek provider 和 sanitizer
  docs/                    架构、权限、审批、命令文档
  infra/docker-compose.yml PostgreSQL / Redis / API 组合配置
  Makefile                 本地运行、测试、迁移命令
```

## 快速启动

```sh
cd agent-company-os
cp .env.example .env
make docker-up
curl http://localhost:8080/health
```

本地运行 API：

```sh
cd agent-company-os
make run
```

本地运行 Telegram bot：

```sh
cd agent-company-os
make run-bot
```

PM2 运行示例：

```sh
cd /home/wen/code/AgentCompanyOS/agent-company-os
pm2 start "make run-bot" --name agent-bot
```

## 关键环境变量

基础配置：

```env
DATABASE_URL=postgres://agent:agent@localhost:5432/agent_company_os?sslmode=disable
REDIS_ADDR=localhost:6379
TELEGRAM_BOT_TOKEN=
TELEGRAM_ALLOWED_USER_IDS=
DEFAULT_PROJECT_NAME=AgentCompanyOS
```

DeepSeek：

```env
LLM_PROVIDER=deepseek
DEEPSEEK_API_KEY=
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-pro
DEEPSEEK_REASONING_EFFORT=high
DEEPSEEK_THINKING=enabled
LLM_TIMEOUT_SECONDS=180
```

Claude Code local：

```env
CODING_RUNTIME=claude_code_local
CLAUDE_CODE_ENABLED=false
CLAUDE_CODE_COMMAND=claude
CLAUDE_CODE_TIMEOUT_SECONDS=900
CLAUDE_CODE_WORKDIR=..
CLAUDE_CODE_ALLOWED_ROOT=
CLAUDE_CODE_MAX_OUTPUT_BYTES=200000
```

部署：

```env
DEPLOY_ALLOWED_ROOT=
DEPLOY_TIMEOUT_SECONDS=600
DEPLOY_MAX_OUTPUT_BYTES=200000
```

说明：`CLAUDE_CODE_ENABLED=false` 是默认安全值。只有明确开启后，低风险 coding task 才会调用本机 Claude Code。

## Telegram 命令

常用命令：

```text
/start
/help
/status
/agents
/projects
/project list
/project show [项目名或编号]
/project config [项目名] workdir=/path doc=project.md auto_commit=true auto_deploy=true service=api:pm2 restart api
/tasks
/task get [编号]
/runs [任务编号]
/artifacts [任务编号]
/approvals
/approve [编号]
/reject [编号] [原因]
/reject all [原因]
/runtime
```

Workflow 命令：

```text
/plan [idea]
/build [task]
/launch [topic]
/review [item]
```

Autopilot：

```text
/autopilot start [项目]
/autopilot status [项目]
/autopilot stop [项目]
/autopilot run [项目]
```

反馈采集：

```text
/feedback [项目] [反馈内容]
```

部署：

```text
/deploy [项目] [原因]
```

## 项目配置示例

以 LiqForge 为例：

```text
/project config LiqForge workdir=/home/wen/code/liqForge doc=project.md auto_commit=true auto_deploy=true service=liqforge-api:pm2 restart liqforge-api service=liqforge-collector:pm2 restart liqforge-collector service=liqforge-frontend:pm2 restart liqforge-frontend service=liqforge-wallet:pm2 restart liqforge-wallet
```

字段说明：

- `workdir`：项目主目录，Claude 和部署命令都在此目录内执行。
- `doc`：项目文档路径，通常是 `project.md`。
- `auto_commit=true`：本轮开始前工作区干净且 Claude 产生主目录代码变化时，自动 `git add -A && git commit`。
- `auto_deploy=true`：检测到代码变化后，按配置服务执行部署。
- `service=name:command args...`：声明一个可部署服务，可重复配置多个服务。

安全规则：

- 如果本轮开始前已有未提交改动，bot 不会自动 commit，避免混入人工修改。
- 如果主项目代码没有变化，bot 不会部署。
- 如果 Claude 写入 `.claude/worktrees` 而不是主目录，bot 会跳过部署。
- 如果 Claude 返回 limit/reset，autopilot 会等待 reset 后继续，不会反复发指令浪费额度。

## REST API

主要接口：

```text
GET  /health
GET  /api/v1/runtime/status
GET  /api/v1/runtime/tools
GET  /api/v1/agents
GET  /api/v1/projects
POST /api/v1/projects
GET  /api/v1/tasks
POST /api/v1/tasks
GET  /api/v1/tasks/:id
PATCH /api/v1/tasks/:id/status
GET  /api/v1/approvals
POST /api/v1/approvals/:id/approve
POST /api/v1/approvals/:id/reject
POST /api/v1/workflows/plan
POST /api/v1/workflows/build
POST /api/v1/workflows/launch
POST /api/v1/workflows/review
GET  /api/v1/agent-runs
GET  /api/v1/artifacts
GET  /api/v1/artifacts/:id
GET  /api/v1/reports/daily
GET  /api/v1/reports/weekly
```

## 安全模型

AgentCompanyOS 默认保守：

- critical sensitive 内容不进入 DeepSeek / Claude。
- 高风险 coding task 不直接执行 Claude，优先进入审批。
- 默认不启用 Claude Code，必须显式配置 `CLAUDE_CODE_ENABLED=true`。
- Claude workdir 必须位于 allowed root 内。
- 部署命令只能在配置项目目录和 allowed root 内执行。
- 所有 task、workflow、approval、deployment、agent run 都写入审计或可查询记录。

## Autopilot 工作方式

当前 autopilot 的目标是持续推进项目，而不是固定轮询：

1. 读取项目配置和 `project.md`。
2. 收集 Telegram 用户反馈。
3. 收集 PM2 关键错误日志，只保留接口错误和运行异常。
4. DeepSeek 生成中文短规划。
5. Claude Code 按规划直接修改项目主目录。
6. 检查 git 主工作区变化。
7. 可选自动 commit。
8. 可选部署多服务。
9. 遇到 Claude limit/reset 自动等待。

Telegram 只汇报短中文摘要，例如：

```text
规划：补齐前端缺失接口并修复 404。
编码：新增 positions API 客户端和页面状态处理。
代码变化：M frontend/app/positions/page.tsx
部署：liqforge-api, liqforge-frontend
```

## 开发验证

```sh
cd agent-company-os
make test
make lint
docker compose -f infra/docker-compose.yml config --quiet
```

## 当前限制

- Admin Web 尚未实现。
- GitHub、Notion、n8n、社媒发布、交易、钱包私钥、生产 MCP 集成尚未接入。
- Autopilot 状态目前保存在进程内，PM2 重启后需要重新 `/autopilot start [项目]`。
- Claude Code 额度限制由本机 Claude CLI 返回信息判断；如果返回格式变化，等待时间可能退回默认值。
- 自动部署适合本地 PM2/脚本场景，生产级回滚、灰度、健康检查仍需继续完善。

## 最近状态

当前后端闭环已经具备：

- Telegram 项目管理和自动调度。
- DeepSeek 中文规划和短摘要。
- Claude Code 本地编码 adapter。
- 多服务部署配置和 approval 后执行。
- 代码变化检测、自动 commit、自动部署。
- PM2 关键错误日志采集。
- Agent run、artifact、approval、task 可见性。


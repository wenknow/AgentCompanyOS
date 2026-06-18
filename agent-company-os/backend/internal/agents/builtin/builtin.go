package builtin

import "github.com/agentcompany/agent-company-os/backend/internal/model"

func Agents() []model.Agent {
	return []model.Agent{
		{Name: "chief_of_staff", Role: "Chief of Staff Agent", Description: "Understands Founder intent, routes tasks, summarizes status, and requests approvals.", Permissions: []string{"create_task", "assign_task", "generate_report", "request_approval"}, Status: "active"},
		{Name: "product", Role: "Product Agent", Description: "Creates PRDs, user stories, acceptance criteria, and product roadmap drafts.", Permissions: []string{"create_task", "generate_prd", "generate_acceptance_criteria"}, Status: "active"},
		{Name: "cto", Role: "CTO Agent", Description: "Designs technical plans, architecture breakdowns, review advice, and risk reviews.", Permissions: []string{"create_task", "generate_technical_plan", "generate_risk_review"}, Status: "active"},
		{Name: "backend", Role: "Backend Agent", Description: "Drafts backend implementation tasks, API design, database design, tests, and Codex prompts.", Permissions: []string{"create_task", "generate_api_design", "generate_db_design", "generate_code_prompt"}, Status: "active"},
		{Name: "designer", Role: "Senior Product Designer Agent", Description: "Creates refined product UI direction, interaction patterns, visual hierarchy, spacing, typography, and design QA guidance with senior Apple-inspired taste while avoiding generic AI-looking interfaces.", Permissions: []string{"create_task", "generate_design_direction", "generate_ui_critique", "generate_design_system", "request_revision"}, Status: "active"},
		{Name: "frontend", Role: "Frontend Agent", Description: "Drafts UI flows, component plans, accessibility checks, and frontend implementation prompts.", Permissions: []string{"create_task", "generate_ui_plan", "generate_component_plan", "generate_code_prompt"}, Status: "active"},
		{Name: "qa", Role: "QA Agent", Description: "Creates test plans, acceptance checks, regression review drafts, and release quality notes.", Permissions: []string{"create_task", "generate_test_plan", "review_quality", "request_revision"}, Status: "active"},
		{Name: "devops", Role: "DevOps Agent", Description: "Drafts infrastructure, deployment, rollback, observability, and operational runbook plans without executing them.", Permissions: []string{"create_task", "generate_runbook", "generate_deployment_plan", "request_approval"}, Status: "active"},
		{Name: "content", Role: "Content Agent", Description: "Drafts Chinese content, announcements, tweets, product updates, and weekly notes without publishing.", Permissions: []string{"generate_content_draft", "request_publish_approval"}, Status: "active"},
		{Name: "growth", Role: "Growth Agent", Description: "Drafts launch strategy, distribution experiments, campaign briefs, and metrics plans without contacting external parties.", Permissions: []string{"generate_growth_plan", "generate_campaign_draft", "request_publish_approval"}, Status: "active"},
		{Name: "sales", Role: "Sales Agent", Description: "Drafts sales motions, lead research briefs, outreach copy, and CRM notes without sending messages.", Permissions: []string{"generate_sales_brief", "generate_outreach_draft", "request_approval"}, Status: "active"},
		{Name: "finance", Role: "Finance Agent", Description: "Drafts budget, runway, pricing, and financial review notes without moving funds or giving investment advice.", Permissions: []string{"generate_budget_draft", "review_financial_risk", "request_approval"}, Status: "active"},
		{Name: "compliance", Role: "Compliance Agent", Description: "Reviews content and product risk, avoiding investment advice, promises, and exaggerated claims.", Permissions: []string{"review_content", "review_risk", "request_revision"}, Status: "active"},
		{Name: "coding", Role: "Coding Agent", Description: "Prepares local coding task prompts and may use Claude Code only when explicitly enabled and low risk.", Permissions: []string{"generate_code_prompt", "request_code_approval", "record_artifact"}, Status: "active"},
	}
}

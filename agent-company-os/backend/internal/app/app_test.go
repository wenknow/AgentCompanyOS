package app

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseProjectConfigArgsServices(t *testing.T) {
	cfg, err := parseProjectConfigArgs("LiqForge", []string{
		"workdir=/home/wen/code/liqForge",
		"doc=project.md",
		"auto_deploy=true",
		"auto_commit=true",
		"service=liqforge-api:pm2", "restart", "liqforge-api",
		"service=liqforge-collector:pm2", "restart", "liqforge-collector",
		"service=liqforge-frontend:pm2", "restart", "liqforge-frontend",
		"service=liqforge-wallet:pm2", "restart", "liqforge-wallet",
	}, "/home/wen/code")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoDeploy || !cfg.AutoCommit || cfg.Workdir != "/home/wen/code/liqForge" || len(cfg.Services) != 4 {
		t.Fatalf("unexpected config %#v", cfg)
	}
	if cfg.Services[0].Name != "liqforge-api" || !reflect.DeepEqual(cfg.Services[0].DeployCommand, []string{"pm2", "restart", "liqforge-api"}) {
		t.Fatalf("unexpected first service %#v", cfg.Services[0])
	}
	if cfg.Services[3].Name != "liqforge-wallet" || !reflect.DeepEqual(cfg.Services[3].DeployCommand, []string{"pm2", "restart", "liqforge-wallet"}) {
		t.Fatalf("unexpected wallet service %#v", cfg.Services[3])
	}
}

func TestParseProjectConfigArgsDeployDoesNotConsumeNextFields(t *testing.T) {
	cfg, err := parseProjectConfigArgs("LiqForge", []string{
		"workdir=/tmp/app",
		"deploy=pm2", "restart", "liqforge-api",
		"auto_deploy=true",
	}, "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AutoDeploy {
		t.Fatalf("expected auto deploy")
	}
	want := []string{"pm2", "restart", "liqforge-api"}
	if !reflect.DeepEqual(cfg.DeployCommand, want) {
		t.Fatalf("deploy command = %#v, want %#v", cfg.DeployCommand, want)
	}
}

func TestSafeAutopilotWorkflowTextRemovesRiskWords(t *testing.T) {
	text := safeAutopilotWorkflowText("liqforge-wallet deploy production private key")
	for _, forbidden := range []string{"wallet", "deploy", "production", "private key"} {
		if strings.Contains(strings.ToLower(text), forbidden) {
			t.Fatalf("expected %q to be removed from %q", forbidden, text)
		}
	}
}

func TestProjectGitSnapshotSeparatesClaudeWorktrees(t *testing.T) {
	s := projectGitSnapshot{Valid: true, MainStatus: " M backend/main.go", ClaudeWorktree: "?? .claude/worktrees/agent/file.go"}
	before := projectGitSnapshot{Valid: true, MainStatus: " M backend/main.go", ClaudeWorktree: ""}
	if s.MainChangedFrom(before) {
		t.Fatalf("claude worktree changes should not count as main checkout changes")
	}
	if !s.ClaudeWorktreeChangedFrom(before) {
		t.Fatalf("expected claude worktree changes to be detected")
	}
}

func TestExtractDiagnosticLogLines(t *testing.T) {
	text := extractDiagnosticLogLines(`info booted
[GIN-debug] [WARNING] Running in debug mode
[GIN] 2026 | 404 | GET /api/missing
Error failed to connect
`)
	if strings.Contains(text, "GIN-debug") || strings.Contains(text, "info booted") || !strings.Contains(text, "404") || !strings.Contains(text, "Error failed") {
		t.Fatalf("unexpected diagnostics: %q", text)
	}
}

func TestGitStatusSummary(t *testing.T) {
	text := gitStatusSummary(" M a.go\n?? b.go")
	if !strings.Contains(text, "M a.go") || !strings.Contains(text, "?? b.go") {
		t.Fatalf("unexpected summary: %q", text)
	}
}

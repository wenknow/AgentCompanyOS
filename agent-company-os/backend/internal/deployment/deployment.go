package deployment

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/agentcompany/agent-company-os/backend/internal/llm"
)

type ProjectConfig struct {
	ProjectName   string   `json:"project_name"`
	Workdir       string   `json:"workdir"`
	DocPath       string   `json:"doc_path"`
	DeployCommand []string `json:"deploy_command"`
	Services      []Target `json:"services,omitempty"`
	AllowedRoot   string   `json:"allowed_root"`
	AutoDeploy    bool     `json:"auto_deploy"`
	AutoCommit    bool     `json:"auto_commit"`
}

type Target struct {
	Name          string   `json:"name"`
	DeployCommand []string `json:"deploy_command"`
}

type Result struct {
	Status     string `json:"status"`
	Output     string `json:"output"`
	ErrorClass string `json:"error_class,omitempty"`
}

type Service struct {
	allowedRoot    string
	timeout        time.Duration
	maxOutputBytes int
}

func NewService(allowedRoot string, timeout time.Duration, maxOutputBytes int) *Service {
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	if maxOutputBytes <= 0 {
		maxOutputBytes = 100000
	}
	return &Service{allowedRoot: allowedRoot, timeout: timeout, maxOutputBytes: maxOutputBytes}
}

func (s *Service) Execute(ctx context.Context, cfg ProjectConfig) (*Result, error) {
	targets := cfg.DeploymentTargets()
	if len(targets) == 0 {
		return &Result{Status: "failed", ErrorClass: "missing_deploy_command"}, nil
	}
	workdir, err := filepath.Abs(cfg.Workdir)
	if err != nil {
		return &Result{Status: "failed", ErrorClass: "invalid_workdir"}, nil
	}
	root, err := resolveAllowedRoot(firstNonEmpty(cfg.AllowedRoot, s.allowedRoot))
	if err != nil || !pathAllowed(workdir, root) {
		return &Result{Status: "failed", ErrorClass: "invalid_workdir"}, nil
	}
	var outputs []string
	for _, target := range targets {
		result, err := s.executeTarget(ctx, workdir, root, target)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(result.Output) != "" {
			outputs = append(outputs, strings.TrimSpace(result.Output))
		}
		if result.Status != "completed" {
			result.Output = strings.Join(outputs, "\n")
			if target.Name != "" && result.ErrorClass != "" {
				result.ErrorClass = target.Name + ":" + result.ErrorClass
			}
			return result, nil
		}
	}
	return &Result{Status: "completed", Output: llm.SanitizeText(strings.Join(outputs, "\n"))}, nil
}

func (s *Service) executeTarget(ctx context.Context, workdir, root string, target Target) (*Result, error) {
	if len(target.DeployCommand) == 0 {
		return &Result{Status: "failed", ErrorClass: "missing_deploy_command"}, nil
	}
	if strings.ContainsAny(target.DeployCommand[0], `/\`) {
		cmdPath := target.DeployCommand[0]
		if !filepath.IsAbs(cmdPath) {
			cmdPath = filepath.Join(workdir, cmdPath)
		}
		absCmd, err := filepath.Abs(cmdPath)
		if err != nil || !pathAllowed(absCmd, root) {
			return &Result{Status: "failed", ErrorClass: "invalid_command"}, nil
		}
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, target.DeployCommand[0], target.DeployCommand[1:]...)
	cmd.Dir = workdir
	var stdout, stderr limitedBuffer
	stdout.limit = s.maxOutputBytes
	stderr.limit = s.maxOutputBytes / 4
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return &Result{Status: "failed", ErrorClass: "timeout", Output: llm.SanitizeText(stdout.String())}, nil
	}
	output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if err != nil {
		return &Result{Status: "failed", ErrorClass: "execution_failed", Output: llm.SanitizeText(output)}, nil
	}
	if target.Name != "" && output != "" {
		output = target.Name + ":\n" + output
	}
	return &Result{Status: "completed", Output: llm.SanitizeText(output)}, nil
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
	return filepath.Abs(cwd)
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

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (cfg ProjectConfig) Validate() error {
	if strings.TrimSpace(cfg.ProjectName) == "" {
		return fmt.Errorf("project name is required")
	}
	if strings.TrimSpace(cfg.Workdir) == "" {
		return fmt.Errorf("workdir is required")
	}
	if len(cfg.DeploymentTargets()) == 0 {
		return fmt.Errorf("deploy command is required")
	}
	return nil
}

func (cfg ProjectConfig) DeploymentTargets() []Target {
	if len(cfg.Services) > 0 {
		return cfg.Services
	}
	if len(cfg.DeployCommand) > 0 {
		return []Target{{DeployCommand: cfg.DeployCommand}}
	}
	return nil
}

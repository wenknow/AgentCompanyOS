package report

import (
	"context"
	"fmt"

	"github.com/agentcompany/agent-company-os/backend/internal/approval"
	"github.com/agentcompany/agent-company-os/backend/internal/task"
)

type Service struct {
	tasks     task.Repository
	approvals approval.Repository
}

func NewService(tasks task.Repository, approvals approval.Repository) *Service {
	return &Service{tasks: tasks, approvals: approvals}
}

func (s *Service) Daily(ctx context.Context) (string, error) {
	tasks, err := s.tasks.List(ctx, 20)
	if err != nil {
		return "", err
	}
	pending, _ := s.approvals.CountPending(ctx)
	blocked, _ := s.tasks.BlockedCount(ctx)
	return fmt.Sprintf("Daily report\nTasks reviewed: %d\nWaiting approvals: %d\nBlocked tasks: %d\nAgent actions: rule-based Phase 0 runs only\nRisk reminder: high-risk work remains draft-only until Founder approval.", len(tasks), pending, blocked), nil
}

func (s *Service) Weekly(ctx context.Context) (string, error) {
	total, _ := s.tasks.Count(ctx)
	pending, _ := s.approvals.CountPending(ctx)
	blocked, _ := s.tasks.BlockedCount(ctx)
	return fmt.Sprintf("Weekly report\nTotal tasks: %d\nPending approvals: %d\nBlocked or approval-gated tasks: %d\nProject progress: Phase 0 simulated control plane\nNext week: continue implementation through audited tasks and approvals.", total, pending, blocked), nil
}

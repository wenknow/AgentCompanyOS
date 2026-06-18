package toolgateway

import (
	"context"
	"errors"
)

type ToolConnection struct {
	Name        string   `json:"name"`
	ToolType    string   `json:"tool_type"`
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
}

type ExecutionRequest struct {
	ToolName string                 `json:"tool_name"`
	Action   string                 `json:"action"`
	Payload  map[string]interface{} `json:"payload"`
}

type ExecutionResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Gateway interface {
	ListTools(ctx context.Context) ([]ToolConnection, error)
	GetTool(ctx context.Context, name string) (*ToolConnection, error)
	Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error)
}

type Phase0Gateway struct{}

func (Phase0Gateway) ListTools(ctx context.Context) ([]ToolConnection, error) {
	return []ToolConnection{}, nil
}
func (Phase0Gateway) GetTool(ctx context.Context, name string) (*ToolConnection, error) {
	return nil, errors.New("tool not connected in phase 0")
}
func (Phase0Gateway) Execute(ctx context.Context, req ExecutionRequest) (*ExecutionResult, error) {
	return &ExecutionResult{Status: "not_implemented", Message: "not implemented in phase 0"}, nil
}

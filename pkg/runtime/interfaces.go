package runtime

import (
	"context"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// LLMGateway defines the interface for LLM interactions
type LLMGateway interface {
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

type ChatRequest struct {
	Model    string
	Messages []types.Message
	Tools    []types.Tool
}

type ChatResponse struct {
	Model     string
	Content   string
	ToolCalls []types.ToolCall
	Usage     types.Usage
}

// ToolExecutor defines the interface for tool execution
type ToolExecutor interface {
	Execute(ctx context.Context, call *types.ToolCall) (*types.ToolResult, error)
	List() []types.Tool
}

// PatchEngine defines the interface for file patching
type PatchEngine interface {
	GenerateDiff(ctx context.Context, filePath, newContent string) (string, error)
	Apply(ctx context.Context, patchID string) error
	// ... minimal interface for runtime
}

// AuditLogger defines the interface for audit logging
type AuditLogger interface {
	Log(ctx context.Context, event string, metadata map[string]any)
}

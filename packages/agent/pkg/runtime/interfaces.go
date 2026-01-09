package runtime

import (
	"context"

	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// LLMGateway defines the interface for LLM interactions
type LLMGateway interface {
	Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error)
	StreamChat(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamChunk, error)
}

// ToolExecutor defines the interface for tool execution
type ToolExecutor interface {
	Execute(ctx context.Context, mode types.RuntimeMode, call *types.ToolCall) (*types.ToolResult, error)
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

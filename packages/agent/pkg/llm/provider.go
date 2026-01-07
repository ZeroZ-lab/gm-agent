package llm

import (
	"context"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Provider defines the interface for an LLM provider (e.g., OpenAI, Anthropic)
type Provider interface {
	// ID returns the unique identifier of the provider
	ID() string

	// Call executes a synchronous chat request
	Call(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error)

	// CallStream executes a streaming chat request returning text chunks
	CallStream(ctx context.Context, req *ProviderRequest) (<-chan StreamChunk, error)
}

type StreamChunk struct {
	Content   string
	ToolCalls []types.ToolCall
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

type ProviderRequest struct {
	Model       string
	Messages    []types.Message
	Tools       []types.Tool
	MaxTokens   int
	Temperature float64
}

type ProviderResponse struct {
	ID        string
	Model     string
	Content   string
	ToolCalls []types.ToolCall
	Usage     types.Usage
}

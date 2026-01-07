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

	// TODO: Add Stream method
	// CallStream(ctx context.Context, req *ProviderRequest) (ProviderStream, error)
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

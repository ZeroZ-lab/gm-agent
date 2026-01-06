package llm

import (
	"context"

	"github.com/gm-agent-org/gm-agent/pkg/runtime"
)

type Gateway struct {
	provider Provider
}

func NewGateway(provider Provider) *Gateway {
	return &Gateway{
		provider: provider,
	}
}

func (g *Gateway) Chat(ctx context.Context, req *runtime.ChatRequest) (*runtime.ChatResponse, error) {
	// Map Runtime Request to Provider Request
	provReq := &ProviderRequest{
		Model:       req.Model,
		Messages:    req.Messages, // Assuming shared types.Message
		Tools:       req.Tools,    // Assuming shared types.Tool
		MaxTokens:   0,            // TODO: Config
		Temperature: 0.7,          // TODO: Config
	}

	resp, err := g.provider.Call(ctx, provReq)
	if err != nil {
		return nil, err
	}

	// Map Provider Response to Runtime Response
	return &runtime.ChatResponse{
		Model:     resp.Model,
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
		Usage:     resp.Usage,
	}, nil
}

package llm

import (
	"context"

	"github.com/gm-agent-org/gm-agent/pkg/config"
)

type Gateway struct {
	provider Provider
	options  config.ProviderOptions
}

func NewGateway(provider Provider, opts config.ProviderOptions) *Gateway {
	if opts.Temperature == 0 {
		opts.Temperature = 0.7 // Default if not set
	}
	return &Gateway{
		provider: provider,
		options:  opts,
	}
}

func (g *Gateway) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Map Runtime Request to Provider Request
	provReq := &ProviderRequest{
		Model:       req.Model,
		Messages:    req.Messages, // Assuming shared types.Message
		Tools:       req.Tools,    // Assuming shared types.Tool
		MaxTokens:   g.options.MaxTokens,
		Temperature: g.options.Temperature,
	}

	resp, err := g.provider.Call(ctx, provReq)
	if err != nil {
		return nil, err
	}

	// Map Provider Response to Runtime Response
	return &ChatResponse{
		Model:     resp.Model,
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
		Usage:     resp.Usage,
	}, nil
}
func (g *Gateway) StreamChat(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	provReq := &ProviderRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Tools:       req.Tools,
		MaxTokens:   g.options.MaxTokens,
		Temperature: g.options.Temperature,
	}

	return g.provider.CallStream(ctx, provReq)
}

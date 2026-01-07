package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/llm"
)

type Provider struct {
	ResponseContent string
}

func New(response string) *Provider {
	return &Provider{
		ResponseContent: response,
	}
}

func (p *Provider) ID() string {
	return "mock"
}

func (p *Provider) Call(ctx context.Context, req *llm.ProviderRequest) (*llm.ProviderResponse, error) {
	// Simple echo or predefined response
	content := p.ResponseContent
	if content == "" {
		lastMsg := req.Messages[len(req.Messages)-1]
		content = fmt.Sprintf("Mock response to: %s", lastMsg.Content)
	}

	return &llm.ProviderResponse{
		ID:      fmt.Sprintf("mock-%d", time.Now().UnixNano()),
		Model:   "mock-model",
		Content: content,
	}, nil
}

func (p *Provider) CallStream(ctx context.Context, req *llm.ProviderRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	go func() {
		defer close(ch)
		content := p.ResponseContent
		if content == "" {
			content = "Mock streaming response"
		}
		ch <- llm.StreamChunk{Content: content}
	}()
	return ch, nil
}

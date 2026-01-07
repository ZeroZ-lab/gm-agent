package llm

import (
	"context"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type stubProvider struct{}

func (stubProvider) ID() string { return "stub" }

func (stubProvider) Call(ctx context.Context, req *ProviderRequest) (*ProviderResponse, error) {
	return &ProviderResponse{Model: req.Model, Content: "ok", ToolCalls: []types.ToolCall{{Name: req.Tools[0].Name}}, Usage: types.Usage{TotalTokens: 1}}, nil
}

func TestGatewayChat(t *testing.T) {
	gw := NewGateway(stubProvider{}, config.ProviderOptions{})
	resp, err := gw.Chat(context.Background(), &runtime.ChatRequest{Model: "m", Messages: []types.Message{{Content: "hi"}}, Tools: []types.Tool{{Name: "t"}}})
	if err != nil {
		t.Fatalf("chat error: %v", err)
	}
	if resp.Model != "m" || resp.Content != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.ToolCalls == nil || len(resp.ToolCalls) != 1 {
		t.Fatalf("expected tool calls to propagate")
	}
	if resp.Usage.TotalTokens != 1 {
		t.Fatalf("expected usage to propagate")
	}
}

package mock

import (
	"context"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestProviderCall(t *testing.T) {
	p := New("preset")
	resp, err := p.Call(context.Background(), &llm.ProviderRequest{Messages: []types.Message{{Content: "hello"}}})
	if err != nil {
		t.Fatalf("call returned error: %v", err)
	}
	if resp.Content != "preset" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
}

func TestProviderCallEcho(t *testing.T) {
	p := New("")
	resp, err := p.Call(context.Background(), &llm.ProviderRequest{Messages: []types.Message{{Content: "hello"}}})
	if err != nil {
		t.Fatalf("call returned error: %v", err)
	}
	if resp.Content == "" {
		t.Fatalf("expected echoed content")
	}
}

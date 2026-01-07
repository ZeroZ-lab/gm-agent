package tool

import (
	"context"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	sample := types.Tool{Name: "read", Description: "d"}
	if err := reg.Register(sample); err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if err := reg.Register(sample); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
	got, ok := reg.Get("read")
	if !ok || got.Name != "read" {
		t.Fatalf("unexpected tool lookup: %+v", got)
	}
	if len(reg.List()) != 1 {
		t.Fatalf("expected one tool in list")
	}
}

func TestPolicyCheck(t *testing.T) {
	reg := NewRegistry()
	cfg := config.SecurityConfig{AllowedTools: []string{"safe"}, AllowFileSystem: false}
	policy := NewPolicy(cfg, reg, nil)

	if action, err := policy.Check(context.Background(), "safe", "{}"); err != nil || action != PolicyConfirm {
		t.Fatalf("expected confirm for allowed tool, got %v %v", action, err)
	}

	if _, err := policy.Check(context.Background(), "other", "{}"); err == nil {
		t.Fatalf("expected error for non-whitelisted tool")
	}

	cfg2 := config.SecurityConfig{AutoApprove: true, AllowFileSystem: false}

	// Register FS tool with metadata
	reg2 := NewRegistry()
	reg2.Register(types.Tool{Name: "read_file", Metadata: map[string]string{"category": "filesystem"}})

	policy = NewPolicy(cfg2, reg2, nil)
	if _, err := policy.Check(context.Background(), "read_file", "{}"); err == nil {
		t.Fatalf("expected filesystem denial when allow flag is false")
	}
}

func TestExecutor(t *testing.T) {
	reg := NewRegistry()
	if err := reg.Register(types.Tool{Name: "echo"}); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	policy := NewPolicy(config.SecurityConfig{AutoApprove: true}, reg, nil)
	exec := NewExecutor(reg, policy)

	exec.RegisterHandler("echo", func(ctx context.Context, args string) (string, error) {
		return args, nil
	})

	call := &types.ToolCall{ID: "1", Name: "echo", Arguments: "hello"}
	res, err := exec.Execute(context.Background(), call)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Content != "hello" || res.ToolCallID != "1" || res.ToolName != "echo" {
		t.Fatalf("unexpected result: %+v", res)
	}

	// Missing handler
	call.Name = "missing"
	if _, err := exec.Execute(context.Background(), call); err == nil {
		t.Fatalf("expected error for missing tool")
	}
}

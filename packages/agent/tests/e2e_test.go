//go:build e2e
// +build e2e

package integration_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type scriptedGateway struct{ toolName string }

func (g scriptedGateway) Chat(ctx context.Context, req *runtime.ChatRequest) (*runtime.ChatResponse, error) {
	return &runtime.ChatResponse{
		Model: "scripted-model",
		ToolCalls: []types.ToolCall{{
			ID:        "call-1",
			Name:      g.toolName,
			Arguments: "{}",
		}},
	}, nil
}

func TestEndToEndGoalCompletion(t *testing.T) {
	ctx := context.Background()
	dataDir := t.TempDir()

	fsStore := store.NewFSStore(dataDir)
	if err := fsStore.Open(ctx); err != nil {
		t.Fatalf("open store: %v", err)
	}

	registry := tool.NewRegistry()
	policy := tool.NewPolicy(config.SecurityConfig{AutoApprove: true}, registry)
	executor := tool.NewExecutor(registry, policy)

	taskCompleteTool := types.Tool{Name: "task_complete", Description: "mark work done", Parameters: types.JSONSchema{"type": "object"}}
	if err := registry.Register(taskCompleteTool); err != nil {
		t.Fatalf("register tool: %v", err)
	}
	executor.RegisterHandler("task_complete", func(ctx context.Context, args string) (string, error) {
		return "completed", nil
	})

	rt := runtime.New(runtime.Config{
		MaxSteps:           5,
		CheckpointInterval: 1,
		DecisionTimeout:    2 * time.Second,
		DispatchTimeout:    2 * time.Second,
		Model:              "scripted",
	}, fsStore, scriptedGateway{toolName: "task_complete"}, executor, slog.New(slog.NewTextHandler(io.Discard, nil)))

	reqEvent := &types.UserMessageEvent{BaseEvent: types.NewBaseEvent("user_request", "user", "e2e"), Content: "finish", Priority: 1}
	if err := rt.Ingest(ctx, reqEvent); err != nil {
		t.Fatalf("ingest: %v", err)
	}

	if err := rt.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}

	state, err := fsStore.LoadLatestState(ctx)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}

	if len(state.Goals) == 0 || state.Goals[0].Status != types.GoalStatusCompleted {
		t.Fatalf("expected completed goal, got %+v", state.Goals)
	}
	if len(state.Context.Messages) < 2 {
		t.Fatalf("expected at least user and tool messages, got %d", len(state.Context.Messages))
	}
}

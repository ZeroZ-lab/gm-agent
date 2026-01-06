package integration_test

import (
	"context"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/mock"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestEndToEnd_Skeleton(t *testing.T) {
	// 1. Setup Temp Dir
	tmpDir, err := os.MkdirTemp("", "gm-agent-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// 2. Setup Modules
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	fsStore := store.NewFSStore(tmpDir)
	if err := fsStore.Open(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Use Mock Provider
	mockProvider := mock.New("Result is 2")
	llmGateway := llm.NewGateway(mockProvider)
	toolRegistry := tool.NewRegistry()

	toolPolicy := tool.NewPolicy(config.SecurityConfig{
		AutoApprove: true, // Allow all for testing
	})
	toolExecutor := tool.NewExecutor(toolRegistry, toolPolicy)

	config := runtime.Config{
		MaxSteps:           5, // Limit steps for test
		CheckpointInterval: 10,
		DecisionTimeout:    5 * time.Second,
		DispatchTimeout:    5 * time.Second,
	}

	rt := runtime.New(config, fsStore, llmGateway, toolExecutor, logger)

	// 3. Inject Start Event (User Request)
	ctx := context.Background()
	reqEvent := &types.UserMessageEvent{
		BaseEvent: types.NewBaseEvent("user_request", "user", "integration"),
		Content:   "Calculate 1 + 1",
		Priority:  1,
	}

	if err := rt.Ingest(ctx, reqEvent); err != nil {
		t.Fatalf("ingest failed: %v", err)
	}

	// 4. Run (Run should process at least one step and then stop or loop)
	// Since LLM returns text response (mock), Reducer will add it to context.
	// But Goal status is never set to Completed in our skeleton Reducer logic.
	// So it will run for MaxSteps (5).

	err = rt.Run(ctx)
	if err == nil {
		t.Fatal("expected max steps exceeded error, got nil")
	}
	if err.Error() != "max steps exceeded" {
		t.Fatalf("expected max steps error, got: %v", err)
	}

	// 5. Verify State
	state, err := fsStore.LoadLatestState(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Goals should exist
	if len(state.Goals) == 0 {
		t.Error("expected at least one goal created")
	}
	if state.Goals[0].Description != "Calculate 1 + 1" {
		t.Errorf("expected goal 'Calculate 1 + 1', got %s", state.Goals[0].Description)
	}

	// Context should have messages:
	// 1. User Message
	// 2. Assistant Message (Mock LLM)
	// 3. ... looped
	if len(state.Context.Messages) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(state.Context.Messages))
	} else {
		lastMsg := state.Context.Messages[len(state.Context.Messages)-1]
		if lastMsg.Role != "assistant" {
			t.Errorf("expected last message from assistant, got %s", lastMsg.Role)
		}
	}
}

package runtime

import (
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestReducerArtifact(t *testing.T) {
	state := types.NewState()
	r := &Runtime{state: state}

	// Simulate Assistant Tool Call
	msg := types.Message{
		Role: "assistant",
		ToolCalls: []types.ToolCall{
			{ID: "call_1", Name: "create_file", Arguments: `{"path": "/tmp/test.txt", "content": "hello"}`},
		},
	}
	state.Context.Messages = append(state.Context.Messages, msg)

	// Simulate Tool Result
	event := &types.ToolResultEvent{
		BaseEvent:  types.NewBaseEvent("tool_result", "system", "sid"),
		ToolCallID: "call_1",
		ToolName:   "create_file",
		Output:     "Success",
		Success:    true,
	}
	// Hack: Set timestamp artificially because NewBaseEvent sets Now
	// Actually NewBaseEvent is fine.

	// Call reducer
	newState, _, err := r.reducer(state, event)
	if err != nil {
		t.Fatalf("reducer failed: %v", err)
	}

	if len(newState.Artifacts) != 1 {
		t.Errorf("expected 1 artifact, got %d", len(newState.Artifacts))
	}

	for _, art := range newState.Artifacts {
		if art.Path != "/tmp/test.txt" {
			t.Errorf("expected path /tmp/test.txt, got %s", art.Path)
		}
		if art.Name != "test.txt" {
			t.Errorf("expected name test.txt, got %s", art.Name)
		}
		if art.Type != "file" {
			t.Errorf("expected type file, got %s", art.Type)
		}
		// Check Time roughly
		if time.Since(art.CreatedAt) > time.Minute {
			t.Errorf("created at seems wrong: %v", art.CreatedAt)
		}
	}
}

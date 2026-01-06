package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// applyEvent applies the event to state and collects side-effect commands
func (r *Runtime) applyEvent(ctx context.Context, event types.Event) error {
	newState, cmds, err := r.safeReduce(ctx, r.state, event)
	if err != nil {
		// Log Reducer Error (likely panic)
		r.log.Error("reducer failed", "error", err)
		return err // Fatal
	}

	r.state = newState
	r.pendingCommands = append(r.pendingCommands, cmds...)

	// Update State in Store (Optimistic save, or wait for checkpoint?)
	// Architecture spec says: "Store: SaveState... Checkpoint"
	// `Run` loop calls `checkpoint` periodically.
	// But `AppendEvent` was called in `dispatch`.
	// State is in-memory and persisted at checkpoint.

	return nil
}

// safeReduce wraps reducer with panic recovery
func (r *Runtime) safeReduce(ctx context.Context, state *types.State, event types.Event) (_ *types.State, _ []types.Command, err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("reducer panic: %v\nstack: %s", p, debug.Stack())
		}
	}()
	return r.reducer(state, event)
}

// reducer is the pure function (State, Event) -> (State, Commands)
func (r *Runtime) reducer(state *types.State, event types.Event) (*types.State, []types.Command, error) {
	// For MVP, simplistic implementation

	// TODO: Deep copy state ideally?
	// Go maps are referenced. Modifying `state` directly updates it.
	// Strict functional reducer would clone state first.
	// r.state.Clone() needed here. for MVP we mutate in place for now but caution.

	state.Version++
	state.UpdatedAt = event.EventTimestamp()

	var cmds []types.Command

	switch e := event.(type) {
	case *types.UserMessageEvent:
		// User added a message -> Maybe trigger LLM?
		// Add to context
		msg := types.Message{
			Role:    "user",
			Content: e.Content,
			// ID, Timestamp...
		}
		state.Context.Messages = append(state.Context.Messages, msg)

		// If Semantic is Fork, create new goal?
		// For now, if no goal, create one?
		if len(state.Goals) == 0 {
			goal := types.Goal{
				ID:          types.GenerateGoalID(),
				Description: e.Content,
				Status:      types.GoalStatusPending,
				Type:        types.GoalTypeUserRequest,
			}
			state.Goals = append(state.Goals, goal)
		}

	case *types.LLMResponseEvent:
		// LLM replied
		msg := types.Message{
			Role:      "assistant",
			Content:   e.Content,
			ToolCalls: e.ToolCalls, // IMPORTANT: Include tool calls in message history
		}
		if msg.Content == "" {
			msg.Content = " " // Hack: Some providers reject empty content
		}
		state.Context.Messages = append(state.Context.Messages, msg)

		// If tool calls, generate ToolCall commands?
		// Wait, LLMResponseEvent in `dispatch` was generated AFTER `CallLLMCommand` executed.
		// The command execution in `dispatch` produced the Event.
		// BUT `dispatch` is responsible for commands.
		// So `CallLLMCommand` -> (LLM Call) -> `LLMResponseEvent`.
		// Now Reducer sees `LLMResponseEvent`.
		// Does Reducer generate `ToolCallCommand`?
		// YES. This is the "Loop".
		// LLM said "Call Tool X", so we generated an event saying "LLM said this".
		// Now Reducer should transform "LLM said call X" into "Command: Call X".

		for _, tc := range e.ToolCalls {
			args := map[string]any{}
			if tc.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					args = map[string]any{}
				}
			}
			cmd := &types.CallToolCommand{
				BaseCommand: types.NewBaseCommand("call_tool"),
				ToolCallID:  tc.ID,
				ToolName:    tc.Name,
				Arguments:   args,
			}
			cmds = append(cmds, cmd)
		}

	case *types.ToolResultEvent:
		// Tool finished
		msg := types.Message{
			Role:       "tool",
			Content:    e.Output,
			ToolCallID: e.ToolCallID,
			ToolName:   e.ToolName,
		}
		if !e.Success {
			msg.Content = fmt.Sprintf("Error: %s", e.Error)
		}
		state.Context.Messages = append(state.Context.Messages, msg)

		// Special Handling: task_complete
		if e.ToolName == "task_complete" && e.Success {
			// Find active goal
			// For MVP, simplistic: Assume first pending/in-progress is targets
			for i := range state.Goals {
				if state.Goals[i].Status == types.GoalStatusPending || state.Goals[i].Status == types.GoalStatusInProgress {
					state.Goals[i].Status = types.GoalStatusCompleted
					// Don't break if we want to mark all?
					// For now, mark the current active one as completed.
					break
				}
			}
		}
	}

	return state, cmds, nil
}

package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime/debug"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// applyEvent applies the event to state and collects side-effect commands
// This method acquires its own locks and should NOT be called while holding r.mu
func (r *Runtime) applyEvent(ctx context.Context, event types.Event) error {
	// Get current state with lock for reducer input
	r.mu.RLock()
	currentState := r.state
	r.mu.RUnlock()

	newState, cmds, err := r.safeReduce(ctx, currentState, event)
	if err != nil {
		// Log Reducer Error (likely panic)
		r.log.Error("reducer failed", "error", err)
		return err // Fatal
	}

	// Atomically update state and append pending commands
	r.updateState(newState)
	if len(cmds) > 0 {
		r.appendPendingCommands(cmds)
	}

	return nil
}

// applyEventLocked applies the event to state while caller already holds r.mu write lock
// This is used by Ingest which needs to hold the lock across multiple operations
func (r *Runtime) applyEventLocked(ctx context.Context, event types.Event) error {
	newState, cmds, err := r.safeReduce(ctx, r.state, event)
	if err != nil {
		r.log.Error("reducer failed", "error", err)
		return err
	}

	// Update state directly since we already hold the lock
	r.state = newState
	r.pendingCommands = append(r.pendingCommands, cmds...)

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
// Follows immutability principle: clones state before mutation
func (r *Runtime) reducer(state *types.State, event types.Event) (*types.State, []types.Command, error) {
	// Deep copy state first to ensure immutability
	// This prevents partial state corruption if a panic occurs mid-processing
	newState := state.Clone()

	newState.Version++
	newState.UpdatedAt = event.EventTimestamp()

	var cmds []types.Command

	switch e := event.(type) {
	case *types.SystemPromptEvent:
		// Update System Prompt configuration
		newState.SystemPrompt = e.Prompt

	case *types.UserMessageEvent:
		// User added a message -> Maybe trigger LLM?
		// Add to context
		msg := types.Message{
			Role:    "user",
			Content: e.Content,
			// ID, Timestamp...
		}
		newState.Context.Messages = append(newState.Context.Messages, msg)

		// Check if there's any active (pending or in-progress) goal
		hasActiveGoal := false
		for _, g := range newState.Goals {
			if g.Status == types.GoalStatusPending || g.Status == types.GoalStatusInProgress {
				hasActiveGoal = true
				break
			}
		}

		// If no active goal, create one for this user message
		if !hasActiveGoal {
			goal := types.Goal{
				ID:          types.GenerateGoalID(),
				Description: e.Content,
				Status:      types.GoalStatusPending,
				Type:        types.GoalTypeUserRequest,
			}
			newState.Goals = append(newState.Goals, goal)
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
		newState.Context.Messages = append(newState.Context.Messages, msg)

		// If LLM responded with content but NO tool calls, this is a direct response
		// The user will receive the content from the llm_response event via streaming
		// Mark the current goal as complete since the LLM answered directly
		if len(e.ToolCalls) == 0 && e.Content != "" && e.Content != " " {
			// Find and complete the active goal
			for i := range newState.Goals {
				if newState.Goals[i].Status == types.GoalStatusPending || newState.Goals[i].Status == types.GoalStatusInProgress {
					newState.Goals[i].Status = types.GoalStatusCompleted
					break
				}
			}
		}

		// If tool calls exist, generate ToolCall commands
		for _, tc := range e.ToolCalls {
			args := map[string]any{}
			if tc.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					// Log the error instead of silently ignoring
					r.log.Warn("failed to parse tool call arguments",
						"tool_call_id", tc.ID,
						"tool_name", tc.Name,
						"error", err)
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
		newState.Context.Messages = append(newState.Context.Messages, msg)

		// Special Handling: task_complete
		if e.ToolName == "task_complete" && e.Success {
			// Find active goal
			// For MVP, simplistic: Assume first pending/in-progress is targets
			for i := range newState.Goals {
				if newState.Goals[i].Status == types.GoalStatusPending || newState.Goals[i].Status == types.GoalStatusInProgress {
					newState.Goals[i].Status = types.GoalStatusCompleted
					// Don't break if we want to mark all?
					// For now, mark the current active one as completed.
					break
				}
			}
		}
		// Special Handling: create_file success
		if e.ToolName == "create_file" && e.Success {
			// Find arguments from context
			var args struct {
				Path string `json:"path"`
			}
			// Search backwards in messages to find the Assistant message with this ToolCallID
			found := false
			for i := len(newState.Context.Messages) - 1; i >= 0; i-- {
				msg := newState.Context.Messages[i]
				if msg.Role == "assistant" {
					for _, tc := range msg.ToolCalls {
						if tc.ID == e.ToolCallID {
							if err := json.Unmarshal([]byte(tc.Arguments), &args); err == nil && args.Path != "" {
								// Create Artifact
								artID := types.GenerateID("art")
								artifact := &types.Artifact{
									ID:        artID,
									Type:      "file",
									Name:      filepath.Base(args.Path),
									Path:      args.Path,
									CreatedAt: e.EventTimestamp(),
								}
								newState.Artifacts[artID] = artifact
							}
							found = true
							break
						}
					}
				}
				if found {
					break
				}
			}
		}
	}

	return newState, cmds, nil
}

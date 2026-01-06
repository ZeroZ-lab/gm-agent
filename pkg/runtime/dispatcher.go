package runtime

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// dispatch executes a list of commands and returns produced events
func (r *Runtime) dispatch(ctx context.Context, cmds []types.Command) ([]types.Event, error) {
	var allEvents []types.Event

	for _, cmd := range cmds {
		var events []types.Event
		var err error

		// Deps injection
		// This is where we wire dependencies.
		// NOTE: In strict architecture, Command.Execute should be a method on the runtime/dispatcher
		// rather than on the DTO itself to avoid dragging deps into `pkg/types`.
		// Since we removed `Execute` from types.Command, we handle it here via switch.

		switch c := cmd.(type) {
		case *types.CallLLMCommand:
			events, err = r.executeCallLLM(ctx, c)
		case *types.CallToolCommand:
			events, err = r.executeCallTool(ctx, c)
		// case *types.ApplyPatchCommand:
		default:
			// log unknown
		}

		if err != nil {
			fmt.Printf("Command Execution Failed: %v\n", err)
			r.log.Error("command execution failed", "command_id", cmd.CommandID(), "error", err)
			// Generate ErrorEvent
			errEvent := &types.ErrorEvent{
				BaseEvent: types.NewBaseEvent("error", "runtime", ""),
				CommandID: cmd.CommandID(),
				Error:     err.Error(),
				Severity:  types.SeverityRecoverable, // Default
			}
			allEvents = append(allEvents, errEvent)
			// For MVP, we don't break on error events, we feed them back to LLM
		} else {
			allEvents = append(allEvents, events...)
		}
	}

	// Persist all events
	if len(allEvents) > 0 {
		if err := r.store.AppendEvents(ctx, allEvents); err != nil {
			return nil, err
		}
	}

	return allEvents, nil
}

func (r *Runtime) executeCallLLM(ctx context.Context, cmd *types.CallLLMCommand) ([]types.Event, error) {
	req := &ChatRequest{
		Model:    cmd.Model,
		Messages: cmd.Messages,
		Tools:    cmd.Tools,
	}
	resp, err := r.llm.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	// Debug Log
	r.log.Info("llm response", "content", resp.Content, "tool_calls", len(resp.ToolCalls))

	if resp.Content != "" {
		fmt.Printf("\nAgent: %s\n", resp.Content)
	}

	evt := &types.LLMResponseEvent{
		BaseEvent: types.NewBaseEvent("llm_response", "llm", ""),
		Model:     resp.Model,
		Content:   resp.Content,
		ToolCalls: resp.ToolCalls,
		Usage:     resp.Usage,
	}
	return []types.Event{evt}, nil
}

func (r *Runtime) executeCallTool(ctx context.Context, cmd *types.CallToolCommand) ([]types.Event, error) {
	call := &types.ToolCall{
		// ID: assign? Or use command ID?
		Name:      cmd.ToolName,
		Arguments: "{}", // TODO: Marshal map[string]any to string
	}

	result, err := r.tools.Execute(ctx, call)

	var resEvent *types.ToolResultEvent
	if err != nil {
		resEvent = &types.ToolResultEvent{
			BaseEvent: types.NewBaseEvent("tool_result", "tool", cmd.ToolName),
			ToolName:  cmd.ToolName,
			Success:   false,
			Error:     err.Error(),
		}
	} else {
		resEvent = &types.ToolResultEvent{
			BaseEvent: types.NewBaseEvent("tool_result", "tool", cmd.ToolName),
			ToolName:  cmd.ToolName,
			Success:   true,
			Output:    result.Content,
		}
	}
	return []types.Event{resEvent}, nil
}

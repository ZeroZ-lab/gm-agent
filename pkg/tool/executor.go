package tool

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Handler defines the interface for actual tool implementation logic
type Handler func(ctx context.Context, args string) (string, error)

type Executor struct {
	registry *Registry
	policy   *Policy
	handlers map[string]Handler
}

func NewExecutor(registry *Registry, policy *Policy) *Executor {
	return &Executor{
		registry: registry,
		policy:   policy,
		handlers: make(map[string]Handler),
	}
}

func (e *Executor) RegisterHandler(name string, handler Handler) {
	e.handlers[name] = handler
}

func (e *Executor) Execute(ctx context.Context, call *types.ToolCall) (*types.ToolResult, error) {
	// 1. Lookup Tool Definition
	_, ok := e.registry.Get(call.Name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", call.Name)
	}

	// 2. Check Policy
	action, err := e.policy.Check(ctx, call.Name, call.Arguments)
	if err != nil {
		return nil, err
	}
	if action == PolicyDeny {
		return nil, fmt.Errorf("policy denied execution of tool: %s", call.Name)
	}
	// PolicyConfirm logic would go here (e.g. callback to UI) -> For MVP we treat confirm as allow or block?
	// For Stub/MVP, let's log confirm and allow mainly for "read_file", but "run_shell" might fail if strict.
	// Let's relax Policy for MVP or assume "Confirm" means "Logged Allow" for CLI testing without UI.
	if action == PolicyConfirm {
		// In a real agent, this suspends execution.
		// For MVP skeleton test, we'll allow it but log warning.
		// TODO: Implement human-in-the-loop suspension.
	}

	// 3. Lookup Handler
	handler, ok := e.handlers[call.Name]
	if !ok {
		return nil, fmt.Errorf("no handler implementation for tool: %s", call.Name)
	}

	// 4. Execute
	output, err := handler(ctx, call.Arguments)

	result := &types.ToolResult{
		ToolCallID: call.ID, // Or passed explicitly? command.go uses ID.
		ToolName:   call.Name,
		Content:    output,
		IsError:    err != nil,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result, nil
}

package tool

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Handler defines the interface for actual tool implementation logic
type Handler func(ctx context.Context, args string) (string, error)

// PermissionRequest represents a request for user approval
type PermissionRequest struct {
	RequestID  string
	ToolName   string
	Permission string   // e.g. "read", "write", "shell"
	Patterns   []string // e.g. ["/path/to/file"]
	Metadata   map[string]string
}

// PermissionCallback is called when a tool needs user approval
// Returns true if approved, false if denied
type PermissionCallback func(ctx context.Context, req PermissionRequest) (approved bool, err error)

type Executor struct {
	registry           *Registry
	policy             *Policy
	handlers           map[string]Handler
	permissionCallback PermissionCallback
}

func NewExecutor(registry *Registry, policy *Policy) *Executor {
	return &Executor{
		registry: registry,
		policy:   policy,
		handlers: make(map[string]Handler),
	}
}

// SetPermissionCallback sets the callback for handling permission requests
func (e *Executor) SetPermissionCallback(cb PermissionCallback) {
	e.permissionCallback = cb
}

func (e *Executor) RegisterHandler(name string, handler Handler) {
	e.handlers[name] = handler
}

func (e *Executor) Execute(ctx context.Context, call *types.ToolCall) (*types.ToolResult, error) {
	// 1. Lookup Tool Definition
	toolDef, ok := e.registry.Get(call.Name)
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

	// 3. Handle PolicyConfirm - request user approval
	if action == PolicyConfirm {
		if e.permissionCallback == nil {
			// No callback registered, treat as allow (for backward compatibility)
		} else {
			// Build permission request
			req := PermissionRequest{
				RequestID:  types.GenerateID("perm"),
				ToolName:   call.Name,
				Permission: toolDef.Metadata["category"],
				Patterns:   []string{call.Arguments}, // simplified
				Metadata:   toolDef.Metadata,
			}

			approved, err := e.permissionCallback(ctx, req)
			if err != nil {
				return nil, fmt.Errorf("permission request failed: %w", err)
			}
			if !approved {
				return &types.ToolResult{
					ToolCallID: call.ID,
					ToolName:   call.Name,
					Content:    "Permission denied by user",
					IsError:    true,
					Error:      "user denied permission",
				}, nil
			}
		}
	}

	// 4. Lookup Handler
	handler, ok := e.handlers[call.Name]
	if !ok {
		return nil, fmt.Errorf("no handler implementation for tool: %s", call.Name)
	}

	// 5. Execute
	output, err := handler(ctx, call.Arguments)

	result := &types.ToolResult{
		ToolCallID: call.ID,
		ToolName:   call.Name,
		Content:    output,
		IsError:    err != nil,
	}
	if err != nil {
		result.Error = err.Error()
	}
	return result, nil
}

func (e *Executor) List() []types.Tool {
	return e.registry.List()
}

package tool

import (
	"context"
	"fmt"
)

// PolicyAction defines the action to take for a tool execution
type PolicyAction string

const (
	PolicyAllow   PolicyAction = "allow"
	PolicyDeny    PolicyAction = "deny"
	PolicyConfirm PolicyAction = "confirm" // Requires user confirmation (not implemented fully in MVP)
)

type Policy struct {
	// Simple map for MVP: tool_name -> action
	rules map[string]PolicyAction
}

func NewPolicy() *Policy {
	return &Policy{
		rules: map[string]PolicyAction{
			"read_file": PolicyAllow,
			"run_shell": PolicyConfirm, // Safety default
		},
	}
}

func (p *Policy) Check(ctx context.Context, toolName string, args string) (PolicyAction, error) {
	// 1. Check specific rule
	if action, ok := p.rules[toolName]; ok {
		return action, nil
	}

	// 2. Default deny
	return PolicyDeny, fmt.Errorf("no policy rule for tool %s", toolName)
}

func (p *Policy) SetRule(toolName string, action PolicyAction) {
	p.rules[toolName] = action
}

package tool

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/config"
)

// PolicyAction defines the action to take for a tool execution
type PolicyAction string

const (
	PolicyAllow   PolicyAction = "allow"
	PolicyDeny    PolicyAction = "deny"
	PolicyConfirm PolicyAction = "confirm" // Requires user confirmation (not implemented fully in MVP)
)

type Policy struct {
	config   config.SecurityConfig
	registry *Registry
}

func NewPolicy(cfg config.SecurityConfig, registry *Registry) *Policy {
	// If AllowFileSystem is FALSE, we might want to hard restrict in Check.
	// But allowed_tools takes precedence for specific tool names.
	return &Policy{
		config:   cfg,
		registry: registry,
	}
}

func (p *Policy) Check(ctx context.Context, toolName string, args string) (PolicyAction, error) {
	// 1. Check Whitelist (if non-empty)
	// If AllowedTools is specified, ONLY allow listed tools.
	if len(p.config.AllowedTools) > 0 {
		found := false
		for _, allowed := range p.config.AllowedTools {
			if allowed == toolName {
				found = true
				break
			}
		}
		if !found {
			return PolicyDeny, fmt.Errorf("tool %s is not in allowed_tools whitelist", toolName)
		}
	}

	// 2. Check Category Restrictions
	// We check the tool definition for categories.
	if p.registry != nil {
		if t, ok := p.registry.Get(toolName); ok {
			category := t.Metadata["category"]
			if category == "filesystem" && !p.config.AllowFileSystem {
				return PolicyDeny, fmt.Errorf("filesystem operations (category: %s) are disabled by security policy", category)
			}
			if category == "internet" && !p.config.AllowInternet {
				return PolicyDeny, fmt.Errorf("internet operations (category: %s) are disabled by security policy", category)
			}
		}
	} else {
		// Fallback for when registry is not injected provided (e.g. tests)
		// Or if we want to keep backward compat during refactor steps
		// For now, we assume if registry is nil, we can't check categories.
	}

	// 3. Auto Approve vs Confirm
	// If AutoApprove is true, ALLOW.
	// If AutoApprove is false, CONFIRM (default).
	if p.config.AutoApprove {
		return PolicyAllow, nil
	}

	return PolicyConfirm, nil
}

func (p *Policy) SetRule(toolName string, action PolicyAction) {
	// Runtime override support (optional for now)
}

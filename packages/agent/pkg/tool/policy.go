package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// PolicyAction defines the action to take for a tool execution
type PolicyAction string

const (
	PolicyAllow   PolicyAction = "allow"
	PolicyDeny    PolicyAction = "deny"
	PolicyConfirm PolicyAction = "confirm" // Requires user confirmation (not implemented fully in MVP)
)

type PermissionReader interface {
	GetPermissionRules(ctx context.Context) ([]types.PermissionRule, error)
}

type Policy struct {
	config   config.SecurityConfig
	registry *Registry
	store    PermissionReader
}

func NewPolicy(cfg config.SecurityConfig, registry *Registry, store PermissionReader) *Policy {
	// If AllowFileSystem is FALSE, we might want to hard restrict in Check.
	// But allowed_tools takes precedence for specific tool names.
	return &Policy{
		config:   cfg,
		registry: registry,
		store:    store,
	}
}

func (p *Policy) Check(ctx context.Context, mode types.RuntimeMode, toolName string, args string) (PolicyAction, error) {
	// 1. Mode-based restriction (HIGHEST PRIORITY)
	// In planning mode, only allow read-only tools
	if mode == types.ModePlanning {
		if t, ok := p.registry.Get(toolName); ok && !t.ReadOnly {
			return PolicyDeny, fmt.Errorf(
				"tool %s requires write access and cannot be used in planning mode",
				toolName,
			)
		}
	}

	// 2. Check Whitelist (if non-empty)
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

	// 3. Check Persistent Permission Rules
	if p.store != nil {
		rules, err := p.store.GetPermissionRules(ctx)
		normalizedArgs := NormalizeArguments(args)
		if err == nil {
			for _, rule := range rules {
				// We compare normalized arguments
				if rule.ToolName == toolName && NormalizeArguments(rule.Pattern) == normalizedArgs {
					if rule.Action == "allow" {
						return PolicyAllow, nil
					} else if rule.Action == "deny" {
						return PolicyDeny, fmt.Errorf("denied by persistent rule")
					}
				}
			}
		}
	}

	// 4. Auto Approve vs Confirm
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

// NormalizeArguments sorts JSON keys to ensure consistent string representation
func NormalizeArguments(s string) string {
	var v interface{}
	// If not valid JSON, return as is (strict string match)
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	// Re-marshal to sort keys
	b, _ := json.Marshal(v)
	return string(b)
}

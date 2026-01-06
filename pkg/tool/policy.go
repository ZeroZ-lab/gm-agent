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
	config config.SecurityConfig
}

func NewPolicy(cfg config.SecurityConfig) *Policy {
	// If AllowFileSystem is FALSE, we might want to hard restrict in Check.
	// But allowed_tools takes precedence for specific tool names.
	return &Policy{
		config: cfg,
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

	// 2. Check Global Flags
	// Example: Deny FS tools if AllowFileSystem is false
	// We need to know which tools are FS tools.
	// MVP: Check prefix or hardcoded list.
	fsTools := map[string]bool{"read_file": true, "write_file": true, "list_dir": true}
	if !p.config.AllowFileSystem && fsTools[toolName] {
		return PolicyDeny, fmt.Errorf("filesystem operations are disabled by security policy")
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

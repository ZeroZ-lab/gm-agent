package types

import "time"

// PermissionRule represents a persistent rule for tool execution
type PermissionRule struct {
	ID        string    `json:"id"`
	ToolName  string    `json:"tool_name"`
	Action    string    `json:"action"`  // "allow" or "deny"
	Pattern   string    `json:"pattern"` // Exact match for arguments
	CreatedAt time.Time `json:"created_at"`
}

package types

import "time"

// Tool definition
type Tool struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Parameters  JSONSchema        `json:"parameters"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ToolCall represents an invocation request from LLM
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolResult represents the output of a tool execution
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
	Error      string `json:"error,omitempty"`
}

// FileChange represents a file modification that can be reverted
type FileChange struct {
	PatchID    string `json:"patch_id"`              // Unique patch identifier
	FilePath   string `json:"file_path"`             // Relative path to the modified file
	BackupPath string `json:"backup_path,omitempty"` // Path to backup file
	Operation  string `json:"operation"`             // "create", "modify", "delete"
}

// Checkpoint structure
type Checkpoint struct {
	ID           string    `json:"id"`
	StateVersion int64     `json:"state_version"`
	LastEventID  string    `json:"last_event_id"`
	Timestamp    time.Time `json:"timestamp"`

	// Optional embedded state
	State *State `json:"state,omitempty"`

	// FileChanges tracks all file modifications since the previous checkpoint
	// Used for Code Rewind functionality
	FileChanges []FileChange `json:"file_changes,omitempty"`
}

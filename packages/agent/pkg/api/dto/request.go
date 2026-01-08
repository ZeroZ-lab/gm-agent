package dto

// CreateSessionRequest is the request body for creating a new session.
type CreateSessionRequest struct {
	Prompt       string `json:"prompt,omitempty"`        // Optional: if empty, session is created without LLM call
	SystemPrompt string `json:"system_prompt,omitempty"`
	Priority     int    `json:"priority,omitempty"`
	Constraints  any    `json:"constraints,omitempty"`
}

// MessageRequest is the request body for posting a message to a session.
type MessageRequest struct {
	Content  string `json:"content" binding:"required"`
	Semantic string `json:"semantic,omitempty"` // append/fork/preempt/cancel
}

// PermissionResponseRequest is the request body for responding to a permission request
type PermissionResponseRequest struct {
	RequestID string `json:"request_id" binding:"required"`
	Approved  bool   `json:"approved"`
	Always    bool   `json:"always"`
}

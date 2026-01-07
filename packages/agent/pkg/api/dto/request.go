package dto

// CreateSessionRequest is the request body for creating a new session.
type CreateSessionRequest struct {
	Prompt      string `json:"prompt" binding:"required"`
	Priority    int    `json:"priority,omitempty"`
	Constraints any    `json:"constraints,omitempty"`
}

// MessageRequest is the request body for posting a message to a session.
type MessageRequest struct {
	Content  string `json:"content" binding:"required"`
	Semantic string `json:"semantic,omitempty"` // append/fork/preempt/cancel
}

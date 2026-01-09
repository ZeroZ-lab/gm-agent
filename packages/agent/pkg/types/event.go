package types

import "time"

// Event is the interface for all system events
type Event interface {
	EventID() string
	EventType() string
	EventTimestamp() time.Time
	EventActor() string
	EventSubject() string

	// Data returns the payload for storage
	// We might need to expose the underlying struct or a map for serialization if not using json directly on the wrapper
	// For now, struct tags handle serialization of the concrete types.
}

// BaseEvent is embedded in all specific event types
type BaseEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Subject   string    `json:"subject"`
}

func (e *BaseEvent) EventID() string           { return e.ID }
func (e *BaseEvent) EventType() string         { return e.Type }
func (e *BaseEvent) EventTimestamp() time.Time { return e.Timestamp }
func (e *BaseEvent) EventActor() string        { return e.Actor }
func (e *BaseEvent) EventSubject() string      { return e.Subject }

func NewBaseEvent(eventType, actor, subject string) BaseEvent {
	return BaseEvent{
		ID:        GenerateEventID(),
		Type:      eventType,
		Timestamp: time.Now(),
		Actor:     actor,
		Subject:   subject,
	}
}

// UserMessageEvent
type UserMessageEvent struct {
	BaseEvent
	Content  string   `json:"content"`
	Priority int      `json:"priority"`
	Semantic Semantic `json:"semantic"`
}

// SystemPromptEvent
type SystemPromptEvent struct {
	BaseEvent
	Prompt string `json:"prompt"`
}

// LLMResponseEvent
type LLMResponseEvent struct {
	BaseEvent
	Model     string     `json:"model"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
}

// LLMTokenEvent represents an incremental text chunk from LLM
type LLMTokenEvent struct {
	BaseEvent
	Delta string `json:"delta"`
}

// ToolResultEvent
type ToolResultEvent struct {
	BaseEvent
	ToolCallID string `json:"tool_call_id"`
	ToolName   string `json:"tool_name"`
	Success    bool   `json:"success"`
	Output     string `json:"output"`
	Error      string `json:"error,omitempty"`
	Duration   int64  `json:"duration_ms"`
}

// ErrorEvent
type ErrorEvent struct {
	BaseEvent
	CommandID string        `json:"command_id"`
	Error     string        `json:"error"`
	Severity  ErrorSeverity `json:"severity"`
}

// CheckpointEvent
type CheckpointEvent struct {
	BaseEvent
	CheckpointID string `json:"checkpoint_id"`
	StateVersion int64  `json:"state_version"`
}

// Usage statistics for LLM
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// PermissionRequestEvent is emitted when a tool needs user approval
type PermissionRequestEvent struct {
	BaseEvent
	RequestID  string            `json:"request_id"`
	ToolName   string            `json:"tool_name"`
	Permission string            `json:"permission"` // e.g. "read", "write", "shell", "network"
	Patterns   []string          `json:"patterns"`   // e.g. ["/path/to/file"]
	Metadata   map[string]string `json:"metadata"`   // Additional context
}

// PermissionResponseEvent is the user's response to a permission request
type PermissionResponseEvent struct {
	BaseEvent
	RequestID string `json:"request_id"`
	Approved  bool   `json:"approved"`
	Always    bool   `json:"always"` // If true, always allow this pattern
}

// PlanGeneratedEvent is emitted when the LLM generates a plan
type PlanGeneratedEvent struct {
	BaseEvent
	GoalID      string `json:"goal_id"`
	PlanContent string `json:"plan_content"`
	Tasks       []Task `json:"tasks,omitempty"` // Proposed task breakdown
}

// PlanApprovedEvent is emitted when user approves a plan
type PlanApprovedEvent struct {
	BaseEvent
	GoalID string `json:"goal_id"`
}

// PlanRejectedEvent is emitted when user rejects a plan
type PlanRejectedEvent struct {
	BaseEvent
	GoalID   string `json:"goal_id"`
	Feedback string `json:"feedback,omitempty"` // User feedback for replanning
}

// ModeTransitionEvent is emitted when runtime mode changes
type ModeTransitionEvent struct {
	BaseEvent
	FromMode RuntimeMode `json:"from_mode"`
	ToMode   RuntimeMode `json:"to_mode"`
	Reason   string      `json:"reason"`
}

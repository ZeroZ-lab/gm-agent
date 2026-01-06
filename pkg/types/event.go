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

// LLMResponseEvent
type LLMResponseEvent struct {
	BaseEvent
	Model     string     `json:"model"`
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
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

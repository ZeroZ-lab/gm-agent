package types

import "time"

// State represents the complete system state snapshot.
// All Runtime decisions are based on this state.
type State struct {
	// Version control
	Version   int64     `json:"version"`    // State version, incremented on each update
	UpdatedAt time.Time `json:"updated_at"` // Last update time

	// System Configuration
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Runtime Mode Control
	Mode        RuntimeMode `json:"mode,omitempty"`         // Current runtime mode (planning/executing)
	PlanContent string      `json:"plan_content,omitempty"` // Generated plan during planning mode

	// Goal Management
	Goals []Goal `json:"goals"` // List of goals (sorted by priority)

	// Task Management
	Tasks map[string]*Task `json:"tasks"` // Task table: task_id -> Task

	// Artifact Management
	Artifacts map[string]*Artifact `json:"artifacts"` // Artifact table: artifact_id -> Artifact

	// Resource Locks
	Locks map[string]*Lock `json:"locks"` // Lock table: resource_path -> Lock

	// Context Window
	Context *ContextWindow `json:"context"` // LLM Context
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		Version:   0,
		UpdatedAt: time.Now(),
		Goals:     make([]Goal, 0),
		Tasks:     make(map[string]*Task),
		Artifacts: make(map[string]*Artifact),
		Locks:     make(map[string]*Lock),
		Context: &ContextWindow{
			Messages: make([]Message, 0),
		},
	}
}

// Clone creates a deep copy of the state (for checkpointing and reducer immutability)
func (s *State) Clone() *State {
	if s == nil {
		return nil
	}

	newState := &State{
		Version:      s.Version,
		UpdatedAt:    s.UpdatedAt,
		SystemPrompt: s.SystemPrompt,
		Mode:         s.Mode,
		PlanContent:  s.PlanContent,
		Goals:        make([]Goal, len(s.Goals)),
		Tasks:        make(map[string]*Task, len(s.Tasks)),
		Artifacts:    make(map[string]*Artifact, len(s.Artifacts)),
		Locks:        make(map[string]*Lock, len(s.Locks)),
	}

	// Deep copy Goals
	for i, g := range s.Goals {
		newState.Goals[i] = g.Clone()
	}

	// Deep copy Tasks map
	for k, v := range s.Tasks {
		if v != nil {
			newState.Tasks[k] = v.Clone()
		}
	}

	// Deep copy Artifacts map
	for k, v := range s.Artifacts {
		if v != nil {
			newState.Artifacts[k] = v.Clone()
		}
	}

	// Deep copy Locks map
	for k, v := range s.Locks {
		if v != nil {
			lockCopy := *v
			newState.Locks[k] = &lockCopy
		}
	}

	// Deep copy Context
	if s.Context != nil {
		newState.Context = s.Context.Clone()
	}

	return newState
}

// Clone creates a deep copy of the Goal
func (g Goal) Clone() Goal {
	clone := g
	if g.Deadline != nil {
		d := *g.Deadline
		clone.Deadline = &d
	}
	return clone
}

// Clone creates a deep copy of the Task
func (t *Task) Clone() *Task {
	if t == nil {
		return nil
	}
	clone := *t

	// Deep copy Inputs map
	if t.Inputs != nil {
		clone.Inputs = make(map[string]any, len(t.Inputs))
		for k, v := range t.Inputs {
			clone.Inputs[k] = v
		}
	}

	// Deep copy Result
	if t.Result != nil {
		resultCopy := *t.Result
		if t.Result.Error != nil {
			errCopy := *t.Result.Error
			resultCopy.Error = &errCopy
		}
		if t.Result.Artifacts != nil {
			resultCopy.Artifacts = make([]string, len(t.Result.Artifacts))
			copy(resultCopy.Artifacts, t.Result.Artifacts)
		}
		clone.Result = &resultCopy
	}

	// Deep copy pointer fields
	if t.StartedAt != nil {
		s := *t.StartedAt
		clone.StartedAt = &s
	}
	if t.EndedAt != nil {
		e := *t.EndedAt
		clone.EndedAt = &e
	}

	return &clone
}

// Clone creates a deep copy of the Artifact
func (a *Artifact) Clone() *Artifact {
	if a == nil {
		return nil
	}
	clone := *a

	// Deep copy Content slice
	if a.Content != nil {
		clone.Content = make([]byte, len(a.Content))
		copy(clone.Content, a.Content)
	}

	// Deep copy Metadata map
	if a.Metadata != nil {
		clone.Metadata = make(map[string]string, len(a.Metadata))
		for k, v := range a.Metadata {
			clone.Metadata[k] = v
		}
	}

	return &clone
}

// Clone creates a deep copy of the ContextWindow
func (c *ContextWindow) Clone() *ContextWindow {
	if c == nil {
		return nil
	}
	clone := &ContextWindow{
		TotalTokens:     c.TotalTokens,
		MaxTokens:       c.MaxTokens,
		ReserveOutput:   c.ReserveOutput,
		CompactionCount: c.CompactionCount,
		Messages:        make([]Message, len(c.Messages)),
	}

	// Deep copy Messages
	for i, m := range c.Messages {
		clone.Messages[i] = m.Clone()
	}

	// Deep copy pointer
	if c.LastCompactionAt != nil {
		t := *c.LastCompactionAt
		clone.LastCompactionAt = &t
	}

	return clone
}

// Clone creates a deep copy of the Message
func (m Message) Clone() Message {
	clone := m

	// Deep copy ToolCalls slice
	if m.ToolCalls != nil {
		clone.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		copy(clone.ToolCalls, m.ToolCalls)
	}

	return clone
}

// Goal represents an objective to be achieved
type Goal struct {
	ID          string     `json:"id"`
	Type        GoalType   `json:"type"`
	Description string     `json:"description"`
	Priority    int        `json:"priority"` // 0 is highest
	Status      GoalStatus `json:"status"`

	// Source
	SourceEventID string `json:"source_event_id"`

	// Constraints
	Deadline *time.Time `json:"deadline,omitempty"`
	MaxSteps int        `json:"max_steps"`

	// Progress
	StepsUsed int `json:"steps_used"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type GoalType string

const (
	GoalTypeUserRequest GoalType = "user_request"
	GoalTypeSubTask     GoalType = "sub_task"
	GoalTypeRecovery    GoalType = "recovery"
)

type GoalStatus string

const (
	GoalStatusPending    GoalStatus = "pending"
	GoalStatusInProgress GoalStatus = "in_progress"
	GoalStatusCompleted  GoalStatus = "completed"
	GoalStatusFailed     GoalStatus = "failed"
	GoalStatusCancelled  GoalStatus = "cancelled"
)

// Task represents an execution unit
type Task struct {
	ID       string `json:"id"`
	GoalID   string `json:"goal_id"`
	ParentID string `json:"parent_id"`

	Type        string         `json:"type"`
	Description string         `json:"description"`
	Inputs      map[string]any `json:"inputs"`

	Status TaskStatus `json:"status"`

	// Constraints
	Constraints TaskConstraints `json:"constraints"`

	// Result
	Result *TaskResult `json:"result,omitempty"`

	// Timestamps
	CreatedAt time.Time  `json:"created_at"`
	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusBlocked   TaskStatus = "blocked"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskConstraints struct {
	MaxSteps     int           `json:"max_steps"`
	MaxTokens    int           `json:"max_tokens"`
	Timeout      time.Duration `json:"timeout"`
	AllowedTools []string      `json:"allowed_tools"`
	PolicyLevel  string        `json:"policy_level"`
}

type TaskResult struct {
	Status    string     `json:"status"`
	Summary   string     `json:"summary"`
	Artifacts []string   `json:"artifacts"` // Artifact IDs
	Error     *TaskError `json:"error,omitempty"`
	Stats     TaskStats  `json:"stats"`
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type TaskStats struct {
	Steps      int           `json:"steps"`
	TokensUsed int           `json:"tokens_used"`
	ToolCalls  int           `json:"tool_calls"`
	Duration   time.Duration `json:"duration"`
}

// ContextWindow manages LLM conversation context
type ContextWindow struct {
	Messages []Message `json:"messages"`

	TotalTokens int `json:"total_tokens"`

	// Configuration
	MaxTokens     int `json:"max_tokens"`
	ReserveOutput int `json:"reserve_output"`

	// Compaction State
	LastCompactionAt *time.Time `json:"last_compaction_at,omitempty"`
	CompactionCount  int        `json:"compaction_count"`
}

type Message struct {
	ID      string `json:"id"`
	Role    string `json:"role"` // system/user/assistant/tool
	Content string `json:"content"`

	// Assistant: Tool Calls
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`

	// Tool: Result
	ToolCallID string `json:"tool_call_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"` // Required for Gemini

	TokenCount int       `json:"token_count"`
	Timestamp  time.Time `json:"timestamp"`
}

// Lock represents a resource lock
type Lock struct {
	Resource   string    `json:"resource"`
	Owner      string    `json:"owner"` // task_id or goal_id
	Type       LockType  `json:"type"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

type LockType string

const (
	LockTypeRead  LockType = "read"
	LockTypeWrite LockType = "write"
)

// Artifact represents a produced file or data
type Artifact struct {
	ID   string `json:"id"`
	Type string `json:"type"` // file/json/text/image
	Name string `json:"name"`

	// Content (one of)
	Path    string `json:"path,omitempty"`
	Content []byte `json:"content,omitempty"`

	// Metadata
	Metadata map[string]string `json:"metadata"`
	Size     int64             `json:"size"`

	// Relations
	TaskID string `json:"task_id,omitempty"`
	GoalID string `json:"goal_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

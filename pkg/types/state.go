package types

import "time"

// State represents the complete system state snapshot.
// All Runtime decisions are based on this state.
type State struct {
	// Version control
	Version   int64     `json:"version"`    // State version, incremented on each update
	UpdatedAt time.Time `json:"updated_at"` // Last update time

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

// Clone creates a deep copy of the state (for checkpointing)
// TODO: Implement deep copy logic
func (s *State) Clone() *State {
	// Placeholder for now, simplistic shallow copy for non-map/slice fields
	newState := &State{
		Version:   s.Version,
		UpdatedAt: s.UpdatedAt,
		Goals:     make([]Goal, len(s.Goals)),
		Tasks:     make(map[string]*Task, len(s.Tasks)),
		Artifacts: make(map[string]*Artifact, len(s.Artifacts)),
		Locks:     make(map[string]*Lock, len(s.Locks)),
	}
	copy(newState.Goals, s.Goals)
	// Deep copy maps and pointers... this needs full implementation
	// For MVP compilation, we return a basic structure.
	// In production, this must deeply copy all pointers.
	return newState
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

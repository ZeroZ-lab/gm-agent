package types

import (
	"github.com/oklog/ulid/v2"
)

// RuntimeMode represents the current operational mode of the runtime
type RuntimeMode string

const (
	ModePlanning  RuntimeMode = "planning"  // Read-only exploration and planning
	ModeExecuting RuntimeMode = "executing" // Full tool access
)

// ErrorSeverity defines the severity of an error for classification
type ErrorSeverity string

const (
	SeverityRetryable   ErrorSeverity = "retryable"
	SeverityRecoverable ErrorSeverity = "recoverable"
	SeverityFatal       ErrorSeverity = "fatal"
)

// Semantic defines the intent of a user message
type Semantic string

const (
	SemanticAppend  Semantic = "append"  // Append to current goal/task
	SemanticFork    Semantic = "fork"    // Create a new independent goal
	SemanticPreempt Semantic = "preempt" // Interrupt current work
	SemanticCancel  Semantic = "cancel"  // Cancel current work
)

// JSONSchema represents a JSON Schema definition
type JSONSchema map[string]any

// ID Generation Helpers

func GenerateID(prefix string) string {
	return prefix + "_" + ulid.Make().String()
}

func GenerateEventID() string   { return GenerateID("evt") }
func GenerateCommandID() string { return GenerateID("cmd") }
func GenerateTaskID() string    { return GenerateID("tsk") }
func GenerateGoalID() string    { return GenerateID("gol") }
func GeneratePatchID() string   { return GenerateID("pch") }

// Helper for random entropy in ULID if needed, though ulid.Make() uses rand.Reader by default now?
// Actually ulid.Make() returns a ULID. The default global usage might need seeding or it uses crypto/rand.
// As of v2, ulid.Make() returns a ULID with current time and entropy from crypto/rand.Reader.
// Ensure we handle potential errors if we were being very strict, but Make() typically panics if entropy fails.

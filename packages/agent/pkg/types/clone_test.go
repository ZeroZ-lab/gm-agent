package types

import (
	"testing"
	"time"
)

func TestStateCloneNil(t *testing.T) {
	var s *State
	clone := s.Clone()
	if clone != nil {
		t.Fatalf("expected nil clone for nil state")
	}
}

func TestStateCloneDeepCopy(t *testing.T) {
	now := time.Now()
	deadline := now.Add(time.Hour)

	original := &State{
		Version:      10,
		UpdatedAt:    now,
		SystemPrompt: "test prompt",
		Goals: []Goal{
			{ID: "g1", Description: "goal 1", Status: GoalStatusPending, Deadline: &deadline},
			{ID: "g2", Description: "goal 2", Status: GoalStatusCompleted},
		},
		Tasks: map[string]*Task{
			"t1": {
				ID:     "t1",
				GoalID: "g1",
				Inputs: map[string]any{"key": "value"},
				Result: &TaskResult{
					Status:    "done",
					Artifacts: []string{"art1"},
					Error:     &TaskError{Code: "E001", Message: "test error"},
				},
			},
		},
		Artifacts: map[string]*Artifact{
			"art1": {
				ID:       "art1",
				Type:     "file",
				Name:     "test.txt",
				Content:  []byte("hello"),
				Metadata: map[string]string{"key": "value"},
			},
		},
		Locks: map[string]*Lock{
			"resource1": {Resource: "resource1", Owner: "t1", Type: LockTypeWrite},
		},
		Context: &ContextWindow{
			Messages: []Message{
				{ID: "m1", Role: "user", Content: "hello", ToolCalls: []ToolCall{{ID: "tc1", Name: "test"}}},
			},
			TotalTokens:     100,
			MaxTokens:       1000,
			LastCompactionAt: &now,
		},
	}

	clone := original.Clone()

	// Verify it's a different instance
	if clone == original {
		t.Fatalf("clone should be a different instance")
	}

	// Verify values are copied correctly
	if clone.Version != original.Version {
		t.Errorf("Version mismatch: got %d, want %d", clone.Version, original.Version)
	}
	if clone.SystemPrompt != original.SystemPrompt {
		t.Errorf("SystemPrompt mismatch")
	}

	// Verify Goals are deep copied
	if len(clone.Goals) != len(original.Goals) {
		t.Fatalf("Goals length mismatch")
	}
	if clone.Goals[0].Deadline == original.Goals[0].Deadline {
		t.Errorf("Goal Deadline should be a different pointer")
	}
	if *clone.Goals[0].Deadline != *original.Goals[0].Deadline {
		t.Errorf("Goal Deadline value mismatch")
	}

	// Verify Tasks are deep copied
	if clone.Tasks["t1"] == original.Tasks["t1"] {
		t.Errorf("Task should be a different pointer")
	}
	if clone.Tasks["t1"].Inputs["key"] != "value" {
		t.Errorf("Task Inputs mismatch")
	}
	// Modify clone's inputs, original should not change
	clone.Tasks["t1"].Inputs["key"] = "modified"
	if original.Tasks["t1"].Inputs["key"] == "modified" {
		t.Errorf("Modifying clone should not affect original")
	}

	// Verify Artifacts are deep copied
	if clone.Artifacts["art1"] == original.Artifacts["art1"] {
		t.Errorf("Artifact should be a different pointer")
	}
	if string(clone.Artifacts["art1"].Content) != "hello" {
		t.Errorf("Artifact Content mismatch")
	}
	// Modify clone's content
	clone.Artifacts["art1"].Content[0] = 'X'
	if original.Artifacts["art1"].Content[0] == 'X' {
		t.Errorf("Modifying clone content should not affect original")
	}

	// Verify Locks are deep copied
	if clone.Locks["resource1"] == original.Locks["resource1"] {
		t.Errorf("Lock should be a different pointer")
	}

	// Verify Context is deep copied
	if clone.Context == original.Context {
		t.Errorf("Context should be a different pointer")
	}
	if clone.Context.LastCompactionAt == original.Context.LastCompactionAt {
		t.Errorf("Context LastCompactionAt should be a different pointer")
	}

	// Verify Messages are deep copied
	if len(clone.Context.Messages) != len(original.Context.Messages) {
		t.Fatalf("Messages length mismatch")
	}
	// Modify clone's message
	clone.Context.Messages[0].Content = "modified"
	if original.Context.Messages[0].Content == "modified" {
		t.Errorf("Modifying clone message should not affect original")
	}

	// Verify ToolCalls are deep copied
	clone.Context.Messages[0].ToolCalls[0].Name = "modified"
	if original.Context.Messages[0].ToolCalls[0].Name == "modified" {
		t.Errorf("Modifying clone tool calls should not affect original")
	}
}

func TestGoalClone(t *testing.T) {
	deadline := time.Now()
	original := Goal{
		ID:       "g1",
		Deadline: &deadline,
		Status:   GoalStatusPending,
	}

	clone := original.Clone()

	// Deadline should be a different pointer
	if clone.Deadline == original.Deadline {
		t.Errorf("Deadline should be a different pointer")
	}
	if *clone.Deadline != *original.Deadline {
		t.Errorf("Deadline value mismatch")
	}

	// Modify clone's deadline
	newTime := time.Now().Add(time.Hour)
	clone.Deadline = &newTime
	if *original.Deadline == newTime {
		t.Errorf("Modifying clone should not affect original")
	}
}

func TestGoalCloneNilDeadline(t *testing.T) {
	original := Goal{ID: "g1", Deadline: nil}
	clone := original.Clone()
	if clone.Deadline != nil {
		t.Errorf("expected nil deadline")
	}
}

func TestTaskCloneNil(t *testing.T) {
	var task *Task
	clone := task.Clone()
	if clone != nil {
		t.Fatalf("expected nil clone for nil task")
	}
}

func TestTaskCloneDeepCopy(t *testing.T) {
	startedAt := time.Now()
	endedAt := startedAt.Add(time.Minute)

	original := &Task{
		ID:        "t1",
		Inputs:    map[string]any{"key": "value", "num": 42},
		StartedAt: &startedAt,
		EndedAt:   &endedAt,
		Result: &TaskResult{
			Status:    "completed",
			Artifacts: []string{"a1", "a2"},
			Error:     &TaskError{Code: "E001", Message: "error"},
		},
	}

	clone := original.Clone()

	if clone == original {
		t.Fatalf("clone should be a different instance")
	}

	// Verify Inputs deep copy
	clone.Inputs["key"] = "modified"
	if original.Inputs["key"] == "modified" {
		t.Errorf("Modifying clone inputs should not affect original")
	}

	// Verify time pointers are different
	if clone.StartedAt == original.StartedAt {
		t.Errorf("StartedAt should be a different pointer")
	}
	if clone.EndedAt == original.EndedAt {
		t.Errorf("EndedAt should be a different pointer")
	}

	// Verify Result deep copy
	if clone.Result == original.Result {
		t.Errorf("Result should be a different pointer")
	}
	if clone.Result.Error == original.Result.Error {
		t.Errorf("Result.Error should be a different pointer")
	}

	// Verify Artifacts slice deep copy
	clone.Result.Artifacts[0] = "modified"
	if original.Result.Artifacts[0] == "modified" {
		t.Errorf("Modifying clone artifacts should not affect original")
	}
}

func TestArtifactCloneNil(t *testing.T) {
	var artifact *Artifact
	clone := artifact.Clone()
	if clone != nil {
		t.Fatalf("expected nil clone for nil artifact")
	}
}

func TestArtifactCloneDeepCopy(t *testing.T) {
	original := &Artifact{
		ID:       "art1",
		Content:  []byte("test content"),
		Metadata: map[string]string{"key": "value"},
	}

	clone := original.Clone()

	if clone == original {
		t.Fatalf("clone should be a different instance")
	}

	// Verify Content deep copy
	clone.Content[0] = 'X'
	if original.Content[0] == 'X' {
		t.Errorf("Modifying clone content should not affect original")
	}

	// Verify Metadata deep copy
	clone.Metadata["key"] = "modified"
	if original.Metadata["key"] == "modified" {
		t.Errorf("Modifying clone metadata should not affect original")
	}
}

func TestContextWindowCloneNil(t *testing.T) {
	var ctx *ContextWindow
	clone := ctx.Clone()
	if clone != nil {
		t.Fatalf("expected nil clone for nil context")
	}
}

func TestContextWindowCloneDeepCopy(t *testing.T) {
	compactionTime := time.Now()
	original := &ContextWindow{
		Messages: []Message{
			{ID: "m1", Content: "hello", ToolCalls: []ToolCall{{ID: "tc1"}}},
		},
		TotalTokens:      100,
		LastCompactionAt: &compactionTime,
	}

	clone := original.Clone()

	if clone == original {
		t.Fatalf("clone should be a different instance")
	}

	// Verify LastCompactionAt deep copy
	if clone.LastCompactionAt == original.LastCompactionAt {
		t.Errorf("LastCompactionAt should be a different pointer")
	}

	// Verify Messages deep copy
	clone.Messages[0].Content = "modified"
	if original.Messages[0].Content == "modified" {
		t.Errorf("Modifying clone message should not affect original")
	}
}

func TestMessageClone(t *testing.T) {
	original := Message{
		ID:        "m1",
		Content:   "test",
		ToolCalls: []ToolCall{{ID: "tc1", Name: "tool1"}, {ID: "tc2", Name: "tool2"}},
	}

	clone := original.Clone()

	// Verify ToolCalls deep copy
	clone.ToolCalls[0].Name = "modified"
	if original.ToolCalls[0].Name == "modified" {
		t.Errorf("Modifying clone tool calls should not affect original")
	}

	// Verify slice independence
	clone.ToolCalls = append(clone.ToolCalls, ToolCall{ID: "tc3"})
	if len(original.ToolCalls) != 2 {
		t.Errorf("Appending to clone should not affect original")
	}
}

func TestMessageCloneNilToolCalls(t *testing.T) {
	original := Message{ID: "m1", ToolCalls: nil}
	clone := original.Clone()
	if clone.ToolCalls != nil {
		t.Errorf("expected nil tool calls")
	}
}

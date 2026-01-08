package dto

import "time"

// CheckpointResponse represents a checkpoint summary
type CheckpointResponse struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	StateVersion  int64     `json:"state_version"`
	LastEventID   string    `json:"last_event_id,omitempty"`
	Description   string    `json:"description,omitempty"`
	MessageCount  int       `json:"message_count"`
}

// CheckpointListResponse contains list of checkpoints
type CheckpointListResponse struct {
	Checkpoints []CheckpointResponse `json:"checkpoints"`
}

// RewindRequest specifies what to rewind
type RewindRequest struct {
	CheckpointID string `json:"checkpoint_id" binding:"required"`
	RewindCode   bool   `json:"rewind_code"`   // Rewind file changes
	RewindConversation bool `json:"rewind_conversation"` // Rewind conversation history
}

// RewindResponse contains the result of rewind operation
type RewindResponse struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	RestoredCheckpoint CheckpointResponse `json:"restored_checkpoint"`
}

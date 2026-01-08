package patch

import (
	"sync"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// FileChangeTracker tracks file modifications for checkpoint-based rewind
type FileChangeTracker interface {
	// Record adds a file change to the current tracking window
	Record(change types.FileChange)

	// Flush returns all pending changes and clears the tracker
	// This should be called when creating a checkpoint
	Flush() []types.FileChange

	// GetPending returns pending changes without clearing
	GetPending() []types.FileChange
}

// NewFileChangeTracker creates a new tracker instance
func NewFileChangeTracker() FileChangeTracker {
	return &fileChangeTracker{
		changes: make([]types.FileChange, 0),
	}
}

type fileChangeTracker struct {
	mu      sync.Mutex
	changes []types.FileChange
}

func (t *fileChangeTracker) Record(change types.FileChange) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.changes = append(t.changes, change)
}

func (t *fileChangeTracker) Flush() []types.FileChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := t.changes
	t.changes = make([]types.FileChange, 0)
	return result
}

func (t *fileChangeTracker) GetPending() []types.FileChange {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]types.FileChange, len(t.changes))
	copy(result, t.changes)
	return result
}

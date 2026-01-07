package store

import (
	"context"
	"errors"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

var (
	ErrNoCheckpoint = errors.New("no checkpoint found")
	ErrNotFound     = errors.New("not found")
)

// Store defines the persistence layer contract
type Store interface {
	// Lifecycle
	Open(ctx context.Context) error
	Close() error

	// Event Operations
	AppendEvent(ctx context.Context, event types.Event) error
	AppendEvents(ctx context.Context, events []types.Event) error
	GetEvent(ctx context.Context, id string) (types.Event, error)
	GetEventsSince(ctx context.Context, afterEventID string) ([]types.Event, error)
	IterEvents(ctx context.Context, fn func(types.Event) error) error

	// State Operations
	SaveState(ctx context.Context, state *types.State) error
	LoadState(ctx context.Context, version int64) (*types.State, error)
	LoadLatestState(ctx context.Context) (*types.State, error)

	// Checkpoint Operations
	SaveCheckpoint(ctx context.Context, cp *types.Checkpoint) error
	LoadCheckpoint(ctx context.Context, id string) (*types.Checkpoint, error)
	LoadLatestCheckpoint(ctx context.Context) (*types.Checkpoint, error)

	// Artifact Operations
	SaveArtifact(ctx context.Context, artifact *types.Artifact) error
	GetArtifact(ctx context.Context, id string) (*types.Artifact, error)
	ListArtifacts(ctx context.Context, filter ArtifactFilter) ([]types.Artifact, error)
	DeleteArtifact(ctx context.Context, id string) error

	// Permission Operations
	AddPermissionRule(ctx context.Context, rule types.PermissionRule) error
	GetPermissionRules(ctx context.Context) ([]types.PermissionRule, error)
}

type ArtifactFilter struct {
	TaskID string
	GoalID string
	Type   string
}

// Transactional extends Store with transaction support
type Transactional interface {
	Begin(ctx context.Context) (Transaction, error)
}

type Transaction interface {
	Store
	Commit() error
	Rollback() error
}

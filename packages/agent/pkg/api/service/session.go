package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/api/dto"
	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/runtime/permission"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// ErrSessionNotFound is returned when a session is not found.
var ErrSessionNotFound = errors.New("session not found")

// RuntimeRunner is the minimal runtime contract the service relies on.
type RuntimeRunner interface {
	Ingest(ctx context.Context, event types.Event) error
	Run(ctx context.Context) error
	GetState() *types.State
}

// SessionResources contains runtime dependencies for a session.
type SessionResources struct {
	Runtime     RuntimeRunner
	Permissions *permission.Manager
	Store       store.Store
	PatchEngine patch.Engine // For Code Rewind support
	Ctx         context.Context
	Cancel      context.CancelFunc
}

// SessionFactory creates per-session runtime resources.
type SessionFactory func(sessionID string) (*SessionResources, error)

// Session represents an active session.
type Session struct {
	ID        string
	Status    string
	CreatedAt time.Time
	LastError string
	Resources *SessionResources

	mu sync.Mutex
}

// SessionService manages sessions.
type SessionService struct {
	factory  SessionFactory
	sessions sync.Map // map[string]*Session
	log      *slog.Logger
}

// NewSessionService creates a new SessionService.
func NewSessionService(factory SessionFactory, log *slog.Logger) *SessionService {
	return &SessionService{
		factory: factory,
		log:     log,
	}
}

// Create creates a new session with the given prompt.
// If prompt is empty, the session is created but no LLM call is made until a message is sent.
func (s *SessionService) Create(ctx context.Context, prompt string, systemPrompt string, priority int) (*Session, error) {
	id := types.GenerateID("ses")
	resources, err := s.factory(id)
	if err != nil {
		s.log.Error("failed to create session resources", "error", err)
		return nil, err
	}

	session := &Session{
		ID:        id,
		Status:    "idle", // idle until first message
		CreatedAt: time.Now(),
		Resources: resources,
	}

	s.sessions.Store(id, session)

	// Ingest System Prompt if present
	if systemPrompt != "" {
		sysEvent := &types.SystemPromptEvent{
			BaseEvent: types.NewBaseEvent("system_prompt", "user", id),
			Prompt:    systemPrompt,
		}
		if err := resources.Runtime.Ingest(resources.Ctx, sysEvent); err != nil {
			s.log.Error("failed to ingest system prompt", "error", err)
			return nil, err
		}
	}

	// Only ingest prompt and start runtime if prompt is not empty
	if prompt != "" {
		session.Status = "running"

		// Ingest prompt
		event := &types.UserMessageEvent{
			BaseEvent: types.NewBaseEvent("user_request", "user", id),
			Content:   prompt,
			Priority:  priority,
		}
		if err := resources.Runtime.Ingest(resources.Ctx, event); err != nil {
			s.log.Error("failed to ingest prompt", "error", err)
			return nil, err
		}

		// Start runtime in background
		go s.runSession(session)
	}

	return session, nil
}

// Get returns a session by ID.
func (s *SessionService) Get(id string) (*Session, error) {
	val, ok := s.sessions.Load(id)
	if !ok {
		return nil, ErrSessionNotFound
	}
	return val.(*Session), nil
}

// List returns all sessions.
func (s *SessionService) List() []*Session {
	var result []*Session
	s.sessions.Range(func(_, v any) bool {
		result = append(result, v.(*Session))
		return true
	})
	return result
}

// Delete deletes a session.
func (s *SessionService) Delete(id string) error {
	val, ok := s.sessions.Load(id)
	if !ok {
		return ErrSessionNotFound
	}

	session := val.(*Session)
	session.Resources.Cancel()
	s.sessions.Delete(id)
	return nil
}

// Cancel cancels a running session.
func (s *SessionService) Cancel(id string) error {
	val, ok := s.sessions.Load(id)
	if !ok {
		return ErrSessionNotFound
	}

	session := val.(*Session)
	session.Resources.Cancel()

	session.mu.Lock()
	session.Status = "cancelled"
	session.mu.Unlock()

	return nil
}

// Message sends a user message to a session.
func (s *SessionService) Message(ctx context.Context, id string, content string, semantic string) (*Session, error) {
	val, ok := s.sessions.Load(id)
	if !ok {
		return nil, ErrSessionNotFound
	}
	session := val.(*Session)

	priority := 0
	if types.Semantic(semantic) == types.SemanticPreempt {
		priority = 100
	}

	event := &types.UserMessageEvent{
		BaseEvent: types.NewBaseEvent("user_request", "user", id),
		Content:   content,
		Priority:  priority,
		Semantic:  types.Semantic(semantic),
	}

	// We use the session's context for ingestion to ensure it respects session lifecycle
	if err := session.Resources.Runtime.Ingest(session.Resources.Ctx, event); err != nil {
		s.log.Error("failed to ingest message", "error", err)
		return nil, err
	}

	// If session was idle or completed, start/restart the runtime
	session.mu.Lock()
	if session.Status == "idle" || session.Status == "completed" {
		session.Status = "running"
		session.mu.Unlock()
		go s.runSession(session)
	} else {
		session.mu.Unlock()
	}

	return session, nil
}

// GetStatus returns the status of a session (thread-safe).
func (sess *Session) GetStatus() (status string, lastError string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	return sess.Status, sess.LastError
}

// runSession runs the session runtime in background.
func (s *SessionService) runSession(sess *Session) {
	defer sess.Resources.Store.Close()

	err := sess.Resources.Runtime.Run(sess.Resources.Ctx)

	sess.mu.Lock()
	defer sess.mu.Unlock()

	if err != nil {
		if errors.Is(err, context.Canceled) {
			sess.Status = "cancelled"
			return
		}
		sess.Status = "error"
		sess.LastError = err.Error()
		return
	}
	sess.Status = "completed"
}

// ListArtifacts returns all artifacts for a session.
func (s *SessionService) ListArtifacts(id string) ([]*types.Artifact, error) {
	session, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	state := session.Resources.Runtime.GetState()
	if state == nil {
		return nil, errors.New("state not available")
	}

	var list []*types.Artifact
	for _, art := range state.Artifacts {
		list = append(list, art)
	}
	return list, nil
}

// GetArtifact returns a specific artifact.
func (s *SessionService) GetArtifact(sessionID string, artifactID string) (*types.Artifact, error) {
	session, err := s.Get(sessionID)
	if err != nil {
		return nil, err
	}

	state := session.Resources.Runtime.GetState()
	if state == nil {
		return nil, errors.New("state not available")
	}

	art, ok := state.Artifacts[artifactID]
	if !ok {
		return nil, errors.New("artifact not found")
	}

	return art, nil
}

// RespondPermission handles a permission response
func (s *SessionService) RespondPermission(id string, requestID string, approved bool, always bool) error {
	val, ok := s.sessions.Load(id)
	if !ok {
		return ErrSessionNotFound
	}
	session := val.(*Session)

	// If manager is not initialized (e.g. tests), ignore or error
	if session.Resources.Permissions == nil {
		return errors.New("permission manager not available")
	}

	return session.Resources.Permissions.Respond(requestID, approved, always)
}

// ListCheckpoints returns all checkpoints for a session
func (s *SessionService) ListCheckpoints(ctx context.Context, id string) (*dto.CheckpointListResponse, error) {
	session, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	checkpoints, err := session.Resources.Store.ListCheckpoints(ctx)
	if err != nil {
		return nil, err
	}

	// Convert to response DTOs
	response := &dto.CheckpointListResponse{
		Checkpoints: make([]dto.CheckpointResponse, 0, len(checkpoints)),
	}

	for _, cp := range checkpoints {
		msgCount := 0
		if cp.State != nil && cp.State.Context != nil {
			msgCount = len(cp.State.Context.Messages)
		}
		response.Checkpoints = append(response.Checkpoints, dto.CheckpointResponse{
			ID:           cp.ID,
			Timestamp:    cp.Timestamp,
			StateVersion: cp.State.Version,
			LastEventID:  cp.LastEventID,
			MessageCount: msgCount,
		})
	}

	return response, nil
}

// Rewind rewinds a session to a previous checkpoint
func (s *SessionService) Rewind(ctx context.Context, id string, req dto.RewindRequest) (*dto.RewindResponse, error) {
	session, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	// Load the checkpoint
	checkpoint, err := session.Resources.Store.LoadCheckpoint(ctx, req.CheckpointID)
	if err != nil {
		return &dto.RewindResponse{
			Success: false,
			Message: "Checkpoint not found: " + err.Error(),
		}, nil
	}

	// Code rewind: restore files from backups
	if req.RewindCode {
		if session.Resources.PatchEngine == nil {
			return &dto.RewindResponse{
				Success: false,
				Message: "Code rewind not available: patch engine not configured",
			}, nil
		}

		// Get all checkpoints after the target checkpoint to find file changes to revert
		allCheckpoints, err := session.Resources.Store.ListCheckpoints(ctx)
		if err != nil {
			return &dto.RewindResponse{
				Success: false,
				Message: "Failed to list checkpoints: " + err.Error(),
			}, nil
		}

		// Collect all file changes that happened AFTER the target checkpoint
		// We need to revert these in reverse chronological order
		var changesToRevert []types.FileChange
		foundTarget := false
		for _, cp := range allCheckpoints {
			if cp.ID == checkpoint.ID {
				foundTarget = true
				break
			}
			// This checkpoint is after our target, collect its file changes
			changesToRevert = append(changesToRevert, cp.FileChanges...)
		}

		if !foundTarget {
			return &dto.RewindResponse{
				Success: false,
				Message: "Target checkpoint not found in checkpoint list",
			}, nil
		}

		// Rollback each file change using patch engine
		var rollbackErrors []string
		for i := len(changesToRevert) - 1; i >= 0; i-- {
			change := changesToRevert[i]
			if err := session.Resources.PatchEngine.Rollback(ctx, change.PatchID); err != nil {
				rollbackErrors = append(rollbackErrors,
					"Failed to rollback "+change.FilePath+": "+err.Error())
			} else {
				s.log.Info("rolled back file change",
					"patch_id", change.PatchID,
					"file", change.FilePath)
			}
		}

		if len(rollbackErrors) > 0 {
			return &dto.RewindResponse{
				Success: false,
				Message: "Code rewind partially failed: " + rollbackErrors[0],
			}, nil
		}

		s.log.Info("code rewind completed",
			"session_id", id,
			"checkpoint_id", req.CheckpointID,
			"files_reverted", len(changesToRevert))
	}

	if req.RewindConversation {
		// Restore the checkpoint state
		if err := session.Resources.Store.SaveState(ctx, checkpoint.State); err != nil {
			return &dto.RewindResponse{
				Success: false,
				Message: "Failed to restore state: " + err.Error(),
			}, nil
		}

		s.log.Info("session rewound to checkpoint",
			"session_id", id,
			"checkpoint_id", req.CheckpointID,
			"state_version", checkpoint.State.Version)
	}

	msgCount := 0
	if checkpoint.State != nil && checkpoint.State.Context != nil {
		msgCount = len(checkpoint.State.Context.Messages)
	}

	message := "Successfully rewound"
	if req.RewindCode && req.RewindConversation {
		message = "Successfully rewound code and conversation"
	} else if req.RewindCode {
		message = "Successfully rewound code"
	} else if req.RewindConversation {
		message = "Successfully rewound conversation"
	}

	return &dto.RewindResponse{
		Success: true,
		Message: message,
		RestoredCheckpoint: dto.CheckpointResponse{
			ID:           checkpoint.ID,
			Timestamp:    checkpoint.Timestamp,
			StateVersion: checkpoint.State.Version,
			LastEventID:  checkpoint.LastEventID,
			MessageCount: msgCount,
		},
	}, nil
}

package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

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
func (s *SessionService) Create(ctx context.Context, prompt string, systemPrompt string, priority int) (*Session, error) {
	id := types.GenerateID("ses")
	resources, err := s.factory(id)
	if err != nil {
		s.log.Error("failed to create session resources", "error", err)
		return nil, err
	}

	session := &Session{
		ID:        id,
		Status:    "running",
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

	// If session was completed, restart the runtime to process the new message
	session.mu.Lock()
	if session.Status == "completed" {
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

package service

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// ErrSessionNotFound is returned when a session is not found.
var ErrSessionNotFound = errors.New("session not found")

// RuntimeRunner is the minimal runtime contract the service relies on.
type RuntimeRunner interface {
	Ingest(ctx context.Context, event types.Event) error
	Run(ctx context.Context) error
}

// SessionResources contains runtime dependencies for a session.
type SessionResources struct {
	Runtime RuntimeRunner
	Store   store.Store
	Ctx     context.Context
	Cancel  context.CancelFunc
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
func (s *SessionService) Create(ctx context.Context, prompt string, priority int) (*Session, error) {
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

package permission

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"
)

var (
	ErrRequestNotFound = errors.New("permission request not found")
	ErrTimeout         = errors.New("permission request timed out")
)

// Response represents the user's decision
type Response struct {
	Approved bool
	Always   bool
}

// Manager handles pending permission requests
type Manager struct {
	pending sync.Map // map[string]chan Response
	log     *slog.Logger
}

func NewManager(log *slog.Logger) *Manager {
	return &Manager{
		log: log,
	}
}

// Request registers a new request and returns a channel to wait on.
// The caller MUST ensure Cleanup is called (usually via defer).
func (m *Manager) Request(id string) <-chan Response {
	ch := make(chan Response, 1) // Buffered to prevent blocking the sender
	m.pending.Store(id, ch)
	return ch
}

// Respond sends a response to a pending request
func (m *Manager) Respond(id string, approved bool, always bool) error {
	val, ok := m.pending.Load(id)
	if !ok {
		return ErrRequestNotFound
	}

	ch := val.(chan Response)
	select {
	case ch <- Response{Approved: approved, Always: always}:
		return nil
	default:
		// Channel full or closed
		return errors.New("failed to send response: channel might be full")
	}
}

// WaitForResponse waits for a response with a timeout
func (m *Manager) WaitForResponse(ctx context.Context, id string, timeout time.Duration) (Response, error) {
	chRaw, ok := m.pending.Load(id)
	if !ok {
		return Response{}, ErrRequestNotFound
	}
	ch := chRaw.(chan Response)

	defer m.pending.Delete(id)

	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(timeout):
		return Response{}, ErrTimeout
	case <-ctx.Done():
		return Response{}, ctx.Err()
	}
}

package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestCreateSessionAndStatus(t *testing.T) {
	memStore := newMemoryStore()
	runtime := &stubRuntime{store: memStore, done: make(chan struct{})}
	factory := func(string) (*SessionResources, error) {
		ctx, cancel := context.WithCancel(context.Background())
		return &SessionResources{Runtime: runtime, Store: memStore, Ctx: ctx, Cancel: cancel}, nil
	}

	srv := NewServer(context.Background(), HTTPConfig{}, factory, nil)

	body := `{"prompt": "hello"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Engine.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("unexpected status: %d", w.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	sessionID, _ := resp["session_id"].(string)
	if sessionID == "" {
		t.Fatalf("session_id missing")
	}

	runtime.wait()

	statusReq, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID, nil)
	statusW := httptest.NewRecorder()
	srv.Engine.ServeHTTP(statusW, statusReq)

	if statusW.Code != http.StatusOK {
		t.Fatalf("status endpoint returned %d", statusW.Code)
	}

	if !runtime.started {
		t.Fatalf("runtime did not start")
	}
}

func TestEventsEndpoint(t *testing.T) {
	memStore := newMemoryStore()
	runtime := &stubRuntime{store: memStore, done: make(chan struct{})}
	factory := func(string) (*SessionResources, error) {
		ctx, cancel := context.WithCancel(context.Background())
		return &SessionResources{Runtime: runtime, Store: memStore, Ctx: ctx, Cancel: cancel}, nil
	}
	srv := NewServer(context.Background(), HTTPConfig{}, factory, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(`{"prompt": "event test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"].(string)

	eventsReq, _ := http.NewRequest(http.MethodGet, "/api/v1/sessions/"+sessionID+"/events", nil)
	eventsW := httptest.NewRecorder()
	srv.Engine.ServeHTTP(eventsW, eventsReq)

	if eventsW.Code != http.StatusOK {
		t.Fatalf("events endpoint returned %d", eventsW.Code)
	}

	var eventsResp map[string]any
	if err := json.Unmarshal(eventsW.Body.Bytes(), &eventsResp); err != nil {
		t.Fatalf("parse events: %v", err)
	}
	items, ok := eventsResp["events"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected events in response")
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	memStore := newMemoryStore()
	runtime := &stubRuntime{store: memStore}
	factory := func(string) (*SessionResources, error) {
		ctx, cancel := context.WithCancel(context.Background())
		return &SessionResources{Runtime: runtime, Store: memStore, Ctx: ctx, Cancel: cancel}, nil
	}
	srv := NewServer(context.Background(), HTTPConfig{APIKey: "secret"}, factory, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(`{"prompt": "secured"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}

	req.Header.Set("X-API-Key", "secret")
	w = httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 with api key, got %d", w.Code)
	}
}

func TestCancelSession(t *testing.T) {
	memStore := newMemoryStore()
	blocker := make(chan struct{})
	runtime := &stubRuntime{store: memStore, blockUntil: blocker, done: make(chan struct{})}
	factory := func(string) (*SessionResources, error) {
		ctx, cancel := context.WithCancel(context.Background())
		return &SessionResources{Runtime: runtime, Store: memStore, Ctx: ctx, Cancel: cancel}, nil
	}
	srv := NewServer(context.Background(), HTTPConfig{}, factory, nil)

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/sessions", strings.NewReader(`{"prompt": "wait"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Engine.ServeHTTP(w, req)

	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	sessionID := resp["session_id"].(string)

	cancelReq, _ := http.NewRequest(http.MethodPost, "/api/v1/sessions/"+sessionID+"/cancel", nil)
	cancelW := httptest.NewRecorder()
	srv.Engine.ServeHTTP(cancelW, cancelReq)

	if cancelW.Code != http.StatusOK {
		t.Fatalf("cancel returned %d", cancelW.Code)
	}

	runtime.waitDone()
	if runtime.lastErr == nil || runtime.lastErr != context.Canceled {
		t.Fatalf("expected runtime cancellation")
	}
}

type stubRuntime struct {
	store      store.Store
	started    bool
	lastErr    error
	blockUntil chan struct{}
	done       chan struct{}
	mu         sync.Mutex
}

func (s *stubRuntime) Ingest(ctx context.Context, event types.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.store != nil {
		_ = s.store.AppendEvent(ctx, event)
		_ = s.store.SaveState(ctx, types.NewState())
	}
	return nil
}

func (s *stubRuntime) Run(ctx context.Context) error {
	if s.done != nil {
		defer close(s.done)
	}
	s.mu.Lock()
	s.started = true
	s.mu.Unlock()

	if s.blockUntil != nil {
		select {
		case <-ctx.Done():
			s.lastErr = ctx.Err()
			return ctx.Err()
		case <-s.blockUntil:
		}
	}
	return s.lastErr
}

func (s *stubRuntime) wait() {
	for i := 0; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		s.mu.Lock()
		started := s.started
		s.mu.Unlock()
		if started {
			return
		}
	}
}

func (s *stubRuntime) waitDone() {
	if s.done == nil {
		return
	}
	select {
	case <-s.done:
	case <-time.After(100 * time.Millisecond):
	}
}

type memoryStore struct {
	events []types.Event
	state  *types.State
	mu     sync.RWMutex
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (m *memoryStore) Open(ctx context.Context) error { return nil }
func (m *memoryStore) Close() error                   { return nil }

func (m *memoryStore) AppendEvent(ctx context.Context, event types.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

func (m *memoryStore) AppendEvents(ctx context.Context, events []types.Event) error {
	for _, e := range events {
		_ = m.AppendEvent(ctx, e)
	}
	return nil
}

func (m *memoryStore) GetEvent(ctx context.Context, id string) (types.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.events {
		if e.EventID() == id {
			return e, nil
		}
	}
	return nil, store.ErrNotFound
}

func (m *memoryStore) GetEventsSince(ctx context.Context, afterEventID string) ([]types.Event, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if afterEventID == "" {
		return append([]types.Event{}, m.events...), nil
	}
	var result []types.Event
	seen := false
	for _, e := range m.events {
		if seen {
			result = append(result, e)
			continue
		}
		if e.EventID() == afterEventID {
			seen = true
		}
	}
	return result, nil
}

func (m *memoryStore) IterEvents(ctx context.Context, fn func(types.Event) error) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, e := range m.events {
		if err := fn(e); err != nil {
			return err
		}
	}
	return nil
}

func (m *memoryStore) SaveState(ctx context.Context, state *types.State) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
	return nil
}

func (m *memoryStore) LoadState(ctx context.Context, version int64) (*types.State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return nil, store.ErrNotFound
	}
	return m.state, nil
}

func (m *memoryStore) LoadLatestState(ctx context.Context) (*types.State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.state == nil {
		return nil, store.ErrNotFound
	}
	return m.state, nil
}

func (m *memoryStore) SaveCheckpoint(ctx context.Context, cp *types.Checkpoint) error { return nil }
func (m *memoryStore) LoadCheckpoint(ctx context.Context, id string) (*types.Checkpoint, error) {
	return nil, store.ErrNoCheckpoint
}
func (m *memoryStore) LoadLatestCheckpoint(ctx context.Context) (*types.Checkpoint, error) {
	return nil, store.ErrNoCheckpoint
}

func (m *memoryStore) SaveArtifact(ctx context.Context, artifact *types.Artifact) error { return nil }
func (m *memoryStore) GetArtifact(ctx context.Context, id string) (*types.Artifact, error) {
	return nil, store.ErrNotFound
}
func (m *memoryStore) ListArtifacts(ctx context.Context, filter store.ArtifactFilter) ([]types.Artifact, error) {
	return nil, nil
}
func (m *memoryStore) DeleteArtifact(ctx context.Context, id string) error { return nil }

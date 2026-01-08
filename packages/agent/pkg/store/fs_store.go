package store

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// FSStore implements Store using the local file system.
// NOTE: This implementation does NOT support transactions (crash consistency depends on individual atomic writes).
// Directory structure:
// workspace/
//
//	├── events.jsonl
//	├── state/
//	│   └── state.json
//	├── checkpoints/
//	│   └── {timestamp}_{id}.json
//	└── artifacts/
//	    └── {id}_{name}
type FSStore struct {
	rootDir string
	mu      sync.RWMutex // Global lock for simplified thread-safety (granularity could be improved)

	eventLogPath string
}

func NewFSStore(rootDir string) *FSStore {
	return &FSStore{
		rootDir:      rootDir,
		eventLogPath: filepath.Join(rootDir, "events.jsonl"),
	}
}

func (s *FSStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dirs := []string{
		s.rootDir,
		filepath.Join(s.rootDir, "state"),
		filepath.Join(s.rootDir, "checkpoints"),
		filepath.Join(s.rootDir, "artifacts"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}
	return nil
}

func (s *FSStore) Close() error {
	return nil
}

// --- Event Operations ---

// internal wrapper for JSONL serialization
type eventRecord struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Timestamp time.Time       `json:"timestamp"`
	Actor     string          `json:"actor"`
	Subject   string          `json:"subject"`
	Data      json.RawMessage `json:"data"`
}

func (s *FSStore) AppendEvent(ctx context.Context, event types.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.appendEventLocked(event)
}

func (s *FSStore) AppendEvents(ctx context.Context, events []types.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range events {
		if err := s.appendEventLocked(e); err != nil {
			return err
		}
	}
	return nil
}

func (s *FSStore) appendEventLocked(event types.Event) error {
	// 1. Serialize entire event to JSON to get the full struct fields
	fullBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// 2. Wrap in a record that allows generic reading later (or just append line as is)
	// Actually, if we use polymorphic types, reading back is tricky without a discriminator.
	// Our `eventRecord` approach or sticking to a convention helps.
	// The simple way: Write the JSON line.
	// We need to ensure newlines.

	f, err := os.OpenFile(s.eventLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open event log: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(fullBytes); err != nil {
		return err
	}
	if _, err := f.WriteString("\n"); err != nil {
		return err
	}

	return f.Sync() // Ensure durability
}

func (s *FSStore) GetEvent(ctx context.Context, id string) (types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var found types.Event
	err := s.scanEventsLocked(func(e types.Event) error {
		if e.EventID() == id {
			found = e
			return context.Canceled // Found, stahp
		}
		return nil
	})

	if err == context.Canceled {
		return found, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, ErrNotFound
}

func (s *FSStore) GetEventsSince(ctx context.Context, afterEventID string) ([]types.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []types.Event
	startCollecting := (afterEventID == "")

	err := s.scanEventsLocked(func(e types.Event) error {
		if startCollecting {
			result = append(result, e)
		} else if e.EventID() == afterEventID {
			startCollecting = true
		}
		return nil
	})

	return result, err
}

func (s *FSStore) IterEvents(ctx context.Context, fn func(types.Event) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.scanEventsLocked(fn)
}

func (s *FSStore) scanEventsLocked(fn func(types.Event) error) error {
	f, err := os.Open(s.eventLogPath)
	if os.IsNotExist(err) {
		return nil // No events yet
	}
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Handle large tokens if necessary
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max line

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Deserialization Strategy:
		// 1. Unmarshal into BaseEvent to get Type
		// 2. Unmarshal into concrete struct
		var base types.BaseEvent
		if err := json.Unmarshal(line, &base); err != nil {
			return fmt.Errorf("corrupt event log line: %w", err)
		}

		var evt types.Event
		switch base.Type {
		case "user_message", "user_request": // Handle alias if needed
			var e types.UserMessageEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "llm_response":
			var e types.LLMResponseEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "tool_result":
			var e types.ToolResultEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "error":
			var e types.ErrorEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "checkpoint":
			var e types.CheckpointEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "permission_request":
			var e types.PermissionRequestEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		case "permission_response":
			var e types.PermissionResponseEvent
			_ = json.Unmarshal(line, &e)
			evt = &e
		default:
			// Fallback or unknown
			evt = &base
		}

		if err := fn(evt); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// --- State Operations ---

func (s *FSStore) SaveState(ctx context.Context, state *types.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.rootDir, "state", "state.json")
	return s.atomicWrite(path, data)
}

func (s *FSStore) atomicWrite(path string, data []byte) error {
	tmpPath := path + ".tmp"

	// 1. Write to temp
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}

	// 2. Sync to disk requires opening the file and calling Sync
	f, err := os.Open(tmpPath)
	if err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	f.Close()

	// 3. Rename
	return os.Rename(tmpPath, path)
}

func (s *FSStore) LoadLatestState(ctx context.Context) (*types.State, error) {
	// FSStore only keeps one state.json currently
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.rootDir, "state", "state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var state types.State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *FSStore) LoadState(ctx context.Context, version int64) (*types.State, error) {
	// FSStore simple implementation does not version state.json.
	// In a real implementation, we might check checkpoints if state.json doesn't match.
	// For now, load latest and check version
	st, err := s.LoadLatestState(ctx)
	if err != nil {
		return nil, err
	}
	if st.Version == version {
		return st, nil
	}
	// Fallback: try to find a checkpoint with this version?
	// This simple FS implementation assumes state.json is the latest.
	return nil, fmt.Errorf("state version %d not found (fs_store only keeps latest tip)", version)
}

// --- Checkpoint Operations ---

func (s *FSStore) SaveCheckpoint(ctx context.Context, cp *types.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := fmt.Sprintf("%d_%s.json", cp.Timestamp.UnixNano(), cp.ID)
	path := filepath.Join(s.rootDir, "checkpoints", filename)

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return err
	}

	return s.atomicWrite(path, data)
}

func (s *FSStore) LoadLatestCheckpoint(ctx context.Context) (*types.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.rootDir, "checkpoints")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) || len(entries) == 0 {
		return nil, ErrNoCheckpoint
	}
	if err != nil {
		return nil, err
	}

	// Sort by Name (which starts with timestamp) desc
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip unreadable
		}
		var cp types.Checkpoint
		if err := json.Unmarshal(data, &cp); err == nil {
			return &cp, nil
		}
	}

	return nil, ErrNoCheckpoint
}

func (s *FSStore) LoadCheckpoint(ctx context.Context, id string) (*types.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.rootDir, "checkpoints")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	// Search for checkpoint with matching ID
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cp types.Checkpoint
		if err := json.Unmarshal(data, &cp); err == nil && cp.ID == id {
			return &cp, nil
		}
	}

	return nil, ErrNotFound
}

// ListCheckpoints returns all checkpoints sorted by timestamp (newest first)
func (s *FSStore) ListCheckpoints(ctx context.Context) ([]types.Checkpoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dir := filepath.Join(s.rootDir, "checkpoints")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) || len(entries) == 0 {
		return []types.Checkpoint{}, nil
	}
	if err != nil {
		return nil, err
	}

	// Sort by name (timestamp prefix) descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	checkpoints := make([]types.Checkpoint, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip unreadable
		}
		var cp types.Checkpoint
		if err := json.Unmarshal(data, &cp); err == nil {
			checkpoints = append(checkpoints, cp)
		}
	}

	return checkpoints, nil
}

// --- Artifact Operations ---

func (s *FSStore) SaveArtifact(ctx context.Context, artifact *types.Artifact) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Metadata
	metaPath := filepath.Join(s.rootDir, "artifacts", artifact.ID+".json")
	metaData, err := json.MarshalIndent(artifact, "", "  ")
	if err != nil {
		return err
	}
	if err := s.atomicWrite(metaPath, metaData); err != nil {
		return err
	}

	// Content (if internal)
	if len(artifact.Content) > 0 {
		contentPath := filepath.Join(s.rootDir, "artifacts", artifact.ID+".blob")
		if err := s.atomicWrite(contentPath, artifact.Content); err != nil {
			return err
		}
	}
	return nil
}

func (s *FSStore) GetArtifact(ctx context.Context, id string) (*types.Artifact, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.rootDir, "artifacts", id+".json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var art types.Artifact
	if err := json.Unmarshal(data, &art); err != nil {
		return nil, err
	}

	// Load content if blob exists and Content is empty
	blobPath := filepath.Join(s.rootDir, "artifacts", id+".blob")
	blobData, err := os.ReadFile(blobPath)
	if err == nil {
		art.Content = blobData
	}

	return &art, nil
}

func (s *FSStore) ListArtifacts(ctx context.Context, filter ArtifactFilter) ([]types.Artifact, error) {
	return nil, errors.New("not implemented")
}

func (s *FSStore) DeleteArtifact(ctx context.Context, id string) error {
	return errors.New("not implemented")
}

// --- Permission Operations ---

func (s *FSStore) AddPermissionRule(ctx context.Context, rule types.PermissionRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	rules, err := s.loadPermissionRulesLocked()
	if err != nil {
		return err
	}

	// Simple deduplication check based on tool and pattern
	for _, r := range rules {
		if r.ToolName == rule.ToolName && r.Pattern == rule.Pattern && r.Action == rule.Action {
			// Already exists
			return nil
		}
	}

	rules = append(rules, rule)
	return s.savePermissionRulesLocked(rules)
}

func (s *FSStore) GetPermissionRules(ctx context.Context) ([]types.PermissionRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadPermissionRulesLocked()
}

func (s *FSStore) loadPermissionRulesLocked() ([]types.PermissionRule, error) {
	path := filepath.Join(s.rootDir, "permissions.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []types.PermissionRule{}, nil
	}
	if err != nil {
		return nil, err
	}

	var rules []types.PermissionRule
	if err := json.Unmarshal(data, &rules); err != nil {
		// If corrupted, return error or empty? Let's return error to be safe
		return nil, fmt.Errorf("failed to parse permissions.json: %w", err)
	}
	return rules, nil
}

func (s *FSStore) savePermissionRulesLocked(rules []types.PermissionRule) error {
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.rootDir, "permissions.json")
	return s.atomicWrite(path, data)
}

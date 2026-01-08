package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// TestConcurrentSwapPendingCommands tests thread safety of swapPendingCommands
func TestConcurrentSwapPendingCommands(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)

	// Spawn multiple goroutines to append and swap concurrently
	var wg sync.WaitGroup
	iterations := 100

	// Append commands
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			rt.appendPendingCommands([]types.Command{
				&types.CallToolCommand{BaseCommand: types.NewBaseCommand("test")},
			})
		}
	}()

	// Swap commands
	totalSwapped := 0
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			cmds := rt.swapPendingCommands()
			totalSwapped += len(cmds)
			time.Sleep(time.Microsecond) // Small delay to interleave
		}
	}()

	wg.Wait()

	// Final swap to get remaining
	remaining := rt.swapPendingCommands()
	totalSwapped += len(remaining)

	if totalSwapped != iterations {
		t.Errorf("expected %d commands total, got %d", iterations, totalSwapped)
	}
}

// TestConcurrentUpdateState tests thread safety of updateState
func TestConcurrentUpdateState(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)

	var wg sync.WaitGroup
	iterations := 100

	// Multiple writers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				state := types.NewState()
				state.Version = int64(id*iterations + j)
				rt.updateState(state)
			}
		}(i)
	}

	// Multiple readers
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = rt.GetState()
			}
		}()
	}

	wg.Wait()

	// No panic means success - race detector will catch issues
}

// TestGetStateReturnsDeepCopy verifies GetState returns independent copy
func TestGetStateReturnsDeepCopy(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)
	rt.state.Goals = []types.Goal{{ID: "g1", Description: "original"}}
	rt.state.Context.Messages = []types.Message{{Content: "original"}}

	// Get a copy
	copy1 := rt.GetState()

	// Modify the copy
	copy1.Goals[0].Description = "modified"
	copy1.Context.Messages[0].Content = "modified"

	// Original should be unchanged
	if rt.state.Goals[0].Description == "modified" {
		t.Errorf("GetState should return deep copy - Goals was modified")
	}
	if rt.state.Context.Messages[0].Content == "modified" {
		t.Errorf("GetState should return deep copy - Messages was modified")
	}
}

// TestSelectGoalReturnsDeepCopy verifies selectGoal returns independent copy
func TestSelectGoalReturnsDeepCopy(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)
	rt.state.Goals = []types.Goal{{ID: "g1", Description: "original", Status: types.GoalStatusPending}}

	goal, err := rt.selectGoal()
	if err != nil {
		t.Fatalf("selectGoal error: %v", err)
	}

	// Modify the returned goal
	goal.Description = "modified"

	// Original should be unchanged
	if rt.state.Goals[0].Description == "modified" {
		t.Errorf("selectGoal should return copy - original was modified")
	}
}

// TestConcurrentIngestAndGetState tests concurrent Ingest and GetState calls
func TestConcurrentIngestAndGetState(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, nil)

	var wg sync.WaitGroup
	iterations := 50

	// Ingest events concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			evt := &types.UserMessageEvent{
				BaseEvent: types.NewBaseEvent("user_message", "user", "cli"),
				Content:   "test message",
			}
			_ = rt.Ingest(context.Background(), evt)
		}
	}()

	// GetState concurrently
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			state := rt.GetState()
			_ = len(state.Context.Messages)
		}
	}()

	wg.Wait()
	// No race condition = success
}

// TestReducerImmutability verifies reducer doesn't modify input state
func TestReducerImmutability(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)

	originalState := types.NewState()
	originalState.Version = 10
	originalState.Goals = []types.Goal{{ID: "g1", Status: types.GoalStatusPending}}
	originalState.Context.Messages = []types.Message{{Content: "original"}}

	evt := &types.UserMessageEvent{
		BaseEvent: types.NewBaseEvent("user_message", "user", "cli"),
		Content:   "new message",
	}

	newState, _, err := rt.reducer(originalState, evt)
	if err != nil {
		t.Fatalf("reducer error: %v", err)
	}

	// Verify original state is unchanged
	if originalState.Version != 10 {
		t.Errorf("original state version was modified")
	}
	if len(originalState.Context.Messages) != 1 {
		t.Errorf("original state messages were modified")
	}
	if originalState.Context.Messages[0].Content != "original" {
		t.Errorf("original message content was modified")
	}

	// Verify new state has changes
	if newState.Version != 11 {
		t.Errorf("new state version should be incremented")
	}
	if len(newState.Context.Messages) != 2 {
		t.Errorf("new state should have 2 messages")
	}
}

// TestApplyEventLockedDirectMutation verifies applyEventLocked works correctly
func TestApplyEventLockedDirectMutation(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)
	initialVersion := rt.state.Version

	evt := &types.UserMessageEvent{
		BaseEvent: types.NewBaseEvent("user_message", "user", "cli"),
		Content:   "test",
	}

	// applyEventLocked should be called with lock already held
	rt.mu.Lock()
	err := rt.applyEventLocked(context.Background(), evt)
	rt.mu.Unlock()

	if err != nil {
		t.Fatalf("applyEventLocked error: %v", err)
	}

	if rt.state.Version != initialVersion+1 {
		t.Errorf("expected version to increment")
	}

	if len(rt.state.Context.Messages) != 1 {
		t.Errorf("expected message to be added")
	}
}

// TestGracefulShutdownWithCancelledContext verifies shutdown works even with cancelled context
func TestGracefulShutdownWithCancelledContext(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, nil)

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Should still succeed because gracefulShutdown uses fresh context
	err := rt.gracefulShutdown(ctx)
	if err != nil {
		t.Fatalf("gracefulShutdown should succeed with cancelled context: %v", err)
	}

	if len(ms.checkpoints) != 1 {
		t.Errorf("expected checkpoint to be saved")
	}
}

// TestSetFileChangeTrackerThreadSafe tests thread safety of SetFileChangeTracker
func TestSetFileChangeTrackerThreadSafe(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, nil)

	var wg sync.WaitGroup

	// Multiple goroutines setting tracker
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rt.SetFileChangeTracker(nil)
		}()
	}

	// Reading state concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rt.GetState()
		}()
	}

	wg.Wait()
	// No race = success
}

// TestCheckpointThreadSafe verifies checkpoint is thread-safe
func TestCheckpointThreadSafe(t *testing.T) {
	ms := &threadSafeMockStore{mockStore: newMockStore()}
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, nil)

	var wg sync.WaitGroup
	iterations := 20

	// Checkpoint calls
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = rt.checkpoint(context.Background())
			}
		}()
	}

	// State modifications
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				state := types.NewState()
				state.Version = int64(j)
				rt.updateState(state)
			}
		}()
	}

	wg.Wait()
	// No race = success
}

// threadSafeMockStore wraps mockStore with mutex for concurrent test
type threadSafeMockStore struct {
	*mockStore
	mu sync.Mutex
}

func (s *threadSafeMockStore) SaveState(ctx context.Context, state *types.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mockStore.SaveState(ctx, state)
}

func (s *threadSafeMockStore) SaveCheckpoint(ctx context.Context, cp *types.Checkpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mockStore.SaveCheckpoint(ctx, cp)
}

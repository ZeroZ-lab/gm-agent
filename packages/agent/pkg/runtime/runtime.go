package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type Config struct {
	MaxSteps           int           `yaml:"max_steps"`
	CheckpointInterval int           `yaml:"checkpoint_interval"`
	DecisionTimeout    time.Duration `yaml:"decision_timeout"`
	DispatchTimeout    time.Duration `yaml:"dispatch_timeout"`
	Model              string        `yaml:"model"` // Active LLM Model Name
}

var DefaultConfig = Config{
	MaxSteps:           100,
	CheckpointInterval: 10,
	DecisionTimeout:    60 * time.Second,
	DispatchTimeout:    300 * time.Second,
}

type Runtime struct {
	config  Config
	store   store.Store
	llm     LLMGateway
	tools   ToolExecutor
	log     *slog.Logger
	tracker patch.FileChangeTracker // Optional: for Code Rewind support

	state *types.State

	// Pending commands from Reducer that need to be executed
	pendingCommands []types.Command

	mu sync.RWMutex
}

// swapPendingCommands atomically retrieves and clears pending commands
func (r *Runtime) swapPendingCommands() []types.Command {
	r.mu.Lock()
	defer r.mu.Unlock()
	cmds := r.pendingCommands
	r.pendingCommands = nil
	return cmds
}

// appendPendingCommands atomically appends commands
func (r *Runtime) appendPendingCommands(cmds []types.Command) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pendingCommands = append(r.pendingCommands, cmds...)
}

// updateState atomically updates state
func (r *Runtime) updateState(newState *types.State) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.state = newState
}

// getCurrentMode returns the current runtime mode
// Defaults to ModeExecuting for backward compatibility
func (r *Runtime) getCurrentMode() types.RuntimeMode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.state.Mode == "" {
		return types.ModeExecuting // Backward compatibility
	}
	return r.state.Mode
}

func New(cfg Config, s store.Store, llm LLMGateway, tools ToolExecutor, logger *slog.Logger) *Runtime {
	if logger == nil {
		logger = slog.Default()
	}
	return &Runtime{
		config: cfg,
		store:  s,
		llm:    llm,
		tools:  tools,
		log:    logger,
		state:  types.NewState(),
	}
}

// SetFileChangeTracker sets the file change tracker for Code Rewind support
func (r *Runtime) SetFileChangeTracker(tracker patch.FileChangeTracker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracker = tracker
}

// Run executes the main loop
func (r *Runtime) Run(ctx context.Context) error {
	// 1. Recover state
	if err := r.recover(ctx); err != nil {
		return fmt.Errorf("recovery failed: %w", err)
	}

	// 2. Main Loop
	for step := 0; step < r.config.MaxSteps; step++ {
		// 2.1 Check cancellation
		select {
		case <-ctx.Done():
			return r.gracefulShutdown(ctx)
		default:
		}

		r.log.Info("step started", "step", step)

		// 2.2 Handle pending commands (from Reducer side-effects)
		// Use atomic accessor for thread safety
		cmds := r.swapPendingCommands()
		if len(cmds) > 0 {
			r.log.Info("executing pending commands", "count", len(cmds))
			events, err := r.dispatch(ctx, cmds)
			if err != nil {
				// Record error logic here
				// For now just log
				r.log.Error("dispatch pending failed", "error", err)
			}

			// Apply events from pending command execution
			for _, event := range events {
				if err := r.applyEvent(ctx, event); err != nil {
					return err
				}
			}

			if step%r.config.CheckpointInterval == 0 {
				if err := r.checkpoint(ctx); err != nil {
					r.log.Warn("checkpoint failed", "error", err)
				}
			}
			continue
		}

		// 2.3 Select Goal (Simplistic Logic for MVP: First Pending Goal)
		goal, err := r.selectGoal()
		if err != nil {
			return err
		}
		if goal == nil {
			r.log.Info("no active goals, runtime finished")
			if err := r.checkpoint(ctx); err != nil {
				r.log.Warn("checkpoint failed", "error", err)
			}
			return nil
		}

		// 2.4 Decide (LLM)
		decisionCtx, cancel := context.WithTimeout(ctx, r.config.DecisionTimeout)
		decision, err := r.decide(decisionCtx, goal)
		cancel()

		if err != nil {
			// Retry logic ...
			r.log.Error("decision failed", "error", err)
			// For MVP, just return error
			return err
		}

		// 2.5 Act (Dispatch)
		dispatchCtx, cancel := context.WithTimeout(ctx, r.config.DispatchTimeout)
		events, err := r.dispatch(dispatchCtx, decision.Commands)
		cancel()

		if err != nil {
			r.log.Error("dispatch failed", "error", err)
			// Continue to apply error events if any
		}

		// 2.6 Observe (Apply Events)
		for _, event := range events {
			if err := r.applyEvent(ctx, event); err != nil {
				return err
			}
		}

		// 2.7 Checkpoint
		if step%r.config.CheckpointInterval == 0 {
			if err := r.checkpoint(ctx); err != nil {
				r.log.Warn("checkpoint failed", "error", err)
			}
		}
	}

	return errors.New("max steps exceeded")
}

func (r *Runtime) gracefulShutdown(ctx context.Context) error {
	r.log.Info("shutting down...")
	// Use a fresh context with timeout to ensure checkpoint can complete
	// even if the original context is cancelled
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return r.checkpoint(shutdownCtx)
}

func (r *Runtime) recover(ctx context.Context) error {
	// 1. Load latest state directly from FS implementation assumption
	// (Real logic would check checkpoints)
	state, err := r.store.LoadLatestState(ctx)
	if errors.Is(err, store.ErrNotFound) || errors.Is(err, store.ErrNoCheckpoint) {
		r.log.Info("no previous state found, starting fresh")
		return nil
	}
	if err != nil {
		return err
	}
	r.updateState(state)
	r.log.Info("state recovered", "version", state.Version)
	return nil
}

func (r *Runtime) selectGoal() (*types.Goal, error) {
	// Thread-safe goal selection with RLock
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Simple FIFO: return first Pending or InProgress goal
	for i := range r.state.Goals {
		g := &r.state.Goals[i]
		if g.Status == types.GoalStatusPending || g.Status == types.GoalStatusInProgress {
			// Return a copy to avoid external mutation
			goalCopy := *g
			return &goalCopy, nil
		}
	}
	return nil, nil
}

// decide asks LLM what to do
type Decision struct {
	Commands []types.Command
}

// Ingest accepts an external event (like user input) and applies it to the state
func (r *Runtime) Ingest(ctx context.Context, event types.Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Persistence
	if err := r.store.AppendEvent(ctx, event); err != nil {
		return err
	}

	// Apply to Memory State (using locked version since we already hold the lock)
	if err := r.applyEventLocked(ctx, event); err != nil {
		return err
	}

	// Save State immediately for safety
	return r.store.SaveState(ctx, r.state)
}

func (r *Runtime) decide(ctx context.Context, goal *types.Goal) (*Decision, error) {
	// Thread-safe: get a snapshot of state data while holding lock
	r.mu.RLock()
	messages := make([]types.Message, len(r.state.Context.Messages))
	for i, m := range r.state.Context.Messages {
		messages[i] = m.Clone()
	}
	systemPrompt := r.state.SystemPrompt
	r.mu.RUnlock()

	// Add System Prompt with Goal
	if systemPrompt == "" {
		systemPrompt = fmt.Sprintf(`You are an AI coding assistant. You help users with software engineering tasks through conversation.

## How to Respond
- For questions, explanations, or simple requests: respond with text directly
- For tasks requiring action (reading files, running commands): use the appropriate tool
- Be helpful, concise, and friendly

## When to Use Tools
- Use 'run_shell' ONLY when the user asks you to run a command or check something specific
- Use 'read_file' when the user asks about file contents
- Do NOT use tools just to gather context - only use them when explicitly needed

## Communication
- Respond directly to the user in your output
- No need to use 'talk' tool unless you have a very long multi-step response

Current context: %s
When the user's request is fully addressed, use 'task_complete'.`, goal.Description)
	} else {
		systemPrompt = fmt.Sprintf("%s\n\nCurrent Goal: %s (Status: %s). Use 'task_complete' when done.", systemPrompt, goal.Description, goal.Status)
	}

	sysMsg := types.Message{
		Role:    "system",
		Content: systemPrompt,
	}
	// Prepend system message
	messages = append([]types.Message{sysMsg}, messages...)

	// Create Command to Call LLM
	cmd := &types.CallLLMCommand{
		BaseCommand: types.NewBaseCommand("call_llm"),
		Model:       r.config.Model,
		Messages:    messages,
		Tools:       r.tools.List(),
	}

	return &Decision{
		Commands: []types.Command{cmd},
	}, nil
}

func (r *Runtime) checkpoint(ctx context.Context) error {
	// Thread-safe: get a deep copy of state while holding lock
	r.mu.RLock()
	stateCopy := r.state.Clone()
	r.mu.RUnlock()

	// Collect file changes from tracker if available
	var fileChanges []types.FileChange
	if r.tracker != nil {
		fileChanges = r.tracker.Flush()
	}

	cp := &types.Checkpoint{
		ID:           types.GenerateID("ckpt"),
		StateVersion: stateCopy.Version,
		Timestamp:    time.Now(),
		State:        stateCopy, // Use the cloned state
		FileChanges:  fileChanges,
	}
	// Save State first
	if err := r.store.SaveState(ctx, stateCopy); err != nil {
		return err
	}
	// Save CP
	return r.store.SaveCheckpoint(ctx, cp)
}

// GetState returns a deep copy of the current state (thread-safe)
func (r *Runtime) GetState() *types.State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state.Clone()
}

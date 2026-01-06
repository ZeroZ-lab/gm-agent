package runtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type Config struct {
	MaxSteps           int           `yaml:"max_steps"`
	CheckpointInterval int           `yaml:"checkpoint_interval"`
	DecisionTimeout    time.Duration `yaml:"decision_timeout"`
	DispatchTimeout    time.Duration `yaml:"dispatch_timeout"`
}

var DefaultConfig = Config{
	MaxSteps:           100,
	CheckpointInterval: 10,
	DecisionTimeout:    60 * time.Second,
	DispatchTimeout:    300 * time.Second,
}

type Runtime struct {
	config Config
	store  store.Store
	llm    LLMGateway
	tools  ToolExecutor
	log    *slog.Logger

	state *types.State

	// Pending commands from Reducer that need to be executed
	pendingCommands []types.Command

	mu sync.Mutex
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
		if len(r.pendingCommands) > 0 {
			cmds := r.pendingCommands
			r.pendingCommands = nil // Clear pending

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
			continue
		}

		// 2.3 Select Goal (Simplistic Logic for MVP: First Pending Goal)
		goal, err := r.selectGoal()
		if err != nil {
			return err
		}
		if goal == nil {
			r.log.Info("no active goals, runtime finished")
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
	return r.checkpoint(ctx)
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
	r.state = state
	r.log.Info("state recovered", "version", state.Version)
	return nil
}

func (r *Runtime) selectGoal() (*types.Goal, error) {
	// Simple FIFO: return first Pending or InProgress goal
	for i := range r.state.Goals {
		g := &r.state.Goals[i]
		if g.Status == types.GoalStatusPending || g.Status == types.GoalStatusInProgress {
			return g, nil
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

	// Apply to Memory State
	if err := r.applyEvent(ctx, event); err != nil {
		return err
	}

	// Save State immediately for safety
	return r.store.SaveState(ctx, r.state)
}

func (r *Runtime) decide(ctx context.Context, goal *types.Goal) (*Decision, error) {
	// Construct messages from context
	messages := make([]types.Message, len(r.state.Context.Messages))
	copy(messages, r.state.Context.Messages)

	// Add System Prompt with Goal
	sysMsg := types.Message{
		Role:    "system",
		Content: fmt.Sprintf("Current Goal: %s (Status: %s)", goal.Description, goal.Status),
	}
	// Prepend system message
	messages = append([]types.Message{sysMsg}, messages...)

	// Create Command to Call LLM
	cmd := &types.CallLLMCommand{
		BaseCommand: types.NewBaseCommand("call_llm"),
		Model:       "default-model", // Should come from config
		Messages:    messages,
		// Tools: r.tools.List(), // Need ToolRegistry interface on Runtime
	}

	return &Decision{
		Commands: []types.Command{cmd},
	}, nil
}

func (r *Runtime) checkpoint(ctx context.Context) error {
	cp := &types.Checkpoint{
		ID:           types.GenerateID("ckpt"),
		StateVersion: r.state.Version,
		Timestamp:    time.Now(),
		State:        r.state, // Embedding state for FS Store simplicity
	}
	// Save State first
	if err := r.store.SaveState(ctx, r.state); err != nil {
		return err
	}
	// Save CP
	return r.store.SaveCheckpoint(ctx, cp)
}

package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type mockStore struct {
	events      []types.Event
	savedState  *types.State
	checkpoints []*types.Checkpoint
	artifacts   map[string]*types.Artifact
}

func newMockStore() *mockStore {
	return &mockStore{artifacts: make(map[string]*types.Artifact)}
}

func (m *mockStore) Open(ctx context.Context) error { return nil }
func (m *mockStore) Close() error                   { return nil }
func (m *mockStore) AppendEvent(ctx context.Context, event types.Event) error {
	m.events = append(m.events, event)
	return nil
}
func (m *mockStore) AppendEvents(ctx context.Context, events []types.Event) error {
	m.events = append(m.events, events...)
	return nil
}
func (m *mockStore) GetEvent(ctx context.Context, id string) (types.Event, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) GetEventsSince(ctx context.Context, afterEventID string) ([]types.Event, error) {
	return nil, nil
}
func (m *mockStore) IterEvents(ctx context.Context, fn func(types.Event) error) error { return nil }
func (m *mockStore) SaveState(ctx context.Context, state *types.State) error {
	m.savedState = state
	return nil
}
func (m *mockStore) LoadState(ctx context.Context, version int64) (*types.State, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) LoadLatestState(ctx context.Context) (*types.State, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) SaveCheckpoint(ctx context.Context, cp *types.Checkpoint) error {
	m.checkpoints = append(m.checkpoints, cp)
	return nil
}
func (m *mockStore) LoadCheckpoint(ctx context.Context, id string) (*types.Checkpoint, error) {
	return nil, store.ErrNoCheckpoint
}
func (m *mockStore) LoadLatestCheckpoint(ctx context.Context) (*types.Checkpoint, error) {
	return nil, store.ErrNoCheckpoint
}
func (m *mockStore) SaveArtifact(ctx context.Context, artifact *types.Artifact) error {
	m.artifacts[artifact.ID] = artifact
	return nil
}
func (m *mockStore) GetArtifact(ctx context.Context, id string) (*types.Artifact, error) {
	return nil, store.ErrNotFound
}
func (m *mockStore) ListArtifacts(ctx context.Context, filter store.ArtifactFilter) ([]types.Artifact, error) {
	return nil, nil
}
func (m *mockStore) DeleteArtifact(ctx context.Context, id string) error { return nil }
func (m *mockStore) ListCheckpoints(ctx context.Context) ([]types.Checkpoint, error) {
	result := make([]types.Checkpoint, len(m.checkpoints))
	for i, cp := range m.checkpoints {
		result[i] = *cp
	}
	return result, nil
}
func (m *mockStore) AddPermissionRule(ctx context.Context, rule types.PermissionRule) error {
	return nil
}
func (m *mockStore) GetPermissionRules(ctx context.Context) ([]types.PermissionRule, error) {
	return nil, nil
}

type mockLLM struct{}

func (mockLLM) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	return &llm.ChatResponse{Model: req.Model, Content: "reply"}, nil
}

func (mockLLM) StreamChat(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	ch := make(chan llm.StreamChunk)
	go func() {
		defer close(ch)
		ch <- llm.StreamChunk{Content: "reply"}
	}()
	return ch, nil
}

type mockTools struct{ executed []*types.ToolCall }

func (m *mockTools) Execute(ctx context.Context, call *types.ToolCall) (*types.ToolResult, error) {
	m.executed = append(m.executed, call)
	if call.Name == "fail" {
		return nil, errors.New("failure")
	}
	return &types.ToolResult{ToolCallID: call.ID, ToolName: call.Name, Content: "ok"}, nil
}
func (m *mockTools) List() []types.Tool { return []types.Tool{{Name: "talk"}} }

func TestApplyEventAndReducer(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, slog.Default())
	evt := &types.UserMessageEvent{BaseEvent: types.NewBaseEvent("user_message", "user", "cli"), Content: "hello"}
	if err := rt.applyEvent(context.Background(), evt); err != nil {
		t.Fatalf("apply event error: %v", err)
	}
	if len(rt.state.Context.Messages) == 0 {
		t.Fatalf("expected context to have messages")
	}

	respEvt := &types.LLMResponseEvent{
		BaseEvent: types.NewBaseEvent("llm_response", "llm", ""),
		Content:   "",
		ToolCalls: []types.ToolCall{{ID: "call-1", Name: "tool", Arguments: `{"key":"value"}`}},
	}
	if err := rt.applyEvent(context.Background(), respEvt); err != nil {
		t.Fatalf("apply llm event error: %v", err)
	}
	if len(rt.pendingCommands) == 0 {
		t.Fatalf("expected pending command for tool call")
	}
	cmd, ok := rt.pendingCommands[0].(*types.CallToolCommand)
	if !ok {
		t.Fatalf("expected CallToolCommand, got %T", rt.pendingCommands[0])
	}
	if cmd.ToolCallID != "call-1" {
		t.Fatalf("expected tool call id to be propagated")
	}
	if cmd.Arguments["key"] != "value" {
		t.Fatalf("expected arguments to be parsed")
	}
}

func TestDispatchAndCheckpoint(t *testing.T) {
	ms := newMockStore()
	tools := &mockTools{}
	rt := New(DefaultConfig, ms, mockLLM{}, tools, slog.Default())
	cmds := []types.Command{&types.CallToolCommand{
		BaseCommand: types.NewBaseCommand("call_tool"),
		ToolCallID:  "call-1",
		ToolName:    "echo",
		Arguments:   map[string]any{"message": "hi"},
	}}
	events, err := rt.dispatch(context.Background(), cmds)
	if err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %d", len(events))
	}
	if len(tools.executed) != 1 {
		t.Fatalf("expected tool to be executed")
	}
	if tools.executed[0].ID != "call-1" {
		t.Fatalf("expected tool call id to be set on execution")
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(tools.executed[0].Arguments), &args); err != nil {
		t.Fatalf("unexpected tool args json: %v", err)
	}
	if args["message"] != "hi" {
		t.Fatalf("expected tool args to be passed through")
	}
	if res, ok := events[0].(*types.ToolResultEvent); !ok || res.ToolCallID != "call-1" {
		t.Fatalf("expected tool result event to include tool call id")
	}
	if len(ms.events) != 1 {
		t.Fatalf("expected events to be persisted")
	}

	if err := rt.checkpoint(context.Background()); err != nil {
		t.Fatalf("checkpoint error: %v", err)
	}
	if len(ms.checkpoints) != 1 {
		t.Fatalf("expected checkpoint saved")
	}
}

func TestDecideBuildsCallCommand(t *testing.T) {
	tools := &mockTools{}
	cfg := DefaultConfig
	cfg.Model = "test-model"
	rt := New(cfg, newMockStore(), mockLLM{}, tools, slog.Default())
	goal := &types.Goal{Description: "test", Status: types.GoalStatusPending}
	decision, err := rt.decide(context.Background(), goal)
	if err != nil {
		t.Fatalf("decide error: %v", err)
	}
	if len(decision.Commands) != 1 {
		t.Fatalf("expected command from decide")
	}
	cmd, ok := decision.Commands[0].(*types.CallLLMCommand)
	if !ok || cmd.Model != "test-model" {
		t.Fatalf("expected call llm command with model")
	}
	if len(cmd.Tools) == 0 {
		t.Fatalf("expected tools to be included")
	}
}

func TestIngestPersistsAndApplies(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, slog.Default())
	evt := &types.UserMessageEvent{BaseEvent: types.NewBaseEvent("user_message", "user", "cli"), Content: "hi"}
	if err := rt.Ingest(context.Background(), evt); err != nil {
		t.Fatalf("ingest error: %v", err)
	}
	if len(ms.events) != 1 {
		t.Fatalf("expected event persisted")
	}
	if len(rt.state.Context.Messages) == 0 {
		t.Fatalf("expected event applied to state")
	}
}

func TestRecoverHandlesMissingState(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, slog.Default())
	if err := rt.recover(context.Background()); err != nil {
		t.Fatalf("expected no error on missing state: %v", err)
	}
}

func TestGracefulShutdown(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, slog.Default())
	if err := rt.gracefulShutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
	if len(ms.checkpoints) != 1 {
		t.Fatalf("expected checkpoint on shutdown")
	}
}

func TestSelectGoal(t *testing.T) {
	rt := New(DefaultConfig, newMockStore(), mockLLM{}, &mockTools{}, slog.Default())
	rt.state.Goals = []types.Goal{{ID: "1", Status: types.GoalStatusPending}}
	goal, err := rt.selectGoal()
	if err != nil || goal == nil || goal.ID != "1" {
		t.Fatalf("unexpected goal selection: %v %+v", err, goal)
	}
}

func TestRunStopsWithoutGoals(t *testing.T) {
	ms := newMockStore()
	rt := New(DefaultConfig, ms, mockLLM{}, &mockTools{}, slog.Default())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := rt.Run(ctx); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

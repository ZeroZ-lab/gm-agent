package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestFSStoreEvents(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := NewFSStore(dir)
	if err := s.Open(ctx); err != nil {
		t.Fatalf("open store: %v", err)
	}

	evt := &types.UserMessageEvent{BaseEvent: types.NewBaseEvent("user_message", "user", "cli"), Content: "hi"}
	if err := s.AppendEvent(ctx, evt); err != nil {
		t.Fatalf("append event: %v", err)
	}

	stored, err := s.GetEvent(ctx, evt.EventID())
	if err != nil {
		t.Fatalf("get event: %v", err)
	}
	if stored.EventID() != evt.EventID() {
		t.Fatalf("unexpected event retrieved: %+v", stored)
	}

	events, err := s.GetEventsSince(ctx, "")
	if err != nil || len(events) != 1 {
		t.Fatalf("expected one event, got %d err %v", len(events), err)
	}
}

func TestFSStoreStateAndCheckpoint(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := NewFSStore(dir)
	if err := s.Open(ctx); err != nil {
		t.Fatalf("open store: %v", err)
	}

	st := types.NewState()
	st.Version = 2
	if err := s.SaveState(ctx, st); err != nil {
		t.Fatalf("save state: %v", err)
	}

	loaded, err := s.LoadLatestState(ctx)
	if err != nil {
		t.Fatalf("load state: %v", err)
	}
	if loaded.Version != st.Version {
		t.Fatalf("unexpected state version %d", loaded.Version)
	}

	cp := &types.Checkpoint{ID: "cp1", StateVersion: st.Version, Timestamp: time.Now(), State: st}
	if err := s.SaveCheckpoint(ctx, cp); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	latest, err := s.LoadLatestCheckpoint(ctx)
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if latest.ID != cp.ID {
		t.Fatalf("unexpected checkpoint: %+v", latest)
	}
}

func TestFSStoreArtifacts(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s := NewFSStore(dir)
	if err := s.Open(ctx); err != nil {
		t.Fatalf("open store: %v", err)
	}

	art := &types.Artifact{ID: "a1", Name: "sample.txt", Content: []byte("data")}
	if err := s.SaveArtifact(ctx, art); err != nil {
		t.Fatalf("save artifact: %v", err)
	}

	loaded, err := s.GetArtifact(ctx, "a1")
	if err != nil {
		t.Fatalf("get artifact: %v", err)
	}
	if string(loaded.Content) != "data" {
		t.Fatalf("unexpected artifact content: %s", string(loaded.Content))
	}

	if _, err := s.ListArtifacts(ctx, ArtifactFilter{}); err == nil {
		t.Fatalf("expected not implemented error for list")
	}
	if _, err := s.LoadCheckpoint(ctx, "missing"); err == nil {
		t.Fatalf("expected error for unimplemented load checkpoint")
	}
	if err := s.DeleteArtifact(ctx, "a1"); err == nil {
		t.Fatalf("expected error for unimplemented delete artifact")
	}

	// Ensure files exist on disk
	if _, err := os.Stat(filepath.Join(dir, "artifacts", "a1.json")); err != nil {
		t.Fatalf("expected artifact metadata file: %v", err)
	}
}

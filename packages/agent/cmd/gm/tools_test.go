package main_test

import (
	"context"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/agent/tools"
	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Test that all new tools are properly registered and can be executed
func TestToolsRegistration(t *testing.T) {
	registry := tool.NewRegistry()

	// Register all tools
	tools := []types.Tool{
		tools.ReadFileTool,
		tools.WriteFileTool,
		tools.EditFileTool,
		tools.GlobTool,
		tools.GrepTool,
		tools.RunShellTool,
		tools.TalkTool,
		tools.TaskCompleteTool,
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			t.Errorf("Failed to register tool %s: %v", tool.Name, err)
		}
	}

	// Verify all tools are in registry
	allTools := registry.List()
	if len(allTools) != len(tools) {
		t.Errorf("Expected %d tools, got %d", len(tools), len(allTools))
	}

	// Check specific tools
	expectedTools := []string{
		"read_file", "write_file", "edit_file",
		"glob", "grep", "run_shell", "talk", "task_complete",
	}

	for _, name := range expectedTools {
		if _, ok := registry.Get(name); !ok {
			t.Errorf("Tool %s not found in registry", name)
		}
	}
}

// Test patch engine integration
func TestPatchEngineIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	engine, err := patch.NewEngine(patch.Config{
		WorkDir:         tmpDir,
		BackupDir:       ".backups",
		MaxContextLines: 3,
	})

	if err != nil {
		t.Fatalf("Failed to create patch engine: %v", err)
	}

	ctx := context.Background()

	// Test diff generation
	diff, err := engine.GenerateDiff(ctx, "test.txt", "old content\n", "new content\n")
	if err != nil {
		t.Errorf("GenerateDiff failed: %v", err)
	}

	if diff == "" {
		t.Error("Expected non-empty diff")
	}
}

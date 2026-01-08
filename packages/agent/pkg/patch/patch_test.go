package patch_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestPatchEngine(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()

	cfg := patch.Config{
		WorkDir:         tmpDir,
		BackupDir:       ".backups",
		MaxContextLines: 3,
	}

	engine, err := patch.NewEngine(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}

	ctx := context.Background()

	t.Run("GenerateDiff", func(t *testing.T) {
		oldContent := "Hello World\nThis is a test\n"
		newContent := "Hello Universe\nThis is a test\nNew line added\n"

		diff, err := engine.GenerateDiff(ctx, "test.txt", oldContent, newContent)
		if err != nil {
			t.Fatalf("GenerateDiff failed: %v", err)
		}

		if diff == "" {
			t.Error("Expected non-empty diff")
		}

		t.Logf("Generated diff:\n%s", diff)
	})

	t.Run("ApplyPatch", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(tmpDir, "apply_test.txt")
		originalContent := "Line 1\nLine 2\nLine 3\n"
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Generate diff
		newContent := "Line 1\nLine 2 Modified\nLine 3\nLine 4\n"
		diff, err := engine.GenerateDiff(ctx, testFile, originalContent, newContent)
		if err != nil {
			t.Fatalf("GenerateDiff failed: %v", err)
		}

		// Apply patch
		cmd := types.ApplyPatchCommand{
			BaseCommand: types.NewBaseCommand("apply_patch"),
			FilePath:    testFile,
			Diff:        diff,
			DryRun:      false,
		}

		result, err := engine.Apply(ctx, cmd)
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Patch application failed: %s", result.Error)
		}

		// Verify file content
		content, err := os.ReadFile(testFile)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(content) != newContent {
			t.Errorf("Content mismatch.\nExpected: %q\nGot: %q", newContent, string(content))
		}

		// Verify backup was created
		if result.BackupPath == "" {
			t.Error("Expected backup path")
		}

		t.Logf("Patch ID: %s", result.PatchID)
		t.Logf("Backup: %s", result.BackupPath)
	})

	t.Run("Rollback", func(t *testing.T) {
		// Create and modify a file
		testFile := filepath.Join(tmpDir, "rollback_test.txt")
		originalContent := "Original\n"
		if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Apply a patch
		newContent := "Modified\n"
		diff, _ := engine.GenerateDiff(ctx, testFile, originalContent, newContent)

		cmd := types.ApplyPatchCommand{
			BaseCommand: types.NewBaseCommand("apply_patch"),
			FilePath:    testFile,
			Diff:        diff,
		}

		result, err := engine.Apply(ctx, cmd)
		if err != nil {
			t.Fatalf("Apply failed: %v", err)
		}

		// Rollback
		if err := engine.Rollback(ctx, result.PatchID); err != nil {
			t.Fatalf("Rollback failed: %v", err)
		}

		// Verify content is restored
		content, _ := os.ReadFile(testFile)
		if string(content) != originalContent {
			t.Errorf("Rollback failed.\nExpected: %q\nGot: %q", originalContent, string(content))
		}
	})

	t.Run("PathValidation", func(t *testing.T) {
		// Test path traversal prevention
		cmd := types.ApplyPatchCommand{
			BaseCommand: types.NewBaseCommand("apply_patch"),
			FilePath:    "../../etc/passwd",
			Diff:        "fake diff",
		}

		_, err := engine.Apply(ctx, cmd)
		if err == nil {
			t.Error("Expected error for path traversal attempt")
		}
	})
}

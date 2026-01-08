package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// WriteFileTool creates or overwrites a file with content and backup
var WriteFileTool = types.Tool{
	Name:        "write_file",
	Description: "Write content to a file with automatic backup. Creates parent directories if needed.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to write to (relative or absolute)",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The content to write to the file",
			},
		},
		"required": []string{"path", "content"},
	},
	Metadata: map[string]string{
		"category": "filesystem",
	},
}

// EditFileTool applies precise edits to a file using diff/patch
var EditFileTool = types.Tool{
	Name:        "edit_file",
	Description: "Edit an existing file by specifying old and new content. Generates a diff and applies it with backup support.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The file path to edit",
			},
			"old_content": map[string]any{
				"type":        "string",
				"description": "The exact old content to replace (must match exactly)",
			},
			"new_content": map[string]any{
				"type":        "string",
				"description": "The new content to replace with",
			},
		},
		"required": []string{"path", "old_content", "new_content"},
	},
	Metadata: map[string]string{
		"category": "filesystem",
	},
}

type WriteFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func HandleWriteFile(ctx context.Context, argsJSON string, patchEngine patch.Engine) (string, error) {
	var args WriteFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Ensure parent directory exists
	dir := filepath.Dir(args.Path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Read existing content for backup
	oldContent := ""
	if data, err := os.ReadFile(args.Path); err == nil {
		oldContent = string(data)
	}

	// Generate diff
	diff := ""
	if oldContent != "" {
		var err error
		diff, err = patchEngine.GenerateDiff(ctx, args.Path, oldContent, args.Content)
		if err != nil && err.Error() != "no changes detected" {
			return "", fmt.Errorf("failed to generate diff: %w", err)
		}
	}

	// Apply using patch engine if we have existing content
	if oldContent != "" && diff != "" {
		cmd := types.ApplyPatchCommand{
			BaseCommand: types.NewBaseCommand("apply_patch"),
			FilePath:    args.Path,
			Diff:        diff,
			DryRun:      false,
		}

		result, err := patchEngine.Apply(ctx, cmd)
		if err != nil {
			return "", fmt.Errorf("failed to apply patch: %w", err)
		}

		return fmt.Sprintf("Successfully wrote %d lines (+%d -%d) to %s\nBackup: %s\nPatch ID: %s",
			len([]rune(args.Content)), result.LinesAdded, result.LinesRemoved,
			args.Path, result.BackupPath, result.PatchID), nil
	}

	// No existing file or no changes - direct write
	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return fmt.Sprintf("Successfully created %s", args.Path), nil
}

type EditFileArgs struct {
	Path       string `json:"path"`
	OldContent string `json:"old_content"`
	NewContent string `json:"new_content"`
}

func HandleEditFile(ctx context.Context, argsJSON string, patchEngine patch.Engine) (string, error) {
	var args EditFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Read current file content
	currentData, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	currentContent := string(currentData)

	// Verify old_content exists in current file
	// For exact replacement, we'll do a simple string replace
	// This is safer than full file diff for targeted edits
	if !contains(currentContent, args.OldContent) {
		return "", fmt.Errorf("old_content not found in file. File may have changed.")
	}

	// Generate new content by replacing
	newContent := replace(currentContent, args.OldContent, args.NewContent)

	// Generate diff
	diff, err := patchEngine.GenerateDiff(ctx, args.Path, currentContent, newContent)
	if err != nil {
		return "", fmt.Errorf("failed to generate diff: %w", err)
	}

	// Apply patch
	cmd := types.ApplyPatchCommand{
		BaseCommand: types.NewBaseCommand("apply_patch"),
		FilePath:    args.Path,
		Diff:        diff,
		DryRun:      false,
	}

	result, err := patchEngine.Apply(ctx, cmd)
	if err != nil {
		return "", fmt.Errorf("failed to apply patch: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("patch application failed: %s", result.Error)
	}

	return fmt.Sprintf("Successfully edited %s\n+%d -%d lines\nBackup: %s\nPatch ID: %s",
		args.Path, result.LinesAdded, result.LinesRemoved,
		result.BackupPath, result.PatchID), nil
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr) != -1)
}

func findSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func replace(s, old, new string) string {
	idx := findSubstring(s, old)
	if idx == -1 {
		return s
	}
	return s[:idx] + new + s[idx+len(old):]
}

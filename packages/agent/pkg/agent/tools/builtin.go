package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Definitions

var ReadFileTool = types.Tool{
	Name:        "read_file",
	Description: "Read the contents of a file at the given path",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to read",
			},
		},
		"required": []string{"path"},
	},
	Metadata: map[string]string{
		"category": "filesystem",
	},
	ReadOnly: true, // Read-only operation, safe for planning mode
}

var RunShellTool = types.Tool{
	Name:        "run_shell",
	Description: "Execute a shell command",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The command line to execute",
			},
		},
		"required": []string{"command"},
	},
	Metadata: map[string]string{
		"category": "shell",
	},
	ReadOnly: false, // Shell commands can modify system state
}

// Implementations

type ReadFileArgs struct {
	Path string `json:"path"`
}

func HandleReadFile(ctx context.Context, argsJSON string) (string, error) {
	var args ReadFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Security: Validate path (simple check for now)
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	data, err := os.ReadFile(args.Path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

type RunShellArgs struct {
	Command string `json:"command"`
}

func HandleRunShell(ctx context.Context, argsJSON string) (string, error) {
	var args RunShellArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Command == "" {
		return "", fmt.Errorf("command is required")
	}

	// Use bash -c (Caution: Security Risk - This is MVP)
	cmd := exec.CommandContext(ctx, "bash", "-c", args.Command)

	// Capture combined output
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Error: %v\nOutput:\n%s", err, string(output)), nil // Return error as content so LLM sees it
	}

	return string(output), nil
}

// CreateFileTool
var CreateFileTool = types.Tool{
	Name:        "create_file",
	Description: "Create or overwrite a file at the given path with content",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to create",
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
	ReadOnly: false, // File modification operation
}

type CreateFileArgs struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func HandleCreateFile(ctx context.Context, argsJSON string) (string, error) {
	var args CreateFileArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Security: MVP does not restrict paths, but in prod we should sandbox
	// Note: Directory creation is not automatic in MVP unless requested?
	// Let's safe-guard by not recursively creating dirs for now, or just do it?
	// Go's os.WriteFile requires directory to exist.
	// Let's keep it simple: if dir doesn't exist, it fails.
	// Or we can be nice and mkdirs.
	// Let's rely on standard WriteFile behavior for failure if dir missing,
	// forcing agent to run_shell mkdir if needed?
	// Actually for agent UX, auto-mkdir is better. But adds import "path/filepath".
	// I'll skip import for now to avoid multiple-edit complexity, just WriteFile.
	// If it fails, Agent will see error "no such directory" and can fix it.

	if err := os.WriteFile(args.Path, []byte(args.Content), 0644); err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully wrote to %s", args.Path), nil
}

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

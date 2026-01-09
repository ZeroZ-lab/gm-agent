package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Definitions

var TalkTool = types.Tool{
	Name:        "talk",
	Description: "Speak to the user. Use this to provide answers, ask questions, or provide updates.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "The message to display to the user",
			},
		},
		"required": []string{"message"},
	},
	ReadOnly: true, // Output-only operation, safe for planning mode
}

var TaskCompleteTool = types.Tool{
	Name:        "task_complete",
	Description: "Signal that the assigned task is completed. Use this when you have achieved the goal.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{
				"type":        "string",
				"description": "A summary of what was accomplished",
			},
		},
		"required": []string{"summary"},
	},
	ReadOnly: false, // Task completion changes system state
}

// Implementations

// Printer interface allows mocking stdout in tests
type Printer interface {
	Println(a ...any)
}

// DefaultPrinter uses fmt
type DefaultPrinter struct{}

func (p DefaultPrinter) Println(a ...any) {
	fmt.Println(a...)
}

// Global printer instance replacement for testing
var printer Printer = DefaultPrinter{}

type TalkArgs struct {
	Message string `json:"message"`
}

func HandleTalk(ctx context.Context, argsJSON string) (string, error) {
	var args TalkArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Message == "" {
		return "", fmt.Errorf("message is required")
	}

	// Print to stdout (via abstraction)
	printer.Println(args.Message)

	return "Message delivered using talk.", nil
}

type TaskCompleteArgs struct {
	Summary string `json:"summary"`
}

func HandleTaskComplete(ctx context.Context, argsJSON string) (string, error) {
	var args TaskCompleteArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	// Just return acknowledgement.
	// The runtime Reducer will inspect the ToolResultEvent from this tool
	// and trigger the goal status update.
	return fmt.Sprintf("Task Completed: %s", args.Summary), nil
}

package gemini

import (
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func TestConvertSchema(t *testing.T) {
	schema := types.JSONSchema{
		"type":        "object",
		"description": "root",
		"properties": map[string]any{
			"field": map[string]any{"type": "string"},
		},
		"required": []any{"field"},
	}
	converted := convertSchema(schema)
	if converted == nil || converted.Type == "" {
		t.Fatalf("expected schema conversion")
	}
	if len(converted.Required) != 1 || converted.Required[0] != "field" {
		t.Fatalf("required fields missing: %+v", converted.Required)
	}
	if _, ok := converted.Properties["field"]; !ok {
		t.Fatalf("expected nested property conversion")
	}
}

func TestConvertMessageErrorsOnBadJSON(t *testing.T) {
	msg := types.Message{Role: "assistant", ToolCalls: []types.ToolCall{{Name: "tool", Arguments: "{bad"}}}
	if _, err := convertMessage(msg); err == nil {
		t.Fatalf("expected error for invalid tool call arguments")
	}
}

func TestConvertMessageToolResponse(t *testing.T) {
	msg := types.Message{Role: "tool", ToolName: "tool", Content: "result"}
	content, err := convertMessage(msg)
	if err != nil {
		t.Fatalf("convert message error: %v", err)
	}
	if len(content.Parts) == 0 {
		t.Fatalf("expected parts for tool response")
	}
}

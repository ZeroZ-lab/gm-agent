package openai

import (
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/types"
	sdk "github.com/sashabaranov/go-openai"
)

func TestConvertMessages(t *testing.T) {
	msgs := []types.Message{{Role: "user", Content: "hi"}, {Role: "assistant", ToolCalls: []types.ToolCall{{ID: "1", Name: "tool", Arguments: "{}"}}}, {Role: "tool", ToolCallID: "1", ToolName: "tool", Content: "result"}}
	converted, err := convertMessages(msgs)
	if err != nil {
		t.Fatalf("convert messages error: %v", err)
	}
	if len(converted) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(converted))
	}
	if converted[1].ToolCalls[0].Function.Name != "tool" {
		t.Fatalf("unexpected tool call conversion: %+v", converted[1].ToolCalls[0])
	}
	if converted[2].Role != "tool" || converted[2].ToolCallID != "1" {
		t.Fatalf("unexpected tool message conversion: %+v", converted[2])
	}
}

func TestConvertToolsAndBack(t *testing.T) {
	tools := []types.Tool{{Name: "read", Description: "desc", Parameters: types.JSONSchema{"type": "object"}}}
	converted := convertTools(tools)
	if len(converted) != 1 || converted[0].Function.Name != "read" {
		t.Fatalf("unexpected conversion: %+v", converted)
	}

	calls := []sdk.ToolCall{{ID: "1", Function: sdk.FunctionCall{Name: "read", Arguments: "{}"}}}
	back := convertToolCalls(calls)
	if len(back) != 1 || back[0].Name != "read" {
		t.Fatalf("unexpected tool call conversion back: %+v", back)
	}
}

func TestConvertUsage(t *testing.T) {
	u := sdk.Usage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3}
	res := convertUsage(u)
	if res.TotalTokens != 3 || res.PromptTokens != 1 || res.CompletionTokens != 2 {
		t.Fatalf("unexpected usage conversion: %+v", res)
	}
}

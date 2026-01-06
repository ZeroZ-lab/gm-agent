package openai

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

type Provider struct {
	client *openai.Client
	config Config
}

type Config struct {
	APIKey  string
	BaseURL string
}

func New(cfg Config) *Provider {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientConfig.BaseURL = cfg.BaseURL
	}
	return &Provider{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}
}

func (p *Provider) ID() string {
	return "openai"
}

func (p *Provider) Call(ctx context.Context, req *llm.ProviderRequest) (*llm.ProviderResponse, error) {
	// 1. Convert Messages
	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("convert messages: %w", err)
	}

	// 2. Convert Tools
	tools := convertTools(req.Tools)

	// 3. Make Request
	openAIReq := openai.ChatCompletionRequest{
		Model:       req.Model,
		Messages:    msgs,
		Tools:       tools,
		MaxTokens:   req.MaxTokens,
		Temperature: float32(req.Temperature),
	}

	resp, err := p.client.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return nil, err
	}

	// 4. Convert Response
	choice := resp.Choices[0]

	llmResp := &llm.ProviderResponse{
		ID:        resp.ID,
		Model:     resp.Model,
		Content:   choice.Message.Content,
		Usage:     convertUsage(resp.Usage),
		ToolCalls: convertToolCalls(choice.Message.ToolCalls),
	}
	return llmResp, nil
}

// Helpers

func convertMessages(msgs []types.Message) ([]openai.ChatCompletionMessage, error) {
	var result []openai.ChatCompletionMessage
	for _, m := range msgs {
		msg := openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}

		// If tool result, we need ToolCallID
		if m.Role == "tool" {
			msg.ToolCallID = m.ToolCallID
		}

		// If assistant has tool calls
		if len(m.ToolCalls) > 0 {
			msg.ToolCalls = make([]openai.ToolCall, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				msg.ToolCalls[i] = openai.ToolCall{
					ID:   tc.ID,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}

		result = append(result, msg)
	}
	return result, nil
}

func convertTools(tools []types.Tool) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}
	result := make([]openai.Tool, len(tools))
	for i, t := range tools {
		result[i] = openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters, // Assuming types.JSONSchema is compatible map[string]any
			},
		}
	}
	return result
}

func convertUsage(u openai.Usage) types.Usage {
	return types.Usage{
		PromptTokens:     u.PromptTokens,
		CompletionTokens: u.CompletionTokens,
		TotalTokens:      u.TotalTokens,
	}
}

func convertToolCalls(calls []openai.ToolCall) []types.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	result := make([]types.ToolCall, len(calls))
	for i, c := range calls {
		result[i] = types.ToolCall{
			ID:        c.ID,
			Name:      c.Function.Name,
			Arguments: c.Function.Arguments,
		}
	}
	return result
}

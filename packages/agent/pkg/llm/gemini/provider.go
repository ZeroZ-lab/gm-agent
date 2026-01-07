package gemini

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/types"
	"google.golang.org/genai"
)

// Config contains Gemini-specific configuration.
type Config struct {
	APIKey    string
	ProjectID string
	Location  string
	Model     string
}

type Provider struct {
	client *genai.Client
	config Config
}

func New(ctx context.Context, cfg Config) (*Provider, error) {
	clientConfig := &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI, // Default to Gemini API
	}

	if cfg.ProjectID != "" && cfg.Location != "" {
		clientConfig.Backend = genai.BackendVertexAI
		clientConfig.Project = cfg.ProjectID
		clientConfig.Location = cfg.Location
	}

	client, err := genai.NewClient(ctx, clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &Provider{
		client: client,
		config: cfg,
	}, nil
}

func (p *Provider) ID() string {
	return "gemini"
}

func (p *Provider) Call(ctx context.Context, req *llm.ProviderRequest) (*llm.ProviderResponse, error) {
	modelName, contents, conf, err := p.prepareCall(req)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Models.GenerateContent(ctx, modelName, contents, conf)
	if err != nil {
		return nil, err
	}

	return convertResponse(resp, modelName)
}

func (p *Provider) CallStream(ctx context.Context, req *llm.ProviderRequest) (<-chan llm.StreamChunk, error) {
	modelName, contents, conf, err := p.prepareCall(req)
	if err != nil {
		return nil, err
	}

	stream := p.client.Models.GenerateContentStream(ctx, modelName, contents, conf)

	ch := make(chan llm.StreamChunk)
	go func() {
		defer close(ch)
		for chunk, err := range stream {
			if err != nil {
				return
			}
			var toolCalls []types.ToolCall
			// Extract tool calls if present in this chunk
			for _, part := range chunk.Candidates[0].Content.Parts {
				if part.FunctionCall != nil {
					argsBytes, _ := json.Marshal(part.FunctionCall.Args)
					toolCalls = append(toolCalls, types.ToolCall{
						Name:      part.FunctionCall.Name,
						Arguments: string(argsBytes),
					})
				}
			}

			if text := chunk.Text(); text != "" || len(toolCalls) > 0 {
				ch <- llm.StreamChunk{
					Content:   text,
					ToolCalls: toolCalls,
				}
			}
		}
	}()

	return ch, nil
}

func (p *Provider) prepareCall(req *llm.ProviderRequest) (string, []*genai.Content, *genai.GenerateContentConfig, error) {
	// 1. Separate System Prompt
	var systemInstruction *genai.Content
	var contents []*genai.Content

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemInstruction = &genai.Content{
				Parts: []*genai.Part{{Text: m.Content}},
			}
			continue
		}

		content, err := convertMessage(m)
		if err != nil {
			return "", nil, nil, err
		}
		contents = append(contents, content)
	}

	// 2. Convert Tools
	tools := convertTools(req.Tools)

	// 3. Prepare Config
	conf := &genai.GenerateContentConfig{
		Temperature:       genai.Ptr(float32(req.Temperature)),
		MaxOutputTokens:   int32(req.MaxTokens),
		SystemInstruction: systemInstruction,
		Tools:             tools,
	}

	// 4. Model Name
	modelName := req.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash"
	}

	return modelName, contents, conf, nil
}

// Helpers

func convertMessage(m types.Message) (*genai.Content, error) {
	role := "user"
	if m.Role == "assistant" {
		role = "model"
	} else if m.Role == "tool" {
		role = "user"
	}

	var parts []*genai.Part

	// 1. Text Content
	if m.Content != "" {
		parts = append(parts, &genai.Part{Text: m.Content})
	}

	// 2. Tool Calls (Assistant -> FunctionCall)
	for _, tc := range m.ToolCalls {
		var args map[string]any
		if tc.Arguments != "" {
			if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to unmarshal tool arguments for %s: %w", tc.Name, err)
			}
		}
		parts = append(parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: tc.Name,
				Args: args,
			},
		})
	}

	// 3. Tool Results (Tool -> FunctionResponse)
	if m.Role == "tool" {
		var response map[string]any
		// Wrap content as "result" to ensure JSON object
		response = map[string]any{"result": m.Content}

		parts = append(parts, &genai.Part{
			FunctionResponse: &genai.FunctionResponse{
				Name:     m.ToolName, // Use the proper Tool Name
				Response: response,
			},
		})
	}

	return &genai.Content{
		Role:  role,
		Parts: parts,
	}, nil
}

func convertTools(tools []types.Tool) []*genai.Tool {
	if len(tools) == 0 {
		return nil
	}
	var fds []*genai.FunctionDeclaration
	for _, t := range tools {
		fds = append(fds, &genai.FunctionDeclaration{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  convertSchema(t.Parameters),
		})
	}

	if len(fds) == 0 {
		return nil
	}

	return []*genai.Tool{
		{
			FunctionDeclarations: fds,
		},
	}
}

func convertSchema(schema types.JSONSchema) *genai.Schema {
	if schema == nil {
		return nil
	}

	valType, _ := schema["type"].(string)

	s := &genai.Schema{
		Type:        toGenaiType(valType),
		Description: getString(schema, "description"),
	}

	if props, ok := schema["properties"].(map[string]any); ok {
		s.Properties = make(map[string]*genai.Schema)
		for k, v := range props {
			if vMap, ok := v.(map[string]any); ok {
				s.Properties[k] = convertSchema(vMap)
			}
		}
	}

	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if str, ok := r.(string); ok {
				s.Required = append(s.Required, str)
			}
		}
	}

	return s
}

func toGenaiType(t string) genai.Type {
	switch t {
	case "string":
		return genai.TypeString
	case "number":
		return genai.TypeNumber
	case "integer":
		return genai.TypeInteger
	case "boolean":
		return genai.TypeBoolean
	case "array":
		return genai.TypeArray
	case "object":
		return genai.TypeObject
	default:
		return genai.TypeString
	}
}

func getString(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

func convertResponse(resp *genai.GenerateContentResponse, model string) (*llm.ProviderResponse, error) {
	if resp == nil || len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}
	cand := resp.Candidates[0]

	var content string
	var toolCalls []types.ToolCall

	for _, part := range cand.Content.Parts {
		if part.Text != "" {
			content += part.Text
		}
		if part.FunctionCall != nil {
			// Marshal args back to JSON string
			argsBytes, _ := json.Marshal(part.FunctionCall.Args)
			toolCalls = append(toolCalls, types.ToolCall{
				ID:        "", // Gemini doesn't always provide Call ID in v1beta.
				Name:      part.FunctionCall.Name,
				Arguments: string(argsBytes),
			})
		}
	}

	llmResp := &llm.ProviderResponse{
		ID:        "", // ID not always available?
		Model:     model,
		Content:   content,
		ToolCalls: toolCalls,
		Usage: types.Usage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		},
	}
	return llmResp, nil
}

package factory

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/gemini"
	"github.com/gm-agent-org/gm-agent/pkg/llm/openai"
)

// NewProvider creates an LLM provider based on configuration.
func NewProvider(ctx context.Context, cfg *config.Config) (llm.Provider, error) {
	active := cfg.ActiveProvider
	if active == "" {
		// Smart Default
		if cfg.Gemini.APIKey != "" {
			active = "gemini"
		} else {
			active = "openai"
		}
	}

	switch active {
	case "gemini":
		return gemini.New(ctx, cfg.Gemini)
	case "openai":
		return openai.New(openai.Config{
			APIKey:  cfg.OpenAI.APIKey,
			BaseURL: cfg.OpenAI.BaseURL,
		}), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", active)
	}
}

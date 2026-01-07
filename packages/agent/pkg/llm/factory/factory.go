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
// Returns the provider instance and the resolved provider ID.
func NewProvider(ctx context.Context, cfg *config.Config) (llm.Provider, string, error) {
	providerID, opts, err := cfg.GetActiveProvider()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get active provider: %w", err)
	}

	provider, err := createProvider(ctx, providerID, opts)
	if err != nil {
		return nil, "", err
	}

	return provider, providerID, nil
}

// createProvider instantiates a provider based on its ID.
func createProvider(ctx context.Context, providerID string, opts config.ProviderOptions) (llm.Provider, error) {
	switch providerID {
	case "gemini":
		return gemini.New(ctx, gemini.Config{
			APIKey:    opts.APIKey,
			ProjectID: opts.ProjectID,
			Location:  opts.Location,
			Model:     opts.Model,
		})
	case "openai", "deepseek":
		return openai.New(openai.Config{
			APIKey:  opts.APIKey,
			BaseURL: opts.BaseURL,
		}), nil
	case "anthropic":
		// TODO: Implement Anthropic provider
		return nil, fmt.Errorf("anthropic provider not yet implemented")
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}
}

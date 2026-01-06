package factory

import (
	"context"
	"os"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/config"
)

func TestNewProviderSelectsOpenAI(t *testing.T) {
	// Set env var for auto-detection
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	cfg := &config.Config{
		ActiveProvider: "openai",
		Providers: map[string]config.ProviderConfig{
			"openai": {
				Options: config.ProviderOptions{
					APIKey: "test-key",
				},
			},
		},
	}
	provider, providerID, err := NewProvider(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected provider, got error %v", err)
	}
	if provider.ID() != "openai" {
		t.Fatalf("expected openai provider, got %s", provider.ID())
	}
	if providerID != "openai" {
		t.Fatalf("expected providerID 'openai', got %s", providerID)
	}
}

func TestNewProviderAutoDetects(t *testing.T) {
	// Set env var for auto-detection
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	cfg := &config.Config{
		Providers: make(map[string]config.ProviderConfig),
	}

	provider, providerID, err := NewProvider(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected provider via auto-detect, got error %v", err)
	}
	if provider.ID() != "gemini" {
		t.Fatalf("expected gemini provider, got %s", provider.ID())
	}
	if providerID != "gemini" {
		t.Fatalf("expected providerID 'gemini', got %s", providerID)
	}
}

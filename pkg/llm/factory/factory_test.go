package factory

import (
	"context"
	"testing"

	"github.com/gm-agent-org/gm-agent/pkg/config"
)

func TestNewProviderSelectsOpenAI(t *testing.T) {
	cfg := &config.Config{ActiveProvider: "openai", OpenAI: config.OpenAIConfig{APIKey: "test"}}
	provider, err := NewProvider(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected provider, got error %v", err)
	}
	if provider.ID() != "openai" {
		t.Fatalf("expected openai provider, got %s", provider.ID())
	}
}

func TestNewProviderDefaults(t *testing.T) {
	cfg := &config.Config{}
	provider, err := NewProvider(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected default provider, got error %v", err)
	}
	if provider.ID() == "" {
		t.Fatalf("expected provider to have id")
	}
}

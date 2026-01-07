package config

import (
	"os"
	"testing"
)

func TestGetActiveProviderFromConfig(t *testing.T) {
	cfg := &Config{
		ActiveProvider: "openai",
		Providers: map[string]ProviderConfig{
			"openai": {
				Options: ProviderOptions{
					APIKey: "test-key",
					Model:  "gpt-4o",
				},
			},
		},
	}

	providerID, opts, err := cfg.GetActiveProvider()
	if err != nil {
		t.Fatalf("expected provider, got error: %v", err)
	}
	if providerID != "openai" {
		t.Fatalf("expected openai, got %s", providerID)
	}
	if opts.APIKey != "test-key" {
		t.Fatalf("expected test-key, got %s", opts.APIKey)
	}
	if opts.Model != "gpt-4o" {
		t.Fatalf("expected gpt-4o, got %s", opts.Model)
	}
}

func TestGetActiveProviderAutoDetect(t *testing.T) {
	// Set env var for auto-detection
	os.Setenv("GEMINI_API_KEY", "env-gemini-key")
	defer os.Unsetenv("GEMINI_API_KEY")

	cfg := &Config{
		Providers: make(map[string]ProviderConfig),
	}

	providerID, opts, err := cfg.GetActiveProvider()
	if err != nil {
		t.Fatalf("expected auto-detect, got error: %v", err)
	}
	if providerID != "gemini" {
		t.Fatalf("expected gemini, got %s", providerID)
	}
	if opts.APIKey != "env-gemini-key" {
		t.Fatalf("expected env-gemini-key, got %s", opts.APIKey)
	}
	// Default model should be applied
	if opts.Model != "gemini-2.0-flash" {
		t.Fatalf("expected gemini-2.0-flash, got %s", opts.Model)
	}
}

func TestMergeOptions(t *testing.T) {
	base := ProviderOptions{
		APIKey:  "base-key",
		BaseURL: "https://base.url",
		Model:   "base-model",
	}
	override := ProviderOptions{
		Model: "override-model",
	}

	result := mergeOptions(base, override)

	if result.APIKey != "base-key" {
		t.Fatalf("expected base-key, got %s", result.APIKey)
	}
	if result.BaseURL != "https://base.url" {
		t.Fatalf("expected base url, got %s", result.BaseURL)
	}
	if result.Model != "override-model" {
		t.Fatalf("expected override-model, got %s", result.Model)
	}
}

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPrefersEnvValues(t *testing.T) {
	cfgDir := t.TempDir()
	cfgPath := filepath.Join(cfgDir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte("active_provider: openai\nopenai:\n  api_key: file-key\nsecurity:\n  auto_approve: false\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("GM_OPENAI_API_KEY", "env-key")
	t.Setenv("GM_ACTIVE_PROVIDER", "openai")
	t.Setenv("GM_SECURITY_AUTO_APPROVE", "true")

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.OpenAI.APIKey != "env-key" {
		t.Fatalf("expected env api key override, got %q", cfg.OpenAI.APIKey)
	}
	if !cfg.Security.AutoApprove {
		t.Fatalf("expected auto approve security override from env")
	}
}

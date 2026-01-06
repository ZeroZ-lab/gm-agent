package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		prepare    func(t *testing.T) string
		env        map[string]string
		wantConfig Config
	}{
		{
			name: "loadFromExplicitPath",
			prepare: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				contents := []byte(`active_provider: openai
openai:
  api_key: file-key
  base_url: http://example.com
  model: gpt-file
security:
  auto_approve: true
  allowed_tools:
    - exec
`)
				if err := os.WriteFile(path, contents, 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
				return path
			},
			wantConfig: Config{
				ActiveProvider: "openai",
				OpenAI: OpenAIConfig{
					APIKey:  "file-key",
					BaseURL: "http://example.com",
					Model:   "gpt-file",
				},
				Security: SecurityConfig{
					AutoApprove:  true,
					AllowedTools: []string{"exec"},
				},
			},
		},
		{
			name: "envOverridesFile",
			prepare: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				contents := []byte(`active_provider: openai
openai:
  api_key: file-key
  base_url: http://example.com
  model: gpt-file
`)
				if err := os.WriteFile(path, contents, 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
				return path
			},
			env: map[string]string{
				"GM_ACTIVE_PROVIDER":       "gemini",
				"GM_OPENAI_API_KEY":        "env-key",
				"GM_OPENAI_BASE_URL":       "http://env",
				"GM_OPENAI_MODEL":          "gpt-env",
				"GM_SECURITY_AUTO_APPROVE": "false",
			},
			wantConfig: Config{
				ActiveProvider: "gemini",
				OpenAI: OpenAIConfig{
					APIKey:  "env-key",
					BaseURL: "http://env",
					Model:   "gpt-env",
				},
				Security: SecurityConfig{
					AutoApprove: false,
				},
			},
		},
		{
			name: "defaultsToLocalConfig",
			prepare: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				contents := []byte(`active_provider: gemini
gemini:
  api_key: gem-key
  model: gem-model
`)
				if err := os.WriteFile(path, contents, 0o644); err != nil {
					t.Fatalf("write config: %v", err)
				}
				return path
			},
			wantConfig: Config{
				ActiveProvider: "gemini",
				Gemini: GeminiConfig{
					APIKey: "gem-key",
					Model:  "gem-model",
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			configPath := ""
			if tt.prepare != nil {
				configPath = tt.prepare(t)
			}

			if tt.name == "defaultsToLocalConfig" {
				origWD, err := os.Getwd()
				if err != nil {
					t.Fatalf("get working dir: %v", err)
				}
				dir := filepath.Dir(configPath)
				if err := os.Chdir(dir); err != nil {
					t.Fatalf("chdir: %v", err)
				}
				t.Cleanup(func() {
					_ = os.Chdir(origWD)
				})

				t.Setenv("HOME", dir)
				configPath = ""
			}

			for key, value := range tt.env {
				t.Setenv(key, value)
			}

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if !reflect.DeepEqual(*cfg, tt.wantConfig) {
				t.Fatalf("unexpected config = %+v, want %+v", *cfg, tt.wantConfig)
			}
		})
	}
}

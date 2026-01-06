package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

type Config struct {
	ActiveProvider string         `yaml:"active_provider" envconfig:"ACTIVE_PROVIDER"` // "openai" or "gemini"
	OpenAI         OpenAIConfig   `yaml:"openai" envconfig:"OPENAI"`
	Gemini         GeminiConfig   `yaml:"gemini" envconfig:"GEMINI"`
	Security       SecurityConfig `yaml:"security" envconfig:"SECURITY"`
}

type OpenAIConfig struct {
	APIKey  string `yaml:"api_key" envconfig:"API_KEY"`
	BaseURL string `yaml:"base_url" envconfig:"BASE_URL"`
	Model   string `yaml:"model" envconfig:"MODEL"`
}

type GeminiConfig struct {
	APIKey    string `yaml:"api_key" envconfig:"API_KEY"`
	ProjectID string `yaml:"project_id" envconfig:"PROJECT_ID"`
	Location  string `yaml:"location" envconfig:"LOCATION"`
	Model     string `yaml:"model" envconfig:"MODEL"`
}

type SecurityConfig struct {
	AutoApprove     bool     `yaml:"auto_approve" envconfig:"AUTO_APPROVE"`
	AllowedTools    []string `yaml:"allowed_tools" envconfig:"ALLOWED_TOOLS"`
	AllowFileSystem bool     `yaml:"allow_fs" envconfig:"ALLOW_FS"`
	AllowInternet   bool     `yaml:"allow_net" envconfig:"ALLOW_NET"`
	WorkspaceRoot   string   `yaml:"workspace_root" envconfig:"WORKSPACE_ROOT"`
}

// Load reads configuration from the specified path, or defaults if path is empty.
// It prioritizes:
// 1. Env Vars (handled by caller currently, or we can merge here)
// 2. Config File
// Load reads configuration from the specified path, or defaults if path is empty.
// It prioritizes:
// 1. Env Vars (handled by caller or loaded from .env)
// 2. Config File
func Load(path string) (*Config, error) {
	// Try loading .env files (ignore error if not present)
	_ = godotenv.Load(".env.local")
	_ = godotenv.Load(".env")

	if path == "" {
		// Try default locations
		home, err := os.UserHomeDir()
		if err == nil {
			defaultPath := filepath.Join(home, ".gm-agent", "config.yaml")
			if _, err := os.Stat(defaultPath); err == nil {
				path = defaultPath
			}
		}

		// Try local directory config.yaml
		localPath := "config.yaml"
		if _, err := os.Stat(localPath); err == nil {
			path = localPath
		}
	}

	cfg := &Config{}

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Process Env Vars (GM_ prefix)
	// This will override values from config file if set in Env
	if err := envconfig.Process("GM", cfg); err != nil {
		return nil, fmt.Errorf("failed to process env vars: %w", err)
	}

	return cfg, nil
}

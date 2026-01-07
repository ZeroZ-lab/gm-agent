package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

// ProviderConfig represents configuration for a single LLM provider.
type ProviderConfig struct {
	Options ProviderOptions `yaml:"options" json:"options"`
}

// ProviderOptions contains the SDK-level options for a provider.
type ProviderOptions struct {
	APIKey      string  `yaml:"apiKey" json:"apiKey" envconfig:"API_KEY"`
	BaseURL     string  `yaml:"baseURL" json:"baseURL" envconfig:"BASE_URL"`
	Model       string  `yaml:"model" json:"model" envconfig:"MODEL"`
	ProjectID   string  `yaml:"projectID" json:"projectID" envconfig:"PROJECT_ID"`   // For Vertex AI
	Location    string  `yaml:"location" json:"location" envconfig:"LOCATION"`       // For Vertex AI
	Timeout     int     `yaml:"timeout" json:"timeout" envconfig:"TIMEOUT"`          // Request timeout in ms
	Temperature float64 `yaml:"temperature" json:"temperature" envconfig:"TEMP"`     // Sampling temperature
	MaxTokens   int     `yaml:"max_tokens" json:"max_tokens" envconfig:"MAX_TOKENS"` // Max tokens to generate
}

// SecurityConfig contains security-related settings.
type SecurityConfig struct {
	AutoApprove     bool     `yaml:"auto_approve" envconfig:"AUTO_APPROVE"`
	AllowedTools    []string `yaml:"allowed_tools" envconfig:"ALLOWED_TOOLS"`
	AllowFileSystem bool     `yaml:"allow_fs" envconfig:"ALLOW_FS"`
	AllowInternet   bool     `yaml:"allow_net" envconfig:"ALLOW_NET"`
	WorkspaceRoot   string   `yaml:"workspace_root" envconfig:"WORKSPACE_ROOT"`
}

// HTTPConfig contains HTTP API related settings.
type HTTPConfig struct {
	Enable bool   `yaml:"enable" envconfig:"ENABLE"`
	Addr   string `yaml:"addr" envconfig:"ADDR"`
	APIKey string `yaml:"api_key" envconfig:"API_KEY"`
}

// Config is the root configuration structure.
type Config struct {
	// ActiveProvider explicitly sets the active provider (optional).
	// If not set, auto-detection is used based on available API keys.
	ActiveProvider string `yaml:"active_provider" envconfig:"ACTIVE_PROVIDER"`

	// LogLevel controls structured logging verbosity (DEBUG, VERBOSE, INFO, WARNING, ERROR).
	LogLevel string `yaml:"log_level" envconfig:"LOG_LEVEL"`

	// Providers is a map of provider ID to its configuration.
	Providers map[string]ProviderConfig `yaml:"provider"`

	// Security settings.
	Security SecurityConfig `yaml:"security" envconfig:"SECURITY"`

	// HTTP server settings.
	HTTP HTTPConfig `yaml:"http" envconfig:"HTTP"`

	// DevMode enables development features like Swagger UI.
	DevMode bool `yaml:"dev_mode" envconfig:"DEV_MODE"`
}

// ProviderEnvVars maps provider IDs to their environment variable names for auto-detection.
// The first env var in the list that is set will be used.
var ProviderEnvVars = map[string]struct {
	APIKey  []string
	BaseURL []string
	Model   []string
}{
	"gemini": {
		APIKey: []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"},
		Model:  []string{"GEMINI_MODEL"},
	},
	"openai": {
		APIKey:  []string{"OPENAI_API_KEY"},
		BaseURL: []string{"OPENAI_API_BASE", "OPENAI_BASE_URL"},
		Model:   []string{"OPENAI_MODEL"},
	},
	"anthropic": {
		APIKey: []string{"ANTHROPIC_API_KEY"},
		Model:  []string{"ANTHROPIC_MODEL"},
	},
	"deepseek": {
		APIKey: []string{"DEEPSEEK_API_KEY"},
		Model:  []string{"DEEPSEEK_MODEL"},
	},
}

// ProviderDefaults contains default options for each provider.
var ProviderDefaults = map[string]ProviderOptions{
	"gemini": {
		Model: "gemini-2.0-flash",
	},
	"openai": {
		BaseURL: "https://api.openai.com/v1",
		Model:   "gpt-4o",
	},
	"deepseek": {
		BaseURL: "https://api.deepseek.com",
		Model:   "deepseek-chat",
	},
	"anthropic": {
		Model: "claude-sonnet-4-20250514",
	},
}

// GetActiveProvider returns the active provider ID and its configuration.
// Priority: ActiveProvider field > First provider with API key in env > First configured provider.
func (c *Config) GetActiveProvider() (string, ProviderOptions, error) {
	// 1. Explicit ActiveProvider
	if c.ActiveProvider != "" {
		if p, ok := c.Providers[c.ActiveProvider]; ok {
			opts := mergeOptions(ProviderDefaults[c.ActiveProvider], p.Options)
			return c.ActiveProvider, opts, nil
		}
		// Check if we can auto-configure from env
		if opts, ok := c.detectProviderFromEnv(c.ActiveProvider); ok {
			return c.ActiveProvider, opts, nil
		}
		return "", ProviderOptions{}, fmt.Errorf("active provider %q not configured", c.ActiveProvider)
	}

	// 2. Auto-detect from environment variables (ordered: gemini first)
	for _, providerID := range []string{"gemini", "openai", "deepseek", "anthropic"} {
		envVars, ok := ProviderEnvVars[providerID]
		if !ok {
			continue
		}
		// Check if any API key env var is set
		var apiKey string
		for _, envVar := range envVars.APIKey {
			if v := os.Getenv(envVar); v != "" {
				apiKey = v
				break
			}
		}
		if apiKey == "" {
			continue
		}

		opts := ProviderDefaults[providerID]
		opts.APIKey = apiKey

		// Check for BaseURL env var
		for _, envVar := range envVars.BaseURL {
			if v := os.Getenv(envVar); v != "" {
				opts.BaseURL = v
				break
			}
		}

		// Check for Model env var
		for _, envVar := range envVars.Model {
			if v := os.Getenv(envVar); v != "" {
				opts.Model = v
				break
			}
		}

		// Merge with config if exists
		if p, ok := c.Providers[providerID]; ok {
			opts = mergeOptions(opts, p.Options)
		}
		return providerID, opts, nil
	}

	// 3. First configured provider with API key
	for providerID, p := range c.Providers {
		if p.Options.APIKey != "" {
			opts := mergeOptions(ProviderDefaults[providerID], p.Options)
			return providerID, opts, nil
		}
	}

	return "", ProviderOptions{}, fmt.Errorf("no provider configured or detected")
}

// detectProviderFromEnv checks if a provider can be configured from environment variables.
func (c *Config) detectProviderFromEnv(providerID string) (ProviderOptions, bool) {
	envVars, ok := ProviderEnvVars[providerID]
	if !ok {
		return ProviderOptions{}, false
	}

	// Check if any API key env var is set
	var apiKey string
	for _, envVar := range envVars.APIKey {
		if v := os.Getenv(envVar); v != "" {
			apiKey = v
			break
		}
	}
	if apiKey == "" {
		return ProviderOptions{}, false
	}

	opts := ProviderDefaults[providerID]
	opts.APIKey = apiKey

	// Check for BaseURL env var
	for _, envVar := range envVars.BaseURL {
		if v := os.Getenv(envVar); v != "" {
			opts.BaseURL = v
			break
		}
	}

	// Check for Model env var
	for _, envVar := range envVars.Model {
		if v := os.Getenv(envVar); v != "" {
			opts.Model = v
			break
		}
	}

	// Merge with config if exists
	if p, ok := c.Providers[providerID]; ok {
		opts = mergeOptions(opts, p.Options)
	}

	return opts, true
}

// mergeOptions merges two ProviderOptions, with 'override' taking precedence.
func mergeOptions(base, override ProviderOptions) ProviderOptions {
	result := base
	if override.APIKey != "" {
		result.APIKey = override.APIKey
	}
	if override.BaseURL != "" {
		result.BaseURL = override.BaseURL
	}
	if override.Model != "" {
		result.Model = override.Model
	}
	if override.ProjectID != "" {
		result.ProjectID = override.ProjectID
	}
	if override.Location != "" {
		result.Location = override.Location
	}
	if override.Timeout != 0 {
		result.Timeout = override.Timeout
	}
	// Note: 0 is valid for Temperature, but here we assume config override implies non-zero or specific intent.
	// Since 0.0 is default, we can't easily distinguish "unset" from "zero" without pointers.
	// For MVP, if override > 0, we use it. If exactly 0, we keep base.
	// Users needing exactly 0.0 might need a pointer refactor later.
	if override.Temperature > 0 {
		result.Temperature = override.Temperature
	}
	if override.MaxTokens != 0 {
		result.MaxTokens = override.MaxTokens
	}
	return result
}

// Load reads configuration from the specified path, or defaults if path is empty.
// Priority: Env Vars > Config File > Defaults
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

	cfg := &Config{
		Providers: make(map[string]ProviderConfig),
	}

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

	// Apply Defaults
	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}

	return cfg, nil
}

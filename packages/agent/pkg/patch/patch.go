package patch

import (
	"context"
	"fmt"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// Engine handles file patching operations
type Engine interface {
	// GenerateDiff creates a unified diff between old and new content
	GenerateDiff(ctx context.Context, filePath, oldContent, newContent string) (string, error)

	// Apply applies a patch to a file
	// Returns the patch ID for potential rollback
	Apply(ctx context.Context, cmd types.ApplyPatchCommand) (*ApplyResult, error)

	// DryRun previews the changes without modifying files
	DryRun(ctx context.Context, cmd types.ApplyPatchCommand) (*ApplyResult, error)

	// Rollback restores a file from backup using patch ID
	Rollback(ctx context.Context, patchID string) error

	// ListBackups returns all available backups
	ListBackups() ([]BackupInfo, error)
}

// ApplyResult contains the result of a patch operation
type ApplyResult struct {
	PatchID     string   `json:"patch_id"`
	FilePath    string   `json:"file_path"`
	Success     bool     `json:"success"`
	Diff        string   `json:"diff"`
	LinesAdded  int      `json:"lines_added"`
	LinesRemoved int     `json:"lines_removed"`
	BackupPath  string   `json:"backup_path,omitempty"`
	Error       string   `json:"error,omitempty"`
	Warnings    []string `json:"warnings,omitempty"`
}

// BackupInfo represents a backup entry
type BackupInfo struct {
	PatchID    string `json:"patch_id"`
	FilePath   string `json:"file_path"`
	BackupPath string `json:"backup_path"`
	Timestamp  string `json:"timestamp"`
	DiffPreview string `json:"diff_preview"` // First few lines
}

// Config for patch engine
type Config struct {
	// WorkDir is the root directory for all file operations
	WorkDir string
	// BackupDir is where backups are stored
	BackupDir string
	// MaxContextLines for diff generation (default: 3)
	MaxContextLines int
	// AllowedPaths restricts patch operations to specific paths
	AllowedPaths []string
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		WorkDir:         ".",
		BackupDir:       ".gm-backups",
		MaxContextLines: 3,
		AllowedPaths:    []string{},
	}
}

// Validate checks if the configuration is valid
func (c Config) Validate() error {
	if c.WorkDir == "" {
		return fmt.Errorf("work directory cannot be empty")
	}
	if c.BackupDir == "" {
		return fmt.Errorf("backup directory cannot be empty")
	}
	if c.MaxContextLines < 0 {
		return fmt.Errorf("max context lines cannot be negative")
	}
	return nil
}

// NewEngine creates a new patch engine instance
func NewEngine(cfg Config) (Engine, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	return &engine{
		cfg: cfg,
	}, nil
}

// engine is the default implementation
type engine struct {
	cfg Config
}

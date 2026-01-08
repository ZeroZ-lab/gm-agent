package patch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Rollback restores a file from backup using patch ID
func (e *engine) Rollback(ctx context.Context, patchID string) error {
	backupDir := filepath.Join(e.cfg.WorkDir, e.cfg.BackupDir)

	// Find the backup file with this patch ID
	pattern := filepath.Join(backupDir, patchID+"_*.bak")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to search for backup: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Errorf("no backup found for patch ID: %s", patchID)
	}

	// Use the first match (should only be one)
	backupPath := matches[0]

	// Read metadata to get original file path
	metadataPath := backupPath + ".meta"
	metadata, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read backup metadata: %w", err)
	}

	// Parse metadata to get original file path
	filePath := ""
	for _, line := range strings.Split(string(metadata), "\n") {
		if strings.HasPrefix(line, "file_path: ") {
			filePath = strings.TrimPrefix(line, "file_path: ")
			break
		}
	}

	if filePath == "" {
		return fmt.Errorf("file path not found in metadata")
	}

	// Read backup content
	content, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Restore the file
	if err := e.writeFile(filePath, string(content)); err != nil {
		return fmt.Errorf("failed to restore file: %w", err)
	}

	return nil
}

// ListBackups returns all available backups
func (e *engine) ListBackups() ([]BackupInfo, error) {
	backupDir := filepath.Join(e.cfg.WorkDir, e.cfg.BackupDir)

	// Check if backup directory exists
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}

	// Find all .bak files
	pattern := filepath.Join(backupDir, "*.bak")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	backups := make([]BackupInfo, 0, len(matches))
	for _, backupPath := range matches {
		// Read metadata
		metadataPath := backupPath + ".meta"
		metadata, err := os.ReadFile(metadataPath)
		if err != nil {
			// Skip if metadata is missing
			continue
		}

		info := BackupInfo{
			BackupPath: backupPath,
		}

		// Parse metadata
		for _, line := range strings.Split(string(metadata), "\n") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) != 2 {
				continue
			}
			key, value := parts[0], parts[1]

			switch key {
			case "patch_id":
				info.PatchID = value
			case "file_path":
				info.FilePath = value
			case "timestamp":
				info.Timestamp = value
			}
		}

		// Read first few lines as preview
		content, err := os.ReadFile(backupPath)
		if err == nil {
			lines := strings.Split(string(content), "\n")
			previewLines := 5
			if len(lines) < previewLines {
				previewLines = len(lines)
			}
			info.DiffPreview = strings.Join(lines[:previewLines], "\n")
		}

		backups = append(backups, info)
	}

	return backups, nil
}

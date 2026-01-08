package patch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/types"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// Apply applies a patch to a file with backup
func (e *engine) Apply(ctx context.Context, cmd types.ApplyPatchCommand) (*ApplyResult, error) {
	// Validate file path
	if err := e.validatePath(cmd.FilePath); err != nil {
		return nil, fmt.Errorf("invalid file path: %w", err)
	}

	// Generate patch ID
	patchID := types.GeneratePatchID()

	// Read current file content
	currentContent, err := e.readFile(cmd.FilePath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Create backup before applying
	backupPath := ""
	if !cmd.DryRun {
		backupPath, err = e.createBackup(patchID, cmd.FilePath, currentContent)
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Apply the patch
	dmp := diffmatchpatch.New()
	patches, err := dmp.PatchFromText(cmd.Diff)
	if err != nil {
		return nil, fmt.Errorf("failed to parse diff: %w", err)
	}

	newContent, results := dmp.PatchApply(patches, currentContent)

	// Check if all patches applied successfully
	var warnings []string
	allSuccess := true
	for i, applied := range results {
		if !applied {
			allSuccess = false
			warnings = append(warnings, fmt.Sprintf("hunk %d failed to apply", i+1))
		}
	}

	// Count changes
	added, removed := countChanges(cmd.Diff)

	result := &ApplyResult{
		PatchID:      patchID,
		FilePath:     cmd.FilePath,
		Success:      allSuccess,
		Diff:         cmd.Diff,
		LinesAdded:   added,
		LinesRemoved: removed,
		BackupPath:   backupPath,
		Warnings:     warnings,
	}

	if !allSuccess {
		result.Error = "some hunks failed to apply"
	}

	// Write the new content if not dry-run and successful
	if !cmd.DryRun && allSuccess {
		if err := e.writeFile(cmd.FilePath, newContent); err != nil {
			result.Success = false
			result.Error = fmt.Sprintf("failed to write file: %v", err)
			return result, fmt.Errorf("failed to write file: %w", err)
		}

		// Record file change for checkpoint tracking
		operation := "modify"
		if currentContent == "" {
			operation = "create"
		}
		e.tracker.Record(types.FileChange{
			PatchID:    patchID,
			FilePath:   cmd.FilePath,
			BackupPath: backupPath,
			Operation:  operation,
		})
	}

	return result, nil
}

// DryRun previews the changes without modifying files
func (e *engine) DryRun(ctx context.Context, cmd types.ApplyPatchCommand) (*ApplyResult, error) {
	cmd.DryRun = true
	return e.Apply(ctx, cmd)
}

// validatePath ensures the file path is within allowed boundaries
func (e *engine) validatePath(filePath string) error {
	// Clean the path to prevent traversal attacks
	cleaned := filepath.Clean(filePath)

	// Prevent absolute paths outside work directory
	absPath := cleaned
	if !filepath.IsAbs(cleaned) {
		absPath = filepath.Join(e.cfg.WorkDir, cleaned)
	}

	// Ensure it's within work directory
	relPath, err := filepath.Rel(e.cfg.WorkDir, absPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Prevent path traversal (../)
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path traversal detected: %s", filePath)
	}

	// Check allowed paths if configured
	if len(e.cfg.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range e.cfg.AllowedPaths {
			if strings.HasPrefix(relPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed list: %s", filePath)
		}
	}

	return nil
}

// readFile safely reads a file with validation
func (e *engine) readFile(filePath string) (string, error) {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(e.cfg.WorkDir, filePath)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// writeFile safely writes a file with atomic operation
func (e *engine) writeFile(filePath, content string) error {
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(e.cfg.WorkDir, filePath)
	}

	// Ensure directory exists
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Atomic write: write to temp file then rename
	tmpPath := absPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		os.Remove(tmpPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// createBackup creates a backup of the file
func (e *engine) createBackup(patchID, filePath, content string) (string, error) {
	// Create backup directory if not exists
	backupDir := filepath.Join(e.cfg.WorkDir, e.cfg.BackupDir)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	backupName := fmt.Sprintf("%s_%s_%s.bak", patchID, timestamp, filepath.Base(filePath))
	backupPath := filepath.Join(backupDir, backupName)

	// Write backup file
	if err := os.WriteFile(backupPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	// Also write metadata
	metadataPath := backupPath + ".meta"
	metadata := fmt.Sprintf("patch_id: %s\nfile_path: %s\ntimestamp: %s\n",
		patchID, filePath, timestamp)
	if err := os.WriteFile(metadataPath, []byte(metadata), 0644); err != nil {
		// Non-fatal, just log
		fmt.Fprintf(os.Stderr, "warning: failed to write backup metadata: %v\n", err)
	}

	return backupPath, nil
}

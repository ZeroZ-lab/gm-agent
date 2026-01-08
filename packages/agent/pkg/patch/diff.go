package patch

import (
	"context"
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// GenerateDiff creates a unified diff between old and new content
func (e *engine) GenerateDiff(ctx context.Context, filePath, oldContent, newContent string) (string, error) {
	// Check for binary content
	if isBinary(oldContent) || isBinary(newContent) {
		return "", fmt.Errorf("binary files are not supported")
	}

	// Use diffmatchpatch library for generating diffs
	dmp := diffmatchpatch.New()

	// Calculate diffs
	diffs := dmp.DiffMain(oldContent, newContent, false)

	// Optimize diffs for readability
	diffs = dmp.DiffCleanupSemantic(diffs)

	// Check if there are any changes
	hasChanges := false
	for _, diff := range diffs {
		if diff.Type != diffmatchpatch.DiffEqual {
			hasChanges = true
			break
		}
	}

	if !hasChanges {
		return "", fmt.Errorf("no changes detected")
	}

	// Convert to unified diff format
	patches := dmp.PatchMake(oldContent, diffs)
	if len(patches) == 0 {
		return "", fmt.Errorf("no changes detected")
	}

	// Return just the patch text without custom header
	// The diffmatchpatch library already includes appropriate headers
	return dmp.PatchToText(patches), nil
}

// isBinary checks if content contains binary data
// Simple heuristic: check for null bytes in first 8KB
func isBinary(content string) bool {
	checkLen := 8192
	if len(content) < checkLen {
		checkLen = len(content)
	}

	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return true
		}
	}
	return false
}

// parseDiff parses a unified diff and extracts hunks
func parseDiff(diff string) ([]diffHunk, error) {
	lines := strings.Split(diff, "\n")
	var hunks []diffHunk
	var currentHunk *diffHunk

	for _, line := range lines {
		if strings.HasPrefix(line, "@@") {
			// New hunk
			if currentHunk != nil {
				hunks = append(hunks, *currentHunk)
			}
			currentHunk = &diffHunk{
				header: line,
				lines:  []string{},
			}
		} else if currentHunk != nil {
			currentHunk.lines = append(currentHunk.lines, line)
		}
	}

	if currentHunk != nil {
		hunks = append(hunks, *currentHunk)
	}

	return hunks, nil
}

// diffHunk represents a single hunk in a unified diff
type diffHunk struct {
	header string   // @@ -1,5 +1,6 @@
	lines  []string // actual diff lines
}

// countChanges analyzes a diff and returns lines added/removed
func countChanges(diff string) (added, removed int) {
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			removed++
		}
	}
	return
}

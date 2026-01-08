package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gm-agent-org/gm-agent/pkg/types"
)

// GlobTool searches for files matching a pattern
var GlobTool = types.Tool{
	Name:        "glob",
	Description: "Search for files matching a glob pattern (e.g., '**/*.go', 'src/**/*.ts'). Returns list of matching file paths.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match files (e.g., '**/*.go', 'pkg/**/*.json')",
			},
			"base_dir": map[string]any{
				"type":        "string",
				"description": "Base directory to search from (default: current directory)",
				"default":     ".",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results to return (default: 100)",
				"default":     100,
			},
		},
		"required": []string{"pattern"},
	},
	Metadata: map[string]string{
		"category": "search",
	},
}

// GrepTool searches for content within files
var GrepTool = types.Tool{
	Name:        "grep",
	Description: "Search for text patterns in files. Supports regular expressions. Returns matching lines with file paths and line numbers.",
	Parameters: types.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Text or regex pattern to search for",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "File or directory to search in (default: current directory)",
				"default":     ".",
			},
			"file_pattern": map[string]any{
				"type":        "string",
				"description": "Only search in files matching this glob pattern (e.g., '*.go')",
				"default":     "*",
			},
			"case_sensitive": map[string]any{
				"type":        "boolean",
				"description": "Whether the search is case sensitive (default: false)",
				"default":     false,
			},
			"max_results": map[string]any{
				"type":        "integer",
				"description": "Maximum number of matches to return (default: 50)",
				"default":     50,
			},
			"context_lines": map[string]any{
				"type":        "integer",
				"description": "Number of context lines to show before and after match (default: 0)",
				"default":     0,
			},
		},
		"required": []string{"pattern"},
	},
	Metadata: map[string]string{
		"category": "search",
	},
}

type GlobArgs struct {
	Pattern    string `json:"pattern"`
	BaseDir    string `json:"base_dir"`
	MaxResults int    `json:"max_results"`
}

func HandleGlob(ctx context.Context, argsJSON string) (string, error) {
	var args GlobArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	if args.BaseDir == "" {
		args.BaseDir = "."
	}

	if args.MaxResults <= 0 {
		args.MaxResults = 100
	}

	// Convert glob pattern to regex for custom matching
	var matches []string
	count := 0

	// Use filepath.Walk to traverse directories
	err := filepath.Walk(args.BaseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if path matches pattern
		relPath, _ := filepath.Rel(args.BaseDir, path)
		matched, _ := filepath.Match(args.Pattern, filepath.Base(path))

		// Also support ** patterns
		if !matched {
			matched = matchDoubleStarPattern(relPath, args.Pattern)
		}

		if matched {
			matches = append(matches, relPath)
			count++
			if count >= args.MaxResults {
				return filepath.SkipAll
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "No files found matching pattern", nil
	}

	result := fmt.Sprintf("Found %d file(s):\n", len(matches))
	for _, match := range matches {
		result += fmt.Sprintf("  %s\n", match)
	}

	if count >= args.MaxResults {
		result += fmt.Sprintf("\n(Limited to %d results)", args.MaxResults)
	}

	return result, nil
}

type GrepArgs struct {
	Pattern       string `json:"pattern"`
	Path          string `json:"path"`
	FilePattern   string `json:"file_pattern"`
	CaseSensitive bool   `json:"case_sensitive"`
	MaxResults    int    `json:"max_results"`
	ContextLines  int    `json:"context_lines"`
}

type GrepMatch struct {
	File       string
	LineNumber int
	Line       string
	Context    []string
}

func HandleGrep(ctx context.Context, argsJSON string) (string, error) {
	var args GrepArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("invalid arguments: %w", err)
	}

	if args.Pattern == "" {
		return "", fmt.Errorf("pattern is required")
	}

	if args.Path == "" {
		args.Path = "."
	}

	if args.FilePattern == "" {
		args.FilePattern = "*"
	}

	if args.MaxResults <= 0 {
		args.MaxResults = 50
	}

	// Compile regex
	regexFlags := ""
	if !args.CaseSensitive {
		regexFlags = "(?i)"
	}
	re, err := regexp.Compile(regexFlags + args.Pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	var matches []GrepMatch
	matchCount := 0

	// Walk through files
	err = filepath.Walk(args.Path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		// Check file pattern
		matched, _ := filepath.Match(args.FilePattern, filepath.Base(path))
		if !matched {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		// Skip binary files
		if isBinaryContent(content) {
			return nil
		}

		// Search in file
		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				match := GrepMatch{
					File:       path,
					LineNumber: i + 1,
					Line:       line,
				}

				// Add context lines
				if args.ContextLines > 0 {
					start := max(0, i-args.ContextLines)
					end := min(len(lines), i+args.ContextLines+1)
					match.Context = lines[start:end]
				}

				matches = append(matches, match)
				matchCount++

				if matchCount >= args.MaxResults {
					return filepath.SkipAll
				}
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(matches) == 0 {
		return "No matches found", nil
	}

	// Format results
	result := fmt.Sprintf("Found %d match(es):\n\n", len(matches))
	for _, match := range matches {
		result += fmt.Sprintf("%s:%d: %s\n", match.File, match.LineNumber, strings.TrimSpace(match.Line))

		if len(match.Context) > 0 {
			result += "---\n"
			for _, ctxLine := range match.Context {
				result += fmt.Sprintf("  %s\n", ctxLine)
			}
			result += "---\n"
		}
	}

	if matchCount >= args.MaxResults {
		result += fmt.Sprintf("\n(Limited to %d results)", args.MaxResults)
	}

	return result, nil
}

// Helper functions

func matchDoubleStarPattern(path, pattern string) bool {
	// Simple ** pattern matching
	// Convert ** to regex .*
	regexPattern := strings.ReplaceAll(pattern, "**", ".*")
	regexPattern = strings.ReplaceAll(regexPattern, "*", "[^/]*")
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(path)
}

func isBinaryContent(data []byte) bool {
	// Check first 8KB for null bytes
	checkLen := 8192
	if len(data) < checkLen {
		checkLen = len(data)
	}

	for i := 0; i < checkLen; i++ {
		if data[i] == 0 {
			return true
		}
	}
	return false
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

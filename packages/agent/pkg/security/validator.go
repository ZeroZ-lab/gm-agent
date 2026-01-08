package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathValidator validates file paths for security
type PathValidator struct {
	workDir      string
	allowedPaths []string
}

// NewPathValidator creates a new path validator
func NewPathValidator(workDir string, allowedPaths []string) *PathValidator {
	return &PathValidator{
		workDir:      workDir,
		allowedPaths: allowedPaths,
	}
}

// ValidatePath ensures a path is safe and within allowed boundaries
func (v *PathValidator) ValidatePath(path string) error {
	// Clean the path to prevent traversal attacks
	cleaned := filepath.Clean(path)

	// Check for absolute paths outside work directory
	absPath := cleaned
	if !filepath.IsAbs(cleaned) {
		absPath = filepath.Join(v.workDir, cleaned)
	}

	// Ensure it's within work directory
	relPath, err := filepath.Rel(v.workDir, absPath)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Prevent path traversal (../)
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("path traversal detected: %s", path)
	}

	// Check for suspicious patterns
	if containsSuspiciousPattern(cleaned) {
		return fmt.Errorf("suspicious path pattern detected: %s", path)
	}

	// Check allowed paths if configured
	if len(v.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range v.allowedPaths {
			if strings.HasPrefix(relPath, allowedPath) || relPath == allowedPath {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed list: %s", path)
		}
	}

	return nil
}

// containsSuspiciousPattern checks for dangerous path patterns
func containsSuspiciousPattern(path string) bool {
	suspiciousPatterns := []string{
		"/../",      // Path traversal
		"/etc/",     // System config
		"/proc/",    // Process info
		"/sys/",     // System info
		"/dev/",     // Devices
		"/root/",    // Root home
		".ssh/",     // SSH keys
		".aws/",     // AWS credentials
		".gcp/",     // GCP credentials
		"id_rsa",    // SSH private key
		"id_dsa",    // SSH private key
		"id_ecdsa",  // SSH private key
		"id_ed25519",// SSH private key
		".env",      // Environment vars
		"credentials", // Generic credentials
		"secrets",   // Secrets
	}

	lowerPath := strings.ToLower(path)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(lowerPath, pattern) {
			return true
		}
	}

	return false
}

// CommandValidator validates shell commands
type CommandValidator struct {
	blockedCommands []string
	blockedPatterns []string
}

// NewCommandValidator creates a new command validator
func NewCommandValidator() *CommandValidator {
	return &CommandValidator{
		blockedCommands: []string{
			"rm -rf /",
			"dd if=/dev/zero",
			"fork bomb",
			":(){ :|:& };:",
		},
		blockedPatterns: []string{
			"rm.*-rf.*/$", // rm -rf ending with /
			"chmod.*777",  // Overly permissive permissions
			"curl.*\\|.*sh", // Piping curl to shell
			"wget.*\\|.*sh", // Piping wget to shell
		},
	}
}

// ValidateCommand checks if a shell command is safe to execute
func (v *CommandValidator) ValidateCommand(cmd string) error {
	lowerCmd := strings.ToLower(strings.TrimSpace(cmd))

	// Check blocked commands
	for _, blocked := range v.blockedCommands {
		if strings.Contains(lowerCmd, strings.ToLower(blocked)) {
			return fmt.Errorf("blocked dangerous command pattern: %s", blocked)
		}
	}

	// Check for suspicious command injection patterns
	if containsCommandInjection(cmd) {
		return fmt.Errorf("potential command injection detected")
	}

	return nil
}

// containsCommandInjection checks for command injection patterns
func containsCommandInjection(cmd string) bool {
	injectionPatterns := []string{
		";",  // Command separator
		"&&", // Command chaining
		"||", // Command chaining
		"|",  // Pipe (can be legitimate, but risky)
		"`",  // Command substitution
		"$(",  // Command substitution
		">",  // Redirection
		"<",  // Redirection
	}

	// Note: This is overly restrictive for a real shell
	// In production, you'd want a more sophisticated parser
	// For now, we allow some patterns but warn about them

	dangerousCount := 0
	for _, pattern := range injectionPatterns {
		if strings.Contains(cmd, pattern) {
			dangerousCount++
		}
	}

	// If multiple dangerous patterns, likely injection
	return dangerousCount >= 3
}

// ResourceLimits defines resource constraints for command execution
type ResourceLimits struct {
	MaxExecutionTime int64 // seconds
	MaxMemory        int64 // bytes
	MaxFileSize      int64 // bytes
}

// DefaultResourceLimits returns safe default limits
func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxExecutionTime: 300,           // 5 minutes
		MaxMemory:        1024 * 1024 * 1024, // 1 GB
		MaxFileSize:      100 * 1024 * 1024,  // 100 MB
	}
}

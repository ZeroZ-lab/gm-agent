package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette (Claude Code inspired)
var (
	// Primary colors
	colorPrimary   = lipgloss.Color("#FF6B35") // Orange accent
	colorSecondary = lipgloss.Color("#7C3AED") // Purple
	colorSuccess   = lipgloss.Color("#10B981") // Green
	colorWarning   = lipgloss.Color("#F59E0B") // Yellow
	colorError     = lipgloss.Color("#EF4444") // Red
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorText      = lipgloss.Color("#E5E7EB") // Light gray

	// Styles
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary)

	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#000")).
			Background(colorPrimary).
			Padding(0, 1)

	styleGitBranch = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	styleDir = lipgloss.NewStyle().
			Foreground(colorText)

	styleCommand = lipgloss.NewStyle().
			Foreground(colorWarning).
			Italic(true)

	styleToolCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	styleToolName = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorSecondary)

	styleToolArg = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleUserLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#3B82F6"))

	styleAssistantLabel = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary)

	styleSystemMsg = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(colorText).
			Padding(0, 1)

	styleStatusItem = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginRight(2)

	styleStatusValue = lipgloss.NewStyle().
			Foreground(colorText)

	stylePermissionBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorWarning).
				Padding(1, 2).
				MarginTop(1).
				MarginBottom(1)
)

const version = "0.1.0"

// WelcomeInfo holds information displayed at startup
type WelcomeInfo struct {
	Version   string
	WorkDir   string
	GitBranch string
	GitDirty  bool
}

// GetWelcomeInfo gathers system information for the welcome screen
func GetWelcomeInfo() WelcomeInfo {
	info := WelcomeInfo{
		Version: version,
	}

	// Get working directory
	if wd, err := os.Getwd(); err == nil {
		info.WorkDir = wd
	}

	// Get git branch
	if branch, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		info.GitBranch = strings.TrimSpace(string(branch))
	}

	// Check if git is dirty
	if status, err := exec.Command("git", "status", "--porcelain").Output(); err == nil {
		info.GitDirty = len(strings.TrimSpace(string(status))) > 0
	}

	return info
}

// RenderWelcome renders the welcome screen
func RenderWelcome(info WelcomeInfo) string {
	var b strings.Builder

	// Logo / Title
	logo := `
   ____  __  __            _    ____  _____ _   _ _____
  / ___|  \/  |          / \  / ___|| ____| \ | |_   _|
 | |  _| |\/| |  _____  / _ \| |  _ |  _| |  \| | | |
 | |_| | |  | | |_____| / ___ \ |_| | |___| |\  | | |
  \____|_|  |_|        /_/   \_\____|_____|_| \_| |_|
`
	b.WriteString(styleTitle.Render(logo))
	b.WriteString("\n")

	// Version badge
	b.WriteString(styleBadge.Render(fmt.Sprintf("v%s", info.Version)))
	b.WriteString("  ")
	b.WriteString(styleSubtitle.Render("Enterprise AI Agent Runtime"))
	b.WriteString("\n\n")

	// Working directory
	if info.WorkDir != "" {
		shortDir := shortenPath(info.WorkDir)
		b.WriteString(styleSubtitle.Render("  Working in: "))
		b.WriteString(styleDir.Render(shortDir))
		b.WriteString("\n")
	}

	// Git info
	if info.GitBranch != "" {
		b.WriteString(styleSubtitle.Render("  Git branch: "))
		b.WriteString(styleGitBranch.Render(info.GitBranch))
		if info.GitDirty {
			b.WriteString(lipgloss.NewStyle().Foreground(colorWarning).Render(" *"))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Quick commands hint
	b.WriteString(styleSubtitle.Render("  Commands: "))
	b.WriteString(styleCommand.Render("/help"))
	b.WriteString(styleSubtitle.Render(" - show help  "))
	b.WriteString(styleCommand.Render("/new"))
	b.WriteString(styleSubtitle.Render(" - new session  "))
	b.WriteString(styleCommand.Render("/exit"))
	b.WriteString(styleSubtitle.Render(" - quit"))
	b.WriteString("\n")

	// Divider
	b.WriteString(styleSubtitle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n")

	return b.String()
}

// RenderStatusBar renders the bottom status bar
func RenderStatusBar(sessionID string, model string, width int) string {
	wd, _ := os.Getwd()
	shortDir := shortenPath(wd)

	left := fmt.Sprintf(" %s  %s",
		styleStatusItem.Render("Session:")+styleStatusValue.Render(truncate(sessionID, 8)),
		styleStatusItem.Render("Dir:")+styleStatusValue.Render(shortDir),
	)

	right := styleStatusItem.Render("Model:") + styleStatusValue.Render(model) + " "

	// Calculate padding
	padding := width - lipgloss.Width(left) - lipgloss.Width(right)
	if padding < 0 {
		padding = 0
	}

	return styleStatusBar.Width(width).Render(left + strings.Repeat(" ", padding) + right)
}

// RenderToolCall renders a tool call in card style
func RenderToolCall(toolName string, args map[string]interface{}, status string) string {
	var b strings.Builder

	// Tool name header
	icon := "🔧"
	if status == "running" {
		icon = "⏳"
	} else if status == "success" {
		icon = "✅"
	} else if status == "error" {
		icon = "❌"
	}

	b.WriteString(fmt.Sprintf("%s %s", icon, styleToolName.Render(toolName)))
	b.WriteString("\n")

	// Arguments (compact view)
	for k, v := range args {
		valStr := fmt.Sprintf("%v", v)
		if len(valStr) > 50 {
			valStr = valStr[:47] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s: %s\n",
			styleToolArg.Render(k),
			styleDir.Render(valStr)))
	}

	return styleToolCard.Render(b.String())
}

// RenderPermissionRequest renders a permission request box with selectable options
func RenderPermissionRequest(toolName string, permission string, patterns []string, selectedOption int) string {
	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(colorWarning)
	b.WriteString(headerStyle.Render("╭─ Permission Request ─────────────────────────────────╮"))
	b.WriteString("\n")

	// Tool info section
	b.WriteString("│ ")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(colorSecondary).Render(toolName))
	b.WriteString(" wants to ")
	b.WriteString(lipgloss.NewStyle().Foreground(colorText).Render(permission))
	b.WriteString("\n")

	// Patterns
	if len(patterns) > 0 {
		b.WriteString("│\n")
		for _, p := range patterns {
			b.WriteString("│   ")
			b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render("→ "))
			// Highlight the pattern
			patternStyle := lipgloss.NewStyle().Foreground(colorText).Bold(true)
			if len(p) > 50 {
				p = p[:47] + "..."
			}
			b.WriteString(patternStyle.Render(p))
			b.WriteString("\n")
		}
	}

	b.WriteString("│\n")
	b.WriteString(headerStyle.Render("├─ Choose an option ────────────────────────────────────┤"))
	b.WriteString("\n")

	// Options - like Claude Code style
	options := []struct {
		key   string
		label string
		desc  string
	}{
		{"Y", "Allow once", "Allow this single request"},
		{"N", "Deny", "Reject this request"},
		{"A", "Always allow", "Allow all future requests for this tool"},
		{"D", "Deny all", "Block all requests for this tool in this session"},
	}

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#374151")).
		Foreground(colorText).
		Bold(true).
		Padding(0, 1)

	normalStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true)

	for i, opt := range options {
		b.WriteString("│  ")
		keyBadge := lipgloss.NewStyle().
			Background(colorPrimary).
			Foreground(lipgloss.Color("#000")).
			Bold(true).
			Padding(0, 1).
			Render(opt.key)

		if i == selectedOption {
			b.WriteString(keyBadge)
			b.WriteString(" ")
			b.WriteString(selectedStyle.Render(opt.label))
			b.WriteString(" ")
			b.WriteString(descStyle.Render(opt.desc))
		} else {
			b.WriteString(keyBadge)
			b.WriteString(" ")
			b.WriteString(normalStyle.Render(opt.label))
			b.WriteString(" ")
			b.WriteString(descStyle.Render(opt.desc))
		}
		b.WriteString("\n")
	}

	b.WriteString(headerStyle.Render("╰───────────────────────────────────────────────────────╯"))
	b.WriteString("\n")

	// Keyboard hint
	hintStyle := lipgloss.NewStyle().Foreground(colorMuted).Italic(true)
	b.WriteString(hintStyle.Render("  Press "))
	b.WriteString(keyStyle.Render("Y"))
	b.WriteString(hintStyle.Render("/"))
	b.WriteString(keyStyle.Render("N"))
	b.WriteString(hintStyle.Render("/"))
	b.WriteString(keyStyle.Render("A"))
	b.WriteString(hintStyle.Render("/"))
	b.WriteString(keyStyle.Render("D"))
	b.WriteString(hintStyle.Render(" or use "))
	b.WriteString(keyStyle.Render("↑↓"))
	b.WriteString(hintStyle.Render(" + "))
	b.WriteString(keyStyle.Render("Enter"))

	return b.String()
}

// RenderHelp renders the help screen
func RenderHelp() string {
	var b strings.Builder

	b.WriteString(styleTitle.Render("Available Commands"))
	b.WriteString("\n\n")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/help", "Show this help message"},
		{"/new", "Start a new session"},
		{"/clear", "Clear the screen"},
		{"/history", "Show message history"},
		{"/checkpoints", "List all checkpoints for current session"},
		{"/rewind <id>", "Rewind conversation to a checkpoint"},
		{"/rewind <id> --code", "Rewind code changes only"},
		{"/rewind <id> --all", "Rewind both code and conversation"},
		{"/exit, /quit", "Exit the CLI"},
	}

	for _, c := range commands {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			styleCommand.Render(fmt.Sprintf("%-12s", c.cmd)),
			styleSubtitle.Render(c.desc)))
	}

	b.WriteString("\n")
	b.WriteString(styleSubtitle.Render("Shortcuts:"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", styleCommand.Render("Enter      "), styleSubtitle.Render("Send message")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", styleCommand.Render("Shift+Enter"), styleSubtitle.Render("New line")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", styleCommand.Render("↑/↓        "), styleSubtitle.Render("Navigate history")))
	b.WriteString(fmt.Sprintf("  %s  %s\n", styleCommand.Render("Ctrl+C     "), styleSubtitle.Render("Exit")))

	return b.String()
}

// Helper functions

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		path = "~" + path[len(home):]
	}

	// Shorten if too long
	if len(path) > 40 {
		parts := strings.Split(path, string(filepath.Separator))
		if len(parts) > 3 {
			path = filepath.Join(parts[0], "...", parts[len(parts)-2], parts[len(parts)-1])
		}
	}

	return path
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

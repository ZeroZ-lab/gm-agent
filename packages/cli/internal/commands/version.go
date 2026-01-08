package commands

import (
	"fmt"
	"os"
	"runtime"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

// Version can be overridden at build time with -ldflags "-X ...".
var Version = "0.1.0"

// NewVersionCmd creates the version command.
func NewVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			title := lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FF6B35")).
				Render("gm-agent CLI")

			ver := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7C3AED")).
				Render(fmt.Sprintf("v%s", Version))

			info := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6B7280")).
				Render(fmt.Sprintf("(%s/%s)", runtime.GOOS, runtime.GOARCH))

			fmt.Fprintf(os.Stdout, "%s %s %s\n", title, ver, info)
		},
	}
}

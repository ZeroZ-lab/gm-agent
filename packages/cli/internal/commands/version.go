package commands

import (
	"fmt"
	"os"

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
			_, _ = fmt.Fprintln(os.Stdout, Version)
		},
	}
}

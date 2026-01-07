package main

import (
	"fmt"
	"os"

	"github.com/gm-agent-org/gm-agent/packages/cli/internal/commands"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

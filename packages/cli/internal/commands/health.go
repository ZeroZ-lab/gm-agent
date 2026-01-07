package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/gm-agent-org/gm-agent/packages/cli/internal/client"
	"github.com/spf13/cobra"
)

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// NewHealthCmd creates the health check command.
func NewHealthCmd(cfg *Config) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check agent health",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := client.New(cfg.Server, cfg.APIKey, cfg.Timeout)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			status, body, err := cli.Get(ctx, "/health")
			if err != nil {
				return err
			}
			if status != 200 {
				return fmt.Errorf("server returned %d: %s", status, string(body))
			}

			var resp healthResponse
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("parse response: %w", err)
			}

			_, _ = fmt.Fprintf(os.Stdout, "status=%s version=%s\n", resp.Status, resp.Version)
			return nil
		},
	}
}

package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
	}

	cmd.AddCommand(newLoginCmd())
	return cmd
}

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login with API Key",
		RunE: func(cmd *cobra.Command, args []string) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter API Key: ")
			apiKey, err := reader.ReadString('\n')
			if err != nil {
				return err
			}
			apiKey = strings.TrimSpace(apiKey)

			if apiKey == "" {
				// Allow empty key for dev mode / no-auth scenarios
				fmt.Println("Warning: Using empty API key. Ensure server has auth disabled.")
			}

			viper.Set("api-key", apiKey)

			// Force write to ~/.gm/config.yaml
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("fail to get home dir: %w", err)
			}
			configDir := filepath.Join(home, ".gm")
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("fail to create config dir: %w", err)
			}

			configFile := filepath.Join(configDir, "config.yaml")
			if err := viper.WriteConfigAs(configFile); err != nil {
				return fmt.Errorf("failed to save config to %s: %w", configFile, err)
			}

			fmt.Println("Successfully logged in!")
			return nil
		},
	}
}

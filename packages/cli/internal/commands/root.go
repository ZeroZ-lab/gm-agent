package commands

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultServer = "http://localhost:8080"
)

// Config holds CLI runtime configuration.
type Config struct {
	Server  string `mapstructure:"server"`
	APIKey  string `mapstructure:"api-key"`
	Timeout time.Duration
}

// NewRootCmd builds the root command with shared flags.
func NewRootCmd() *cobra.Command {
	cobra.OnInitialize(initConfig)

	cfg := &Config{
		Server:  defaultServer,
		Timeout: 10 * time.Second,
	}

	cmd := &cobra.Command{
		Use:           "gmcli",
		Short:         "CLI client for gm-agent",
		Long:          "CLI client for gm-agent HTTP API.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Reload config into struct
			return viper.Unmarshal(cfg)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return replLoop(cfg)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringP("server", "s", defaultServer, "Agent server base URL")
	cmd.PersistentFlags().String("api-key", "", "API key for server authentication")
	cmd.PersistentFlags().Duration("timeout", 10*time.Second, "HTTP request timeout")

	viper.BindPFlag("server", cmd.PersistentFlags().Lookup("server"))
	viper.BindPFlag("api-key", cmd.PersistentFlags().Lookup("api-key"))
	viper.BindPFlag("timeout", cmd.PersistentFlags().Lookup("timeout"))

	cmd.AddCommand(NewHealthCmd(cfg))
	cmd.AddCommand(NewVersionCmd())
	cmd.AddCommand(NewAuthCmd())
	cmd.AddCommand(NewRunCmd())

	return cmd
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Search config in ~/.gm
	viper.AddConfigPath(filepath.Join(home, ".gm"))
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")

	viper.SetEnvPrefix("GM")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	viper.ReadInConfig()
}

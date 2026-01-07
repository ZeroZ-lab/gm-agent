package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/gm-agent-org/gm-agent/packages/cli/internal/client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [prompt]",
		Short: "Run a one-off agent command",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			prompt := strings.Join(args, " ")
			return executeOneShot(prompt)
		},
	}
}

func executeOneShot(prompt string) error {
	apiKey := viper.GetString("api-key")
	// Allow empty API key if auth is disabled on server

	baseUrl := viper.GetString("server")
	c, err := client.New(baseUrl, apiKey, viper.GetDuration("timeout"))
	if err != nil {
		return err
	}

	ctx := context.Background()
	// Create Session
	sessionID, err := c.CreateSession(ctx, prompt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	// Stream Events
	eventCh, err := c.StreamEvents(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("stream events: %w", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for evt := range eventCh {
			switch evt.Type {
			case "assistant_message":
				var data struct {
					Content string `json:"content"`
				}
				if err := json.Unmarshal(evt.Data, &data); err == nil {
					fmt.Print(data.Content) // Print directly to stdout
				}
			case "error":
				fmt.Printf("\nError: %s\n", string(evt.Data))
			case "session_ended":
				return
			}
		}
	}()

	wg.Wait()
	fmt.Println() // Newline at end
	return nil
}

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gm-agent-org/gm-agent/pkg/agent/tools" // Moved to avoid "duplicate" interpretation, though it was not duplicated.
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/openai"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
	"github.com/gm-agent-org/gm-agent/pkg/types"
)

func main() {
	// 1. Setup Logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// 2. Setup Context with Cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// CLI Command Dispatch
	if len(os.Args) > 1 && os.Args[1] == "clean" {
		workingDir, _ := os.Getwd()
		dataDir := filepath.Join(workingDir, ".runtime")
		logger.Info("Cleaning runtime data...", "path", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			logger.Error("failed to clean data", "error", err)
			os.Exit(1)
		}
		logger.Info("Cleanup complete")
		return
	}

	// 3. Initialize Modules
	workingDir, _ := os.Getwd()
	dataDir := filepath.Join(workingDir, ".runtime")
	fsStore := store.NewFSStore(dataDir)

	if err := fsStore.Open(ctx); err != nil {
		logger.Error("failed to open store", "error", err)
		os.Exit(1)
	}
	defer fsStore.Close()

	// Setup LLM Provider
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		logger.Warn("OPENAI_API_KEY not set, functionality will be limited")
	}
	llmProvider := openai.New(openai.Config{APIKey: apiKey})
	llmGateway := llm.NewGateway(llmProvider)

	// Setup Tool System
	toolRegistry := tool.NewRegistry()
	toolPolicy := tool.NewPolicy()

	// Register Built-in Tools
	if err := toolRegistry.Register(tools.ReadFileTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.RunShellTool); err != nil {
		panic(err)
	}

	toolExecutor := tool.NewExecutor(toolRegistry, toolPolicy)

	// Register Handlers
	toolExecutor.RegisterHandler("read_file", tools.HandleReadFile)
	toolExecutor.RegisterHandler("run_shell", tools.HandleRunShell)

	// 4. Initialize Runtime
	config := runtime.DefaultConfig
	rt := runtime.New(config, fsStore, llmGateway, toolExecutor, logger)

	// 5. Run
	logger.Info("gm-agent starting...")

	// CLI Argument Handling (Simple One-Shot Goal)
	if len(os.Args) > 1 {
		goal := os.Args[1]
		logger.Info("received goal from cli", "goal", goal)

		event := &types.UserMessageEvent{
			BaseEvent: types.NewBaseEvent("user_request", "user", "cli"),
			Content:   goal,
			Priority:  10,
		}
		if err := rt.Ingest(ctx, event); err != nil {
			logger.Error("failed to ingest goal", "error", err)
			os.Exit(1)
		}
	}

	if err := rt.Run(ctx); err != nil {
		logger.Error("runtime exited with error", "error", err)
		os.Exit(1)
	}
	logger.Info("gm-agent finished successfully")
}

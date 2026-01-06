package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gm-agent-org/gm-agent/pkg/agent/tools" // Moved to avoid "duplicate" interpretation, though it was not duplicated.
	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/factory"
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

	// CLI Flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// CLI Command Dispatch
	// Handle "clean" command
	if flag.Arg(0) == "clean" {
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

	// 4. Initialize Config
	// Use flag if provided, otherwise empty string triggers default search
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("failed to load config", "error", err)
	}

	// Env Vars are now automatically merged by config.Load() using "GM_" prefix.
	// e.g. GM_OPENAI_API_KEY, GM_ACTIVE_PROVIDER

	// Setup LLM Provider
	llmProvider, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		logger.Error("failed to create llm provider", "error", err)
		os.Exit(1)
	}
	llmGateway := llm.NewGateway(llmProvider)

	// Setup Tool System
	toolRegistry := tool.NewRegistry()
	toolPolicy := tool.NewPolicy(cfg.Security)

	// Register Built-in Tools
	if err := toolRegistry.Register(tools.ReadFileTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.RunShellTool); err != nil {
		panic(err)
	}
	// Interactive Tools
	if err := toolRegistry.Register(tools.TalkTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.TaskCompleteTool); err != nil {
		panic(err)
	}

	toolExecutor := tool.NewExecutor(toolRegistry, toolPolicy)

	// Register Handlers
	toolExecutor.RegisterHandler("read_file", tools.HandleReadFile)
	toolExecutor.RegisterHandler("run_shell", tools.HandleRunShell)
	toolExecutor.RegisterHandler("talk", tools.HandleTalk)
	toolExecutor.RegisterHandler("task_complete", tools.HandleTaskComplete)

	// 4. Initialize Runtime
	config := runtime.DefaultConfig

	// Determine Model
	switch cfg.ActiveProvider {
	case "gemini":
		config.Model = cfg.Gemini.Model
	case "openai":
		config.Model = cfg.OpenAI.Model
	}
	// Fallback if empty in config (though we have defaults/yaml)
	if config.Model == "" {
		if cfg.ActiveProvider == "gemini" {
			config.Model = "gemini-2.0-flash"
		} else {
			config.Model = "deepseek-chat"
		}
	}

	logger.Info("Initializing Runtime", "provider", cfg.ActiveProvider, "model", config.Model)
	rt := runtime.New(config, fsStore, llmGateway, toolExecutor, logger)

	// 5. Run
	logger.Info("gm-agent starting...")

	// CLI Argument Handling (Simple One-Shot Goal)
	args := flag.Args()
	if len(args) > 0 && args[0] != "clean" { // "clean" handled above, but just in case
		goal := args[0]
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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gm-agent-org/gm-agent/pkg/agent/tools" // Moved to avoid "duplicate" interpretation, though it was not duplicated.
	"github.com/gm-agent-org/gm-agent/pkg/api"
	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/factory"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
)

func main() {
	// 1. Setup Logger
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// 2. Setup Context with Cancellation
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx, os.Args[1:]); err != nil {
		logger.Error("gm-agent exited with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	// CLI Flags
	flagSet := flag.NewFlagSet("gm", flag.ContinueOnError)
	configPath := flagSet.String("config", "", "Path to configuration file")
	if err := flagSet.Parse(args); err != nil {
		return err
	}

	remaining := flagSet.Args()
	mode := ""
	if len(remaining) > 0 {
		mode = remaining[0]
	}

	// CLI Command Dispatch
	// Handle "clean" command
	if mode == "clean" {
		workingDir, _ := os.Getwd()
		dataDir := filepath.Join(workingDir, ".runtime")
		logger.Info("Cleaning runtime data...", "path", dataDir)
		if err := os.RemoveAll(dataDir); err != nil {
			logger.Error("failed to clean data", "error", err)
			return fmt.Errorf("clean runtime data: %w", err)
		}
		logger.Info("Cleanup complete")
		return nil
	}

	// 3. Initialize Modules
	workingDir, _ := os.Getwd()
	dataDir := filepath.Join(workingDir, ".runtime")
	fsStore := store.NewFSStore(dataDir)

	if err := fsStore.Open(ctx); err != nil {
		logger.Error("failed to open store", "error", err)
		return fmt.Errorf("open store: %w", err)
	}
	defer fsStore.Close()

	// 4. Initialize Config
	// Use flag if provided, otherwise empty string triggers default search
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("failed to load config", "error", err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Env Vars are now automatically merged by config.Load() using "GM_" prefix.
	// e.g. GM_OPENAI_API_KEY, GM_ACTIVE_PROVIDER

	if cfg.HTTP.Addr == "" {
		cfg.HTTP.Addr = ":8080"
	}

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	// Setup LLM Provider
	llmProvider, providerID, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		logger.Error("failed to create llm provider", "error", err)
		return fmt.Errorf("create llm provider: %w", err)
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
	rtConfig := runtime.DefaultConfig

	// Get model from active provider options
	_, opts, err := cfg.GetActiveProvider()
	if err == nil && opts.Model != "" {
		rtConfig.Model = opts.Model
	} else {
		// Fallback default
		rtConfig.Model = "gemini-2.0-flash"
	}

	// 5. Run
	logger.Info("gm-agent starting...")

	sessionFactory := func(sessionID string) (*api.SessionResources, error) {
		sessionCtx, cancel := context.WithCancel(ctx)
		sessionDir := filepath.Join(dataDir, "sessions", sessionID)
		sessionStore := store.NewFSStore(sessionDir)
		if err := sessionStore.Open(sessionCtx); err != nil {
			cancel()
			return nil, err
		}

		rt := runtime.New(rtConfig, sessionStore, llmGateway, toolExecutor, logger)
		return &api.SessionResources{Runtime: rt, Store: sessionStore, Ctx: sessionCtx, Cancel: cancel}, nil
	}

	apiCfg := api.HTTPConfig{Enable: cfg.HTTP.Enable, Addr: cfg.HTTP.Addr, APIKey: cfg.HTTP.APIKey}
	server := api.NewServer(ctx, apiCfg, sessionFactory, logger)
	httpSrv := &http.Server{Addr: cfg.HTTP.Addr, Handler: server.Engine}

	go func() {
		<-ctx.Done()
		_ = httpSrv.Shutdown(context.Background())
	}()

	logger.Info("http api listening", "addr", cfg.HTTP.Addr, "provider", providerID, "model", rtConfig.Model)
	if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("http server error", "error", err)
		return fmt.Errorf("http server: %w", err)
	}
	logger.Info("http api stopped")
	return nil
}

func parseLogLevel(level string) slog.Level {
	normalized := strings.ToUpper(strings.TrimSpace(level))
	switch normalized {
	case "DEBUG":
		return slog.LevelDebug
	case "VERBOSE":
		return slog.LevelDebug
	case "WARN", "WARNING":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	case "INFO", "":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}

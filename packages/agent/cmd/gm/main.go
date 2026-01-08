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
	"time"

	"github.com/gm-agent-org/gm-agent/pkg/agent/tools"
	"github.com/gm-agent-org/gm-agent/pkg/api"
	"github.com/gm-agent-org/gm-agent/pkg/api/service"
	"github.com/gm-agent-org/gm-agent/pkg/config"
	"github.com/gm-agent-org/gm-agent/pkg/llm"
	"github.com/gm-agent-org/gm-agent/pkg/llm/factory"
	"github.com/gm-agent-org/gm-agent/pkg/patch"
	"github.com/gm-agent-org/gm-agent/pkg/runtime"
	"github.com/gm-agent-org/gm-agent/pkg/runtime/permission"
	"github.com/gm-agent-org/gm-agent/pkg/store"
	"github.com/gm-agent-org/gm-agent/pkg/tool"
	"github.com/gm-agent-org/gm-agent/pkg/types"

	_ "github.com/gm-agent-org/gm-agent/docs" // Swagger docs
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
		return cmdClean(logger)
	}

	// Default: Run Server
	return cmdServe(ctx, logger, *configPath)
}

func cmdClean(logger *slog.Logger) error {
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

func cmdServe(ctx context.Context, logger *slog.Logger, configPath string) error {
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
	cfg, err := config.Load(configPath)
	if err != nil {
		logger.Warn("failed to load config", "error", err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}

	// Env Vars are now automatically merged by config.Load() using "GM_" prefix.
	// e.g. GM_OPENAI_API_KEY, GM_ACTIVE_PROVIDER

	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}))
	slog.SetDefault(logger)

	// Setup LLM Provider
	llmProvider, providerID, err := factory.NewProvider(ctx, cfg)
	if err != nil {
		logger.Error("failed to create llm provider", "error", err)
		return fmt.Errorf("create llm provider: %w", err)
	}

	// Get active provider options for Gateway configuration
	_, opts, err := cfg.GetActiveProvider()
	if err != nil {
		// This should theoretically not fail if factory.NewProvider succeeded, but handle safely
		logger.Warn("could not retrieve active provider options for gateway config", "error", err)
	}
	llmGateway := llm.NewGateway(llmProvider, opts)

	// Setup Tool System
	toolRegistry := tool.NewRegistry()
	// Pass fsStore to Policy for persistent rules
	toolPolicy := tool.NewPolicy(cfg.Security, toolRegistry, fsStore)

	// Initialize Patch Engine
	workingDir, _ = os.Getwd()
	patchEngine, err := patch.NewEngine(patch.Config{
		WorkDir:         workingDir,
		BackupDir:       ".gm-backups",
		MaxContextLines: 3,
	})
	if err != nil {
		logger.Error("failed to create patch engine", "error", err)
		return fmt.Errorf("create patch engine: %w", err)
	}

	// Register Built-in Tools
	if err := toolRegistry.Register(tools.ReadFileTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.CreateFileTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.RunShellTool); err != nil {
		panic(err)
	}

	// New Advanced File Tools (2026-01-08)
	if err := toolRegistry.Register(tools.WriteFileTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.EditFileTool); err != nil {
		panic(err)
	}

	// Search Tools (2026-01-08)
	if err := toolRegistry.Register(tools.GlobTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.GrepTool); err != nil {
		panic(err)
	}

	// Interactive Tools
	if err := toolRegistry.Register(tools.TalkTool); err != nil {
		panic(err)
	}
	if err := toolRegistry.Register(tools.TaskCompleteTool); err != nil {
		panic(err)
	}

	// Sub-function to register handlers (avoids duplication)
	registerHandlers := func(executor *tool.Executor, patchEng patch.Engine) {
		executor.RegisterHandler("read_file", tools.HandleReadFile)
		executor.RegisterHandler("create_file", tools.HandleCreateFile)
		executor.RegisterHandler("run_shell", tools.HandleRunShell)
		executor.RegisterHandler("talk", tools.HandleTalk)
		executor.RegisterHandler("task_complete", tools.HandleTaskComplete)

		// New handlers with patch engine (2026-01-08)
		executor.RegisterHandler("write_file", func(ctx context.Context, args string) (string, error) {
			return tools.HandleWriteFile(ctx, args, patchEng)
		})
		executor.RegisterHandler("edit_file", func(ctx context.Context, args string) (string, error) {
			return tools.HandleEditFile(ctx, args, patchEng)
		})
		executor.RegisterHandler("glob", tools.HandleGlob)
		executor.RegisterHandler("grep", tools.HandleGrep)
	}

	// 4. Initialize Runtime
	rtConfig := runtime.DefaultConfig

	// Apply Model from options
	if opts.Model != "" {
		rtConfig.Model = opts.Model
	} else {
		// Fallback default
		rtConfig.Model = "gemini-2.0-flash"
	}

	// 5. Run
	logger.Info("gm-agent starting...")

	sessionFactory := func(sessionID string) (*service.SessionResources, error) {
		sessionCtx, cancel := context.WithCancel(ctx)
		sessionDir := filepath.Join(dataDir, "sessions", sessionID)
		sessionStore := store.NewFSStore(sessionDir)
		if err := sessionStore.Open(sessionCtx); err != nil {
			cancel()
			return nil, err
		}

		// Create Permission Manager for this session
		permManager := permission.NewManager(logger)

		// Create per-session Executor
		// We reuse the registry and policy as they are thread-safe and stateless/config-based
		sessionExecutor := tool.NewExecutor(toolRegistry, toolPolicy)
		registerHandlers(sessionExecutor, patchEngine)

		// Wire Permission Callback
		sessionExecutor.SetPermissionCallback(func(ctx context.Context, req tool.PermissionRequest) (bool, error) {
			logger.Info("requesting permission", "tool", req.ToolName, "id", req.RequestID)

			// Emit PermissionRequestEvent so Client can see it via SSE
			reqEvent := &types.PermissionRequestEvent{
				BaseEvent:  types.NewBaseEvent("permission_request", "system", sessionID),
				RequestID:  req.RequestID,
				ToolName:   req.ToolName,
				Permission: req.Permission,
				Patterns:   req.Patterns,
				Metadata:   req.Metadata,
			}
			if err := sessionStore.AppendEvent(ctx, reqEvent); err != nil {
				logger.Error("failed to emit permission request event", "error", err)
			}

			// Register request to manager so we can wait on it
			permManager.Request(req.RequestID)

			// Wait for user approval (timeout 5m)
			// TODO: Make timeout configurable
			resp, err := permManager.WaitForResponse(ctx, req.RequestID, 300*time.Second)
			if err != nil {
				return false, err
			}

			// If Always Allow selected, persist the rule
			if resp.Always && resp.Approved {
				if len(req.Patterns) > 0 {
					rule := types.PermissionRule{
						ID:        types.GenerateID("rule"),
						ToolName:  req.ToolName,
						Action:    "allow",
						Pattern:   tool.NormalizeArguments(req.Patterns[0]), // MVP: Use exact match of first pattern (args JSON)
						CreatedAt: time.Now(),
					}
					// Only save if action is allowed
					if err := fsStore.AddPermissionRule(ctx, rule); err != nil {
						logger.Error("failed to save permission rule", "error", err)
						// Proceed anyway since current request is approved
					} else {
						logger.Info("saved persistent permission rule", "tool", req.ToolName)
					}
				}
			}

			return resp.Approved, nil
		})

		rt := runtime.New(rtConfig, sessionStore, llmGateway, sessionExecutor, logger)
		// Set file change tracker for Code Rewind support
		rt.SetFileChangeTracker(patchEngine.GetTracker())
		return &service.SessionResources{
			Runtime:     rt,
			Permissions: permManager,
			Store:       sessionStore,
			PatchEngine: patchEngine,
			Ctx:         sessionCtx,
			Cancel:      cancel,
		}, nil
	}

	sessionSvc := service.NewSessionService(sessionFactory, logger)
	apiCfg := api.Config{Enable: cfg.HTTP.Enable, Addr: cfg.HTTP.Addr, APIKey: cfg.HTTP.APIKey, DevMode: cfg.DevMode}
	server := api.NewServer(apiCfg, sessionSvc, logger)
	httpSrv := &http.Server{Addr: cfg.HTTP.Addr, Handler: server.Engine()}

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

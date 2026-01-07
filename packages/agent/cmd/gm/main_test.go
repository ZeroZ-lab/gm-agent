package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMainRunsWithoutArgs(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer os.Chdir(cwd)

	t.Setenv("GEMINI_API_KEY", "test-token")
	t.Setenv("GM_HTTP_ADDR", ":0")

	done := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		done <- run(ctx, []string{})
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for run to exit")
	}

	if _, err := os.Stat(filepath.Join(dir, ".runtime")); err != nil {
		t.Fatalf("expected runtime data directory: %v", err)
	}
}

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  slog.Level
	}{
		{name: "debug", input: "DEBUG", want: slog.LevelDebug},
		{name: "verbose", input: "VERBOSE", want: slog.LevelDebug},
		{name: "warning", input: "WARNING", want: slog.LevelWarn},
		{name: "warn", input: "WARN", want: slog.LevelWarn},
		{name: "info default", input: "INFO", want: slog.LevelInfo},
		{name: "empty", input: "", want: slog.LevelInfo},
		{name: "unknown", input: "nope", want: slog.LevelInfo},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLogLevel(tc.input)
			if got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleReadFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(file, []byte("hello"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	t.Run("success", func(t *testing.T) {
		content, err := HandleReadFile(context.Background(), `{"path":"`+file+`"}`)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if content != "hello" {
			t.Fatalf("unexpected content %q", content)
		}
	})

	t.Run("invalid input", func(t *testing.T) {
		if _, err := HandleReadFile(context.Background(), "{}"); err == nil {
			t.Fatalf("expected error for missing path")
		}
	})
}

func TestHandleRunShell(t *testing.T) {
	t.Run("empty command", func(t *testing.T) {
		if _, err := HandleRunShell(context.Background(), "{}"); err == nil {
			t.Fatalf("expected error for empty command")
		}
	})

	t.Run("executes command", func(t *testing.T) {
		output, err := HandleRunShell(context.Background(), `{"command":"echo hi"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.TrimSpace(output) != "hi" {
			t.Fatalf("unexpected output %q", output)
		}
	})
}

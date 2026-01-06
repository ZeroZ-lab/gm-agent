package main

import (
	"os"
	"path/filepath"
	"testing"
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

	os.Args = []string{"gm"}
	main()

	if _, err := os.Stat(filepath.Join(dir, ".runtime")); err != nil {
		t.Fatalf("expected runtime data directory: %v", err)
	}
}

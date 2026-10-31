package main

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// TestRunFailures checks missing fixtures and blocked logger setup without exiting the process.
func TestRunFailures(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := run(); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("missing fixture error: %v", err)
	}
	data, err := os.ReadFile("logs/go-practices.log")
	if err != nil || !strings.Contains(string(data), `"msg":"open fixture"`) {
		t.Fatalf("failure not logged: %s, %v", data, err)
	}
	if err := os.RemoveAll("logs"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("logs", []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err == nil {
		t.Fatal("expected logger initialization error")
	}
}

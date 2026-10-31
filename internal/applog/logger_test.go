package applog

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestLoggerSetup checks log file creation, JSON output, and setup failures.
func TestLoggerSetup(t *testing.T) {
	var (
		data              []byte
		path              = filepath.Join(t.TempDir(), "logs", "go-practices.log")
		logger, file, err = NewLogger(path)
	)

	if err != nil {
		t.Fatal(err)
	}

	logger.Info("test")
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	data, err = os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(`"msg":"test"`)) {
		t.Fatalf("log not written: %s, %v", data, err)
	}

	if _, _, err := NewLogger(filepath.Join(path, "blocked.log")); err == nil {
		t.Fatal("expected directory creation failure")
	}

	if _, _, err := NewLogger(filepath.Dir(path)); err == nil {
		t.Fatal("expected file open failure")
	}
}

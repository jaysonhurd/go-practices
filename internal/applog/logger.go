// Package applog creates append-only structured application logs.
package applog

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
)

var (
	// ErrCreateLogDirectory indicates that the log directory could not be created.
	ErrCreateLogDirectory = errors.New("create log directory")
	// ErrOpenLogFile indicates that the application log could not be opened.
	ErrOpenLogFile = errors.New("open log file")
)

// NewLogger creates the parent directory and opens an append-only JSON log.
// The caller owns the file and must close it after the server shuts down.
func NewLogger(path string) (*slog.Logger, *os.File, error) {
	var (
		file *os.File
		err  error
	)

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, nil, errors.Join(ErrCreateLogDirectory, err)
	}

	file, err = os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, nil, errors.Join(ErrOpenLogFile, err)
	}

	return slog.New(slog.NewJSONHandler(file, nil)), file, nil
}

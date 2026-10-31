package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	apphttp "github.com/jaysonhurd/go-practices/httpserver/internal/http"
	"github.com/jaysonhurd/go-practices/httpserver/internal/middlewares/logging"
	"github.com/jaysonhurd/go-practices/internal/telemetry"
)

const (
	logTelemetryShutdownFailed = "shut down telemetry"
	// jwtSecretEnv names the environment variable containing the JWT signing secret.
	jwtSecretEnv = "JWT_SECRET"
	// logPath is the repository-relative destination for structured server logs.
	logPath = "logs/go-practices.log"
	// logServerFailed identifies a fatal server startup or runtime failure.
	logServerFailed = "HTTP server failed"
	// logCloseFailed identifies a failure to close the server log file.
	logCloseFailed = "close HTTP log"
)

var (
	// ErrInitializeLogger indicates that the HTTP logger could not be initialized.
	ErrInitializeLogger = errors.New("initialize HTTP logger")
)

// main starts the server and reports startup or runtime errors.
func main() {
	// Keep setup failures in JSON even when the log file cannot be opened.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	if err := run(); err != nil {
		slog.Error(logServerFailed, slog.Any("error", err))
		os.Exit(1)
	}
}

// run owns application configuration, logs, telemetry, and signal handling.
func run() error {
	var (
		ctx, stop         = signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		logger, file, err = logging.NewLogger(logPath)
		server            apphttp.Runner
	)

	defer stop()

	if err != nil {
		return errors.Join(ErrInitializeLogger, err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			slog.Error(logCloseFailed, slog.Any("error", err))
		}
	}()

	telemetryRuntime, err := telemetry.New(os.Getenv("OTEL_STDOUT_ENABLED") == "true", os.Stdout)
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryRuntime.Shutdown(ctx); err != nil {
			slog.Error(logTelemetryShutdownFailed, slog.Any("error", err))
		}
	}()

	server, err = apphttp.NewWithMiddleware(logger, []byte(os.Getenv(jwtSecretEnv)), telemetryRuntime.Wrap)
	if err != nil {
		return err
	}

	return server.Run(ctx)
}

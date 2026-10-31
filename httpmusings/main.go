// Package main demonstrates an HTTP client behind a standard-library handler.
package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jaysonhurd/go-practices/internal/applog"
	"github.com/jaysonhurd/go-practices/internal/telemetry"
)

const (
	logTelemetryShutdownFailed = "shut down telemetry"
	// PREVCLOSE is the aggregate endpoint prefix retained for compatibility.
	PREVCLOSE = "/v2/aggs/ticker/"
	// APIKEY is the upstream API key query parameter.
	APIKEY           = "apiKey"
	logServerFailed  = "HTTP example failed"
	logRequestFailed = "previous close request failed"
)

var (
	// ErrAPIKeyRequired indicates that POLYGON_API_KEY was not configured.
	ErrAPIKeyRequired = errors.New("POLYGON_API_KEY must be set")
	apiurl            = "https://api.polygon.io"
)

// ClientDoer is the HTTP operation needed by the market-data client.
type ClientDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// main starts the HTTP example using an environment-provided API credential.
func main() {
	logger, file, err := applog.NewLogger("logs/go-practices.log")
	if err != nil {
		slog.Error(logServerFailed, slog.Any("error", err))
		os.Exit(1)
	}
	defer file.Close()
	slog.SetDefault(logger)

	telemetryRuntime, err := telemetry.New(os.Getenv("OTEL_STDOUT_ENABLED") == "true", os.Stdout)
	if err != nil {
		logger.Error(logServerFailed, slog.Any("error", err))
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := telemetryRuntime.Shutdown(ctx); err != nil {
			slog.Error(logTelemetryShutdownFailed, slog.Any("error", err))
		}
	}()

	server := &http.Server{
		Addr: ":8080", Handler: telemetryRuntime.Wrap(http.HandlerFunc(handler)),
		ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorLog: slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := serve(ctx, server); err != nil {
		logger.Error(logServerFailed, slog.Any("error", err))
	}
}

// handler proxies the previous close response for the example ticker.
func handler(w http.ResponseWriter, r *http.Request) {
	result, err := previousClose(r.Context(), &http.Client{Timeout: 10 * time.Second}, apiurl, os.Getenv("POLYGON_API_KEY"), "AAPL")
	if err != nil {
		// Upstream errors may include credential-bearing URLs; keep them out of logs.
		slog.ErrorContext(r.Context(), logRequestFailed)
	}
	_, _ = io.WriteString(w, result)
}

// PreviousClose fetches the upstream response body for ticker.
// Set POLYGON_API_KEY before calling. The caller receives transport and read errors.
func PreviousClose(ticker string) (string, error) {
	return previousClose(context.Background(), &http.Client{Timeout: 10 * time.Second}, apiurl, os.Getenv("POLYGON_API_KEY"), ticker)
}

// previousClose uses an injected client and propagates request cancellation.
func previousClose(ctx context.Context, client ClientDoer, baseURL, apiKey, ticker string) (string, error) {
	if apiKey == "" {
		return "", ErrAPIKeyRequired
	}
	requestURL := strings.TrimRight(baseURL, "/") + PREVCLOSE + url.PathEscape(ticker) + "/prev?" + url.Values{
		"adjusted": {"true"}, APIKEY: {apiKey},
	}.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// serve drains active requests on cancellation before telemetry is flushed.
func serve(ctx context.Context, server *http.Server) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	finished := make(chan error, 1)
	go func() { finished <- server.ListenAndServe() }()
	select {
	case err := <-finished:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return errors.Join(err, server.Close(), <-finished)
		}
		err := <-finished
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

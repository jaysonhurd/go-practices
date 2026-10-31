// Package logging sets up file-based structured HTTP request logging.
package logging

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jaysonhurd/go-practices/internal/applog"
)

const (
	// logHTTPRequest identifies a completed HTTP request in structured logs.
	logHTTPRequest = "HTTP request"
)

var (
	// _ verifies that responseWriter preserves the standard HTTP response contract.
	_ http.ResponseWriter = (*responseWriter)(nil)
	// _ verifies that responseWriter exposes its underlying writer to response controllers.
	_ responseUnwrapper = (*responseWriter)(nil)

	// ErrCreateLogDirectory indicates that the parent directory for the log file could not be created.
	ErrCreateLogDirectory = applog.ErrCreateLogDirectory
	// ErrOpenLogFile indicates that the log file could not be opened for appending.
	ErrOpenLogFile = applog.ErrOpenLogFile
)

// NewLogger creates an append-only JSON log; the caller must close the returned file.
func NewLogger(path string) (*slog.Logger, *os.File, error) {
	return applog.NewLogger(path)
}

// responseUnwrapper exposes the underlying writer to http.ResponseController.
type responseUnwrapper interface {
	// Unwrap returns the underlying HTTP response writer.
	Unwrap() http.ResponseWriter
}

// responseWriter records status while retaining access to the underlying writer.
type responseWriter struct {
	http.ResponseWriter `json:"-"`
	status              int `json:"-"`
}

// Unwrap returns the underlying response writer for use by HTTP response controllers.
func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// WriteHeader forwards the status code and records the first final response status.
// Later status writes are ignored after a final response has been recorded.
func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}

	w.ResponseWriter.WriteHeader(status)
	if status >= 200 {
		w.status = status
	}
}

// Write writes the response body, recording an implicit HTTP 200 status if needed.
func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

// New logs requests, including authentication failures, without logging tokens.
func New(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				start    = time.Now()
				recorder = &responseWriter{ResponseWriter: w}
			)

			defer func() {
				var (
					status = recorder.status
				)

				if status == 0 {
					status = http.StatusOK
				}

				logger.InfoContext(r.Context(), logHTTPRequest,
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.String("duration", time.Since(start).String()),
					slog.String("remote_addr", r.RemoteAddr),
				)
			}()

			next.ServeHTTP(recorder, r)
		})
	}
}

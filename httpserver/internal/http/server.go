// Package http configures the application HTTP server, routes, and middleware.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jaysonhurd/go-practices/httpserver/internal/middlewares/authentication"
	"github.com/jaysonhurd/go-practices/httpserver/internal/middlewares/authorization"
	"github.com/jaysonhurd/go-practices/httpserver/internal/middlewares/logging"
)

const (
	// serverAddress is the listening address for the HTTP server.
	serverAddress = ":8080"
	// headerContentType names the HTTP response media type header.
	headerContentType = "Content-Type"
	// contentTypeJSON is the media type used for JSON responses.
	contentTypeJSON = "application/json"
	// shutdownTimeout limits how long the server waits for active requests to finish.
	shutdownTimeout = 10 * time.Second
	// logHealthFailed identifies a failure to write the health response.
	logHealthFailed = "write health response"
	// logServerStarting identifies the start of the HTTP serving lifecycle.
	logServerStarting = "HTTP server starting"
	// logServerStopped identifies successful graceful shutdown.
	logServerStopped = "HTTP server stopped"
)

var (
	// _ verifies that Server implements the application lifecycle contract.
	_ Runner = (*Server)(nil)

	// ErrLoggerRequired indicates that server configuration omitted the required logger.
	ErrLoggerRequired = errors.New("HTTP logger is required")
	// ErrSigningSecretRequired indicates that server configuration omitted the JWT signing secret.
	ErrSigningSecretRequired = errors.New("JWT signing secret must be set")
	// ErrConfigureAuthentication indicates that authentication middleware setup failed.
	ErrConfigureAuthentication = errors.New("configure authentication")
	// ErrShutdownServer indicates that graceful HTTP server shutdown failed.
	ErrShutdownServer = errors.New("shut down HTTP server")
)

// middleware composes a handler with a single request concern.
type middleware func(http.Handler) http.Handler

// Runner defines the lifecycle contract for an application HTTP server.
type Runner interface {
	// Run serves requests until ctx is canceled, then shuts down gracefully.
	Run(ctx context.Context) error
}

// Server owns the configured HTTP server and its lifecycle logger.
type Server struct {
	server *http.Server `json:"-"`
	logger *slog.Logger `json:"-"`
}

// New sets up all middleware, routes, and HTTP server defaults.
func New(logger *slog.Logger, secret []byte) (*Server, error) {
	return NewWithMiddleware(logger, secret)
}

// NewWithMiddleware adds outer instrumentation without changing route behavior.
// Wrappers run in argument order, with the first wrapper outermost.
func NewWithMiddleware(logger *slog.Logger, secret []byte, wrappers ...func(http.Handler) http.Handler) (*Server, error) {
	var (
		authenticate func(http.Handler) http.Handler
		handler      http.Handler
		err          error
	)

	if logger == nil {
		return nil, ErrLoggerRequired
	}

	if len(secret) == 0 {
		return nil, ErrSigningSecretRequired
	}

	// Own the key so callers cannot change it after configuring the server.
	secret = append([]byte(nil), secret...)
	authenticate, err = authentication.New(func(_ *jwt.Token) (any, error) {
		return secret, nil
	}, []string{jwt.SigningMethodHS256.Alg()})

	if err != nil {
		return nil, errors.Join(ErrConfigureAuthentication, err)
	}
	handler = newHandler(logger, authenticate, authorization.New(), logging.New(logger))

	for i := len(wrappers) - 1; i >= 0; i-- {
		handler = wrappers[i](handler)
	}

	return &Server{server: newServer(logger, handler), logger: logger}, nil
}

// newHandler composes public and protected routes with request logging.
func newHandler(logger *slog.Logger, authenticate, authorize, logRequests middleware) http.Handler {
	var (
		mux = http.NewServeMux()
	)

	mux.HandleFunc("GET /health", healthHandler(logger))
	mux.Handle("GET /items", authenticate(authorize(http.HandlerFunc(itemsHandler))))

	return logRequests(mux)
}

// healthHandler writes the existing JSON health response.
func healthHandler(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(headerContentType, contentTypeJSON)

		if err := json.NewEncoder(w).Encode(map[string]string{"health": "true"}); err != nil {
			logger.ErrorContext(r.Context(), logHealthFailed, slog.Any("error", err))
		}
	}
}

// itemsHandler returns the placeholder items response.
func itemsHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// newServer configures the listener and request timeouts.
func newServer(logger *slog.Logger, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              serverAddress,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

// Run serves requests until ctx is canceled, then shuts down gracefully.
func (s *Server) Run(ctx context.Context) error {
	var (
		server = s.server
		logger = s.logger
		result = make(chan error, 1)
	)

	if err := ctx.Err(); err != nil {
		return err
	}

	go func() { result <- server.ListenAndServe() }()
	logger.Info(logServerStarting, slog.String("address", server.Addr))

	select {
	case err := <-result:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	case <-ctx.Done():
		if err := shutdownServer(ctx, server); err != nil {
			return err
		}
		if err := <-result; !errors.Is(err, http.ErrServerClosed) {
			return err
		}

		logger.Info(logServerStopped)

		return nil
	}
}

// shutdownServer drains requests using a fresh deadline after cancellation.
func shutdownServer(ctx context.Context, server *http.Server) error {
	// Shutdown must drain requests even though the serving context is canceled.
	var (
		shutdownCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return errors.Join(ErrShutdownServer, err)
	}

	return nil
}

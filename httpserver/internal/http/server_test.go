package http

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TestRoutes checks routing, authentication, and request logging together.
func TestRoutes(t *testing.T) {
	var (
		secret = []byte("test-secret")
		sign   = func(method jwt.SigningMethod, key []byte, expiration time.Time) string {
			t.Helper()
			var (
				token, err = jwt.NewWithClaims(method, jwt.MapClaims{"sub": "user", "exp": expiration.Unix()}).SignedString(key)
			)

			if err != nil {
				t.Fatal(err)
			}

			return token
		}
		valid = sign(jwt.SigningMethodHS256, secret, time.Now().Add(time.Hour))
	)

	for _, tc := range []struct {
		name, method, path, header string
		status                     int
		body                       string
	}{
		{"health", "GET", "/health", "", 200, "{\"health\":\"true\"}\n"},
		{"health ignores bad token", "GET", "/health", "Bearer invalid", 200, "{\"health\":\"true\"}\n"},
		{"valid", "GET", "/items", "Bearer " + valid, 200, ""},
		{"lowercase bearer", "GET", "/items", "bearer " + valid, 200, ""},
		{"missing", "GET", "/items", "", 401, "Unauthorized\n"},
		{"malformed", "GET", "/items", "Bearer invalid", 401, "Unauthorized\n"},
		{"wrong scheme", "GET", "/items", "Basic " + valid, 401, "Unauthorized\n"},
		{"expired", "GET", "/items", "Bearer " + sign(jwt.SigningMethodHS256, secret, time.Now().Add(-time.Hour)), 401, "Unauthorized\n"},
		{"bad signature", "GET", "/items", "Bearer " + sign(jwt.SigningMethodHS256, []byte("wrong"), time.Now().Add(time.Hour)), 401, "Unauthorized\n"},
		{"wrong algorithm", "GET", "/items", "Bearer " + sign(jwt.SigningMethodHS384, secret, time.Now().Add(time.Hour)), 401, "Unauthorized\n"},
		{"wrong method", "POST", "/items", "Bearer " + valid, 405, "Method Not Allowed\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				logs        bytes.Buffer
				entry       map[string]any
				w           = httptest.NewRecorder()
				req         = httptest.NewRequest(tc.method, tc.path, nil)
				logger      = slog.New(slog.NewJSONHandler(&logs, nil))
				server, err = New(logger, secret)
			)

			if err != nil {
				t.Fatal(err)
			}

			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}

			server.server.Handler.ServeHTTP(w, req)

			if w.Code != tc.status || w.Body.String() != tc.body {
				t.Fatalf("got %d %q", w.Code, w.Body.String())
			}

			if tc.status == 401 && w.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("missing bearer challenge")
			}

			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}

			if entry["status"] != float64(tc.status) || entry["path"] != tc.path {
				t.Fatalf("unexpected log: %v", entry)
			}

			if strings.Contains(logs.String(), valid) {
				t.Fatal("token leaked into log")
			}
		})
	}
}

// TestNewRejectsInvalidConfiguration checks that server setup rejects missing dependencies.
func TestNewRejectsInvalidConfiguration(t *testing.T) {
	var (
		logger = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	)

	for _, tc := range []struct {
		name   string
		logger *slog.Logger
		secret []byte
	}{
		{"missing logger", nil, []byte("secret")},
		{"missing secret", logger, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := New(tc.logger, tc.secret); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

// TestRunClosedServer checks that running a closed server returns without error.
func TestRunClosedServer(t *testing.T) {
	var (
		logger      = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
		server, err = New(logger, []byte("secret"))
	)

	if err != nil {
		t.Fatal(err)
	}

	if err := server.server.Close(); err != nil {
		t.Fatal(err)
	}

	if err := server.Run(context.Background()); err != nil {
		t.Fatalf("closed server returned an error: %v", err)
	}
}

// TestServerOwnsSigningKey checks that changing the caller’s signing key does not affect the server.
func TestServerOwnsSigningKey(t *testing.T) {
	var (
		secret     = []byte("test-secret")
		logger     = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
		server     *Server
		request    = httptest.NewRequest(http.MethodGet, "/items", nil)
		response   = httptest.NewRecorder()
		token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user"}).SignedString(secret)
	)

	if err != nil {
		t.Fatal(err)
	}

	server, err = New(logger, secret)
	if err != nil {
		t.Fatal(err)
	}

	secret[0] = 'X'

	request.Header.Set("Authorization", "Bearer "+token)
	server.server.Handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; want %d", response.Code, http.StatusOK)
	}
}

// TestRunCanceledContext checks that Run returns cancellation before starting a listener.
func TestRunCanceledContext(t *testing.T) {
	var (
		logger      = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
		ctx, cancel = context.WithCancel(context.Background())
		server, err = New(logger, []byte("test-secret"))
	)

	defer cancel()

	if err != nil {
		t.Fatal(err)
	}

	cancel()

	if err := server.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v; want context.Canceled", err)
	}
}

// TestRunShutsDownOnCancellation checks that cancellation shuts down a running server gracefully.
func TestRunShutsDownOnCancellation(t *testing.T) {
	var (
		logger      = slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
		ctx, cancel = context.WithCancel(context.Background())
		started     = make(chan struct{})
		result      = make(chan error, 1)
		server, err = New(logger, []byte("test-secret"))
	)

	defer cancel()

	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = server.server.Close() })
	server.server.Addr = "127.0.0.1:0"
	server.server.BaseContext = func(net.Listener) context.Context {
		close(started)
		return ctx
	}

	go func() { result <- server.Run(ctx) }()

	select {
	case <-started:
	case err := <-result:
		t.Fatalf("server stopped before starting: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start")
	}

	cancel()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("Run error = %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down")
	}
}

// TestOuterMiddleware verifies wrapper order and preservation of public health behavior.
func TestOuterMiddleware(t *testing.T) {
	var order []string
	wrapper := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	server, err := NewWithMiddleware(slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)), []byte("secret"), wrapper("first"), wrapper("second"))
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/health", nil))
	if strings.Join(order, ",") != "first,second" || response.Code != http.StatusOK {
		t.Fatalf("order=%v status=%d", order, response.Code)
	}
}

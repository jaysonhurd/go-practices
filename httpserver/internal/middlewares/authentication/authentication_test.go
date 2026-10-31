package authentication

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// TestClaimsContext checks that authenticated JWT claims reach the handler through its context.
func TestClaimsContext(t *testing.T) {
	var (
		secret = []byte("test-secret")
		claims = jwt.MapClaims{
			"sub":    "user",
			"roles":  []string{"reader"},
			"custom": map[string]any{"enabled": true},
		}
		called          bool
		token           string
		handler         http.Handler
		request         = httptest.NewRequest(http.MethodGet, "/", nil)
		middleware, err = New(func(_ *jwt.Token) (any, error) { return secret, nil }, []string{jwt.SigningMethodHS256.Alg()})
	)

	if err != nil {
		t.Fatal(err)
	}

	token, err = jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
	if err != nil {
		t.Fatal(err)
	}

	handler = middleware(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		var (
			claims, ok = ClaimsFromContext(r.Context())
			custom     map[string]any
		)

		called = true
		if !ok || claims["sub"] != "user" || len(claims) != 3 {
			t.Fatalf("claims missing: %v", claims)
		}

		custom, ok = claims["custom"].(map[string]any)

		if !ok || custom["enabled"] != true {
			t.Fatalf("custom claims missing: %v", claims)
		}
	}))

	request.Header.Set("Authorization", "Bearer "+token)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if !called {
		t.Fatal("authenticated handler not called")
	}

	if _, ok := ClaimsFromContext(context.Background()); ok {
		t.Fatal("unexpected claims")
	}
}

// TestInvalidConfiguration rejects missing key resolvers and signing algorithms.
func TestInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct {
		key        jwt.Keyfunc `json:"-"`
		algorithms []string    `json:"-"`
	}{
		{nil, []string{"HS256"}},
		{func(*jwt.Token) (any, error) { return []byte("secret"), nil }, nil},
	} {
		if _, err := New(tc.key, tc.algorithms); err != ErrInvalidConfiguration {
			t.Fatalf("error = %v", err)
		}
	}
}

// TestDuplicateAuthorization rejects ambiguous credentials before invoking the handler.
func TestDuplicateAuthorization(t *testing.T) {
	middleware, err := New(func(*jwt.Token) (any, error) { t.Fatal("unexpected resolver call"); return nil, nil }, []string{"HS256"})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Add("Authorization", "Bearer first")
	request.Header.Add("Authorization", "Bearer second")
	response := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected handler call") })).ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", response.Code)
	}
}

// TestInvalidTokens rejects missing, malformed, or unverifiable credentials.
func TestInvalidTokens(t *testing.T) {
	middleware, err := New(func(*jwt.Token) (any, error) { return []byte("secret"), nil }, []string{"HS256"})
	if err != nil {
		t.Fatal(err)
	}
	for _, header := range []string{"", "Basic secret", "Bearer malformed"} {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if header != "" {
			request.Header.Set("Authorization", header)
		}
		response := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("unexpected handler call") })).ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") != "Bearer" {
			t.Fatalf("invalid challenge: %v", response.Result())
		}
	}
}

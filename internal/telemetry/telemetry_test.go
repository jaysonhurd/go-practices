package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

// lockedBuffer allows both exporters to write concurrently.
type lockedBuffer struct {
	sync.Mutex      `json:"-"`
	strings.Builder `json:"-"`
}

// Write serializes exporter output.
func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.Lock()
	defer b.Unlock()
	return b.Builder.Write(p)
}

// TestTelemetry verifies opt-in spans, metric export, and preservation of handler behavior.
func TestTelemetry(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		var output lockedBuffer
		runtime, err := New(enabled, &output)
		if err != nil {
			t.Fatal(err)
		}
		handler := runtime.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if trace.SpanFromContext(r.Context()).SpanContext().IsValid() != enabled {
				t.Fatal("unexpected span state")
			}
			w.WriteHeader(http.StatusCreated)
		}))
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest("CUSTOM", "/private?token=secret", nil))
		if response.Code != http.StatusCreated {
			t.Fatal(response.Code)
		}
		if err := runtime.Shutdown(context.Background()); err != nil {
			t.Fatal(err)
		}
		text := output.String()
		if enabled && (!strings.Contains(text, "http.server.requests") || !strings.Contains(text, "HTTP request") || !strings.Contains(text, "_OTHER")) {
			t.Fatalf("missing telemetry: %s", text)
		}
		if !enabled && text != "" {
			t.Fatal("telemetry exported without opt-in")
		}
		if strings.Contains(text, "private") || strings.Contains(text, "secret") {
			t.Fatal("request URL leaked")
		}
	}
}

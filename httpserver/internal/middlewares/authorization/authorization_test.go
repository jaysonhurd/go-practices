package authorization

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPassThrough protects the authorization stub's documented behavior.
func TestPassThrough(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/items", nil)
	response := httptest.NewRecorder()
	New()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r != request {
			t.Fatal("request replaced")
		}
		w.WriteHeader(http.StatusAccepted)
	})).ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatal(response.Code)
	}
}

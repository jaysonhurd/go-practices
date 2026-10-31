package logging

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestRequestStatus checks that request logs record the status sent to the client.
func TestRequestStatus(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		want    int
	}{
		{"implicit OK", func(http.ResponseWriter, *http.Request) {}, http.StatusOK},
		{"body", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) }, http.StatusOK},
		{"first final status", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			w.WriteHeader(http.StatusInternalServerError)
		}, http.StatusCreated},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var (
				logs  bytes.Buffer
				entry struct {
					Status int `json:"status"`
				}
				logger   = slog.New(slog.NewJSONHandler(&logs, nil))
				response = httptest.NewRecorder()
			)

			New(logger)(tc.handler).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatal(err)
			}

			if response.Code != tc.want || entry.Status != tc.want {
				t.Fatalf("response status = %d, logged status = %d; want %d", response.Code, entry.Status, tc.want)
			}
		})
	}
}

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

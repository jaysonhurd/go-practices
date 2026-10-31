package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

// clientFunc adapts a function to the consumer's HTTP client interface.
type clientFunc func(*http.Request) (*http.Response, error)

// Do performs the injected test operation.
func (f clientFunc) Do(r *http.Request) (*http.Response, error) { return f(r) }

// trackingBody records response closure, including when reading fails.
type trackingBody struct {
	io.Reader `json:"-"`
	closed    bool `json:"-"`
}

// Close records that the response body was released.
func (b *trackingBody) Close() error { b.closed = true; return nil }

// failingReader simulates a broken response stream.
type failingReader struct{}

// Read returns the injected stream error.
func (failingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// TestPreviousClose covers request construction, cancellation, body ownership, and failures.
func TestPreviousClose(t *testing.T) {
	for _, failRead := range []bool{false, true} {
		body := &trackingBody{Reader: strings.NewReader("response")}
		if failRead {
			body.Reader = failingReader{}
		}
		ctx, cancel := context.WithCancel(context.Background())
		client := clientFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodGet || r.URL.Path != "/v2/aggs/ticker/AAPL/prev" || r.URL.Query().Get(APIKEY) != "test & key" || r.URL.Query().Get("adjusted") != "true" {
				t.Fatalf("unexpected request: %v", r)
			}
			if r.Context() != ctx {
				t.Fatal("caller context lost")
			}
			return &http.Response{Body: body}, nil
		})
		result, err := previousClose(ctx, client, "https://example.com/", "test & key", "AAPL")
		cancel()
		if !body.closed {
			t.Fatal("response body not closed")
		}
		if failRead {
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("read error: %v", err)
			}
		} else if err != nil || result != "response" {
			t.Fatalf("result=%q err=%v", result, err)
		}
	}
	failure := errors.New("transport failed")
	client := clientFunc(func(*http.Request) (*http.Response, error) { return nil, failure })
	if _, err := previousClose(context.Background(), client, "https://example.com", "key", "AAPL"); !errors.Is(err, failure) {
		t.Fatal(err)
	}
	if _, err := previousClose(context.Background(), nil, "https://example.com", "", "AAPL"); !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatal(err)
	}
	if _, err := previousClose(context.Background(), nil, "://invalid", "key", "AAPL"); err == nil {
		t.Fatal("invalid URL accepted")
	}
	t.Setenv("POLYGON_API_KEY", "")
	if _, err := PreviousClose("AAPL"); !errors.Is(err, ErrAPIKeyRequired) {
		t.Fatal(err)
	}
}

// TestServeLifecycle verifies cancellation, startup failures, and graceful shutdown.
func TestServeLifecycle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := serve(ctx, &http.Server{}); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if err := serve(context.Background(), &http.Server{Addr: "invalid-address"}); err == nil {
		t.Fatal("expected bind error")
	}

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	started := make(chan struct{})
	server := &http.Server{Addr: "127.0.0.1:0", BaseContext: func(net.Listener) context.Context { close(started); return ctx }}
	t.Cleanup(func() { _ = server.Close() })
	result := make(chan error, 1)
	go func() { result <- serve(ctx, server) }()
	select {
	case <-started:
	case err := <-result:
		t.Fatalf("unexpected exit: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("server did not start")
	}
	cancel()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown timed out")
	}
}

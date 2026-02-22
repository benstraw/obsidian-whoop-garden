package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// newTestClient builds a Client pointing at a local test server.
func newTestClient(srv *httptest.Server) *Client {
	return &Client{
		accessToken: "test-token",
		baseURL:     srv.URL,
		httpClient:  &http.Client{Timeout: 5 * time.Second},
	}
}

func TestGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"records":[],"next_token":""}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get("/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"records":[],"next_token":""}` {
		t.Errorf("unexpected body: %q", body)
	}
}

func TestGet_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get("/missing", nil)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGet_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get("/error", nil)
	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestGet_QueryParams(t *testing.T) {
	var received url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.URL.Query()
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	params := url.Values{}
	params.Set("start", "2026-01-01")
	params.Set("end", "2026-01-02")

	if _, err := c.Get("/path", params); err != nil {
		t.Fatal(err)
	}
	if received.Get("start") != "2026-01-01" {
		t.Errorf("query param 'start' = %q, want 2026-01-01", received.Get("start"))
	}
	if received.Get("end") != "2026-01-02" {
		t.Errorf("query param 'end' = %q, want 2026-01-02", received.Get("end"))
	}
}

// TestGet_RateLimitRetry verifies the client retries on 429 and eventually succeeds.
// Uses -short to skip the ~1s sleep in fast CI runs.
func TestGet_RateLimitRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping rate-limit retry test in short mode (involves real sleep)")
	}

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	body, err := c.Get("/rate", nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("unexpected body: %q", body)
	}
	if attempts != 3 {
		t.Errorf("server received %d attempts, want 3", attempts)
	}
}

// TestGet_RateLimitExhausted verifies the error message when all retries are consumed.
// Skipped by default because it sleeps 1+2+4 = 7 seconds.
func TestGet_RateLimitExhausted(t *testing.T) {
	t.Skip("skipping: requires ~7s of real sleep; refactor client to accept injectable sleep fn to enable this")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Get("/always-rate-limited", nil)
	if err == nil {
		t.Error("expected error after exhausting retries")
	}
}

func TestGet_PathAppended(t *testing.T) {
	var receivedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	if _, err := c.Get("/activity/sleep", nil); err != nil {
		t.Fatal(err)
	}
	if receivedPath != "/activity/sleep" {
		t.Errorf("server received path %q, want /activity/sleep", receivedPath)
	}
}

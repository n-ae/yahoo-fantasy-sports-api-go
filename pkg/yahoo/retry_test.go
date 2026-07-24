package yahoo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestRetryAfterDelayParsing(t *testing.T) {
	if d, ok := retryAfterDelay("2"); !ok || d != 2*time.Second {
		t.Errorf("seconds: got (%v, %v), want (2s, true)", d, ok)
	}
	if _, ok := retryAfterDelay(""); ok {
		t.Error("empty header should not yield a delay")
	}
	if _, ok := retryAfterDelay("not-a-date"); ok {
		t.Error("garbage header should not yield a delay")
	}
	if _, ok := retryAfterDelay("-5"); ok {
		t.Error("negative seconds should not yield a delay")
	}
}

func TestIsRetryableStatus(t *testing.T) {
	retryable := []int{429, 500, 502, 503, 504}
	for _, code := range retryable {
		if !isRetryableStatus(code) {
			t.Errorf("status %d should be retryable", code)
		}
	}
	for _, code := range []int{200, 400, 401, 403, 404} {
		if isRetryableStatus(code) {
			t.Errorf("status %d should not be retryable", code)
		}
	}
}

func newTokenClient(baseURL string) *Client {
	c, _ := NewClient(WithCredentials("k", "s"))
	c.baseURL = baseURL
	c.accessToken = "token"
	return c
}

func TestMakeRequestRetriesOn503ThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	body, err := newTokenClient(srv.URL).makeRequest(context.Background(), "x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, want ok", body)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server calls = %d, want 2 (1 fail + 1 retry)", got)
	}
}

func TestMakeRequestRetriesOn429WithRetryAfter(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	if _, err := newTokenClient(srv.URL).makeRequest(context.Background(), "x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("server calls = %d, want 2", got)
	}
}

func TestMakeRequestExhaustsRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	_, err := newTokenClient(srv.URL).makeRequest(context.Background(), "x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError after exhausting retries, got %v", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, want 429", apiErr.StatusCode)
	}
	// defaultMaxRetries retries => defaultMaxRetries+1 total attempts.
	if got := atomic.LoadInt32(&calls); got != int32(defaultMaxRetries+1) {
		t.Errorf("server calls = %d, want %d", got, defaultMaxRetries+1)
	}
}

func TestRetryPolicyValidation(t *testing.T) {
	bad := map[string]RetryPolicy{
		"zero value":        {},
		"negative retries":  {MaxRetries: -1, BaseBackoff: time.Second, MaxBackoff: time.Second},
		"zero base backoff": {MaxRetries: 1, BaseBackoff: 0, MaxBackoff: time.Second},
		"max below base":    {MaxRetries: 1, BaseBackoff: 2 * time.Second, MaxBackoff: time.Second},
	}
	for name, p := range bad {
		if _, err := NewClient(WithRetryPolicy(p)); err == nil {
			t.Errorf("%s: expected validation error, got nil", name)
		}
	}

	good := RetryPolicy{MaxRetries: 1, BaseBackoff: time.Millisecond, MaxBackoff: 2 * time.Millisecond}
	if _, err := NewClient(WithRetryPolicy(good)); err != nil {
		t.Errorf("valid policy rejected: %v", err)
	}
}

// MaxRetries=0 must disable retrying: a transient status returns immediately.
func TestWithRetryPolicyDisablesRetries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c, err := NewClient(
		WithTokens("tok", ""),
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithRetryPolicy(RetryPolicy{MaxRetries: 0, BaseBackoff: time.Millisecond, MaxBackoff: time.Millisecond}),
	)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}

	_, err = c.makeRequest(context.Background(), "x")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 APIError, got %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("server calls = %d, want 1 (no retries)", got)
	}
}

func TestMakeRequestBackoffHonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable) // always transient -> forces backoff
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := newTokenClient(srv.URL).makeRequest(ctx, "x")
	if err == nil {
		t.Fatal("expected error from cancelled backoff")
	}
}

package yahoo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewClientWithOptionsValid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c, err := NewClientWithOptions(
		WithTokens("tok", ""),
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
	)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	body, err := c.makeRequest(context.Background(), "x")
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", body)
	}
}

func TestNewClientWithOptionsValidation(t *testing.T) {
	cases := map[string]Option{
		"nil http client": WithHTTPClient(nil),
		"nil cache":       WithCache(nil),
		"nil sqlite db":   WithSQLiteCache(nil),
		"empty base url":  WithBaseURL(""),
		"nil logger":      WithLogger(nil),
	}
	for name, opt := range cases {
		if _, err := NewClientWithOptions(opt); err == nil {
			t.Errorf("%s: expected construction error, got nil", name)
		}
	}
}

func TestCacheEnabledWithoutCacheErrors(t *testing.T) {
	t.Setenv("YAHOO_ENABLE_CACHE", "true")
	if _, err := NewClientWithOptions(FromEnv()); err == nil {
		t.Error("expected error when caching enabled but no cache configured")
	}
}

func TestFromEnvFillsGapsExplicitWins(t *testing.T) {
	t.Setenv("YAHOO_CONSUMER_KEY", "env-key")
	t.Setenv("YAHOO_ACCESS_TOKEN", "env-token")

	c, err := NewClientWithOptions(
		WithCredentials("explicit-key", "explicit-secret"),
		FromEnv(),
	)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	if c.apiKey != "explicit-key" {
		t.Errorf("apiKey = %q, want explicit-key (explicit wins over env)", c.apiKey)
	}
	if got := c.currentAccessToken(); got != "env-token" {
		t.Errorf("accessToken = %q, want env-token (filled from env)", got)
	}
}

type capturingLogger struct{ msgs []string }

func (l *capturingLogger) Printf(format string, args ...interface{}) {
	l.msgs = append(l.msgs, fmt.Sprintf(format, args...))
}

func TestLoggerObservesRetry(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`ok`))
	}))
	defer srv.Close()

	logger := &capturingLogger{}
	c, err := NewClientWithOptions(
		WithTokens("tok", ""),
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	if _, err := c.makeRequest(context.Background(), "x"); err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if len(logger.msgs) == 0 {
		t.Error("expected a retry log message, got none")
	}
}

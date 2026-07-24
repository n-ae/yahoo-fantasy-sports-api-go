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

func TestClassifySlotEmptyIsUnknown(t *testing.T) {
	if got := classifySlot(""); got != SlotUnknown {
		t.Errorf("classifySlot(\"\") = %q, want %q (absent slot is not a starter)", got, SlotUnknown)
	}
	if got := classifySlot("PG"); got != SlotStarting {
		t.Errorf("classifySlot(\"PG\") = %q, want starting", got)
	}
}

// A roster entry with no selected_position must be Unknown, not starting.
func TestFetchRosterEmptySlotNotStarting(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"fantasy_content":{"team":{"roster":{"players":[
			{"player":{"player_key":"p1","player_id":"1","eligible_positions":[{"position":"PG"}]}}
		]}}}}`))
	}))
	defer srv.Close()

	c, _ := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	roster, err := c.fetchRoster(context.Background(), "454.l.1.t.1")
	if err != nil {
		t.Fatalf("fetchRoster: %v", err)
	}
	if roster[0].SlotState != SlotUnknown || roster[0].IsStarting {
		t.Errorf("empty slot: state=%q starting=%v, want unknown/false", roster[0].SlotState, roster[0].IsStarting)
	}
}

// An unknown sport code must fail locally without a network request.
func TestGameKeyRejectsUnknownSport(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	c, _ := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if _, err := c.GameKey(context.Background(), "soccer", 2026); err == nil {
		t.Fatal("expected error for unknown sport code")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("made %d requests for invalid sport, want 0", got)
	}
}

func TestURLValidation(t *testing.T) {
	bad := []string{
		"", "://bad", "ftp://example.com", "notaurl", "http://",
		"https://example.com/base?x=1",  // query would swallow the endpoint
		"https://example.com/base#frag", // fragment
		"https://user:pass@example.com", // user info
	}
	for _, u := range bad {
		if _, err := NewClient(WithBaseURL(u)); err == nil {
			t.Errorf("WithBaseURL(%q) should error", u)
		}
		if _, err := NewClient(WithTokenURL(u)); err == nil {
			t.Errorf("WithTokenURL(%q) should error", u)
		}
	}
	if _, err := NewClient(WithBaseURL("https://ok.example/fantasy/v2")); err != nil {
		t.Errorf("valid URL rejected: %v", err)
	}
}

func TestAuthComboValidation(t *testing.T) {
	// Invalid combinations must fail at construction.
	bad := map[string][]Option{
		"key without secret":    {WithCredentials("k", "")},
		"secret without key":    {WithCredentials("", "s")},
		"refresh without creds": {WithTokens("a", "refresh-only")},
		"refresh with key only": {WithCredentials("k", ""), WithTokens("a", "r")},
	}
	for name, opts := range bad {
		if _, err := NewClient(opts...); err == nil {
			t.Errorf("%s: expected construction error, got nil", name)
		}
	}

	// Valid combinations must succeed.
	good := map[string][]Option{
		"access token only":          {WithTokens("a", "")},
		"creds + full tokens":        {WithCredentials("k", "s"), WithTokens("a", "r")},
		"creds + refresh, no access": {WithCredentials("k", "s"), WithTokens("", "r")},
		"nothing":                    {}, // deferred config; fails later at request time
	}
	for name, opts := range good {
		if _, err := NewClient(opts...); err != nil {
			t.Errorf("%s: unexpected error: %v", name, err)
		}
	}
}

type errCache struct{}

func (errCache) Get(context.Context, string) ([]byte, bool, error) {
	return nil, false, errors.New("backend down")
}
func (errCache) Set(context.Context, string, []byte, time.Duration) error {
	return errors.New("backend down")
}

// A broken cache must be observable via the logger, not silently swallowed.
func TestCacheErrorsAreLogged(t *testing.T) {
	logger := &capturingLogger{}
	c, err := NewClient(WithTokens("t", ""), WithCache(errCache{}), WithLogger(logger))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	ctx := context.Background()

	if _, ok := cacheGet[[]League](ctx, c, "k"); ok {
		t.Error("errCache should miss")
	}
	cacheSet(ctx, c, "k", []League{{LeagueName: "x"}}, time.Hour)

	if len(logger.msgs) < 2 {
		t.Errorf("expected get+set cache errors logged, got %v", logger.msgs)
	}
}

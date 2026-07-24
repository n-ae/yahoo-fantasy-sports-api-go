package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// A season in the static map must resolve without any HTTP request.
func TestGameKeyStaticFastPath(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c, err := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}

	key, err := c.GameKey(context.Background(), "nba", 2025)
	if err != nil {
		t.Fatalf("GameKey: %v", err)
	}
	if key != "466" {
		t.Errorf("nba 2025 = %q, want 466", key)
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("static path made %d requests, want 0", got)
	}
}

// A season beyond the static map must be discovered from Yahoo, then cached.
func TestGameKeyDiscoveryAndCache(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		if !strings.Contains(r.URL.Path, "games;game_codes=nba;seasons=2026") {
			t.Errorf("unexpected endpoint: %s", r.URL.Path)
		}
		// Representative Yahoo games response: numbered keys + count, game as array.
		w.Write([]byte(`{"fantasy_content":{"games":{
			"0":{"game":[{"game_key":"470","game_id":"470","code":"nba","season":"2026"}]},
			"count":1
		}}}`))
	}))
	defer srv.Close()

	c, err := NewClient(
		WithTokens("t", ""),
		WithBaseURL(srv.URL),
		WithHTTPClient(srv.Client()),
		WithCache(&memCache{}),
	)
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	ctx := context.Background()

	key, err := c.GameKey(ctx, "nba", 2026)
	if err != nil {
		t.Fatalf("GameKey discovery: %v", err)
	}
	if key != "470" {
		t.Errorf("discovered nba 2026 = %q, want 470", key)
	}

	// Second call is served from cache — no additional request.
	if _, err := c.GameKey(ctx, "nba", 2026); err != nil {
		t.Fatalf("cached GameKey: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("made %d requests, want 1 (second served from cache)", got)
	}
}

func TestParseGameKeyShapes(t *testing.T) {
	// game as a single object rather than an array.
	obj := []byte(`{"fantasy_content":{"games":{"0":{"game":{"game_key":"500"}},"count":1}}}`)
	if key, err := parseGameKey(obj); err != nil || key != "500" {
		t.Errorf("object shape: got (%q, %v), want (500, nil)", key, err)
	}

	// no game present.
	empty := []byte(`{"fantasy_content":{"games":{"count":0}}}`)
	if _, err := parseGameKey(empty); err == nil {
		t.Error("empty games should return an error")
	}
}

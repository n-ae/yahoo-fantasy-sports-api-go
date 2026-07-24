package yahoo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// N requests expiring at once must trigger exactly one token refresh, and all
// must succeed on retry with the new token.
func TestRefreshSingleFlight(t *testing.T) {
	var refreshes int32
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&refreshes, 1)
		time.Sleep(20 * time.Millisecond) // widen the race window
		w.Write([]byte(`{"access_token":"new-token","refresh_token":"new-refresh","expires_in":3600}`))
	}))
	defer tokenSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer old-token" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":"token_expired"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer apiSrv.Close()

	c, _ := NewClient(WithCredentials("k", "s"))
	c.baseURL = apiSrv.URL
	c.tokenURL = tokenSrv.URL
	c.accessToken = "old-token"
	c.refreshToken = "old-refresh"

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			_, errs[i] = c.makeRequest(context.Background(), "some/endpoint")
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("request %d failed: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(&refreshes); got != 1 {
		t.Errorf("token refreshes = %d, want 1 (single-flight)", got)
	}
	if c.currentAccessToken() != "new-token" {
		t.Errorf("access token = %q, want new-token", c.currentAccessToken())
	}
}

// A cancelled context must abort the refresh HTTP call.
func TestRefreshHonoursContext(t *testing.T) {
	release := make(chan struct{})
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release // hold the request open until the test releases it
	}))
	// LIFO: release the handler first so Close() does not block on it.
	defer tokenSrv.Close()
	defer close(release)

	c, _ := NewClient(WithCredentials("k", "s"))
	c.tokenURL = tokenSrv.URL
	c.accessToken = "old-token"
	c.refreshToken = "old-refresh"

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- c.refreshIfStale(ctx, "old-token") }()

	time.Sleep(20 * time.Millisecond) // let the refresh request go in-flight
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected error from cancelled refresh, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("refresh did not return after context cancellation")
	}
}

func TestClassifySlot(t *testing.T) {
	cases := map[string]SlotState{
		"PG":   SlotStarting,
		"C":    SlotStarting,
		"Util": SlotStarting,
		"BN":   SlotBench,
		"IL":   SlotInjured,
		"IL+":  SlotInjured,
		"IR":   SlotInjured,
		"NA":   SlotInjured,
	}
	for slot, want := range cases {
		if got := classifySlot(slot); got != want {
			t.Errorf("classifySlot(%q) = %q, want %q", slot, got, want)
		}
	}
}

// A player parked in an injury slot must not be reported as starting.
func TestFetchRosterInjuredNotStarting(t *testing.T) {
	body := `{"fantasy_content":{"team":{"roster":{"players":[
		{"player":{"player_key":"p1","player_id":"1",
			"eligible_positions":[{"position":"PG"},{"position":"SG"}],
			"selected_position":{"position":"IL"}}},
		{"player":{"player_key":"p2","player_id":"2",
			"eligible_positions":[{"position":"C"}],
			"selected_position":{"position":"C"}}}
	]}}}}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer srv.Close()

	c, _ := NewClient(WithCredentials("k", "s"))
	c.baseURL = srv.URL
	c.accessToken = "token"

	roster, err := c.fetchRoster(context.Background(), "teamKey")
	if err != nil {
		t.Fatalf("fetchRoster: %v", err)
	}
	if len(roster) != 2 {
		t.Fatalf("got %d roster entries, want 2", len(roster))
	}

	il := roster[0]
	if il.IsStarting {
		t.Error("player in IL slot reported as starting")
	}
	if il.SlotState != SlotInjured {
		t.Errorf("IL slot state = %q, want %q", il.SlotState, SlotInjured)
	}
	if len(il.EligiblePositions) != 2 {
		t.Errorf("EligiblePositions = %v, want both preserved", il.EligiblePositions)
	}

	active := roster[1]
	if !active.IsStarting || active.SlotState != SlotStarting {
		t.Errorf("active player misclassified: starting=%v state=%q", active.IsStarting, active.SlotState)
	}
}

// A non-retryable non-200 response must surface as a typed *APIError callers
// can inspect, without retrying.
func TestMakeRequestReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("no such resource"))
	}))
	defer srv.Close()

	c, _ := NewClient(WithCredentials("k", "s"))
	c.baseURL = srv.URL
	c.accessToken = "token"

	_, err := c.makeRequest(context.Background(), "league/x/teams")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error is not *APIError: %v", err)
	}
	if apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusNotFound)
	}
	if apiErr.Endpoint != "league/x/teams" {
		t.Errorf("Endpoint = %q, want %q", apiErr.Endpoint, "league/x/teams")
	}
}

// Cache methods must not panic when constructed without a database.
func TestCacheNilDBNoPanic(t *testing.T) {
	cache := &apiCache{db: nil}
	ctx := context.Background()

	if _, ok, err := cache.Get(ctx, "k"); ok || err == nil {
		t.Error("Get with nil db should return an error and ok=false")
	}
	if err := cache.Set(ctx, "k", []byte("v"), 0); err == nil {
		t.Error("Set with nil db should return an error")
	}
}

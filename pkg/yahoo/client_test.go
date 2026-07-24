package yahoo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

	c := NewClient("k", "s", nil)
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

// A non-200 response must surface as a typed *APIError callers can inspect.
func TestMakeRequestReturnsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte("rate limited"))
	}))
	defer srv.Close()

	c := NewClient("k", "s", nil)
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
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, http.StatusTooManyRequests)
	}
	if apiErr.Endpoint != "league/x/teams" {
		t.Errorf("Endpoint = %q, want %q", apiErr.Endpoint, "league/x/teams")
	}
}

// Cache methods must not panic when constructed without a database.
func TestCacheNilDBNoPanic(t *testing.T) {
	cache := &APICache{db: nil}

	if _, err := cache.Get("k"); err == nil {
		t.Error("Get with nil db should return an error")
	}
	if err := cache.Set("k", "v", 0); err == nil {
		t.Error("Set with nil db should return an error")
	}
	if err := cache.Delete("k"); err == nil {
		t.Error("Delete with nil db should return an error")
	}
	if err := cache.CleanExpired(); err == nil {
		t.Error("CleanExpired with nil db should return an error")
	}
}

package yahoo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// WithBaseURL must win over YAHOO_BASE_URL regardless of option order (the
// documented "explicit options take precedence" contract).
func TestFromEnvDoesNotOverrideExplicitBaseURL(t *testing.T) {
	t.Setenv("YAHOO_BASE_URL", "https://env.example")

	// Explicit before FromEnv — the previously-broken order.
	c, err := NewClient(WithBaseURL("https://explicit.example"), FromEnv())
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	if c.baseURL != "https://explicit.example" {
		t.Errorf("baseURL = %q, want explicit value (env must not override)", c.baseURL)
	}

	// Reverse order still yields the explicit value.
	c2, _ := NewClient(FromEnv(), WithBaseURL("https://explicit.example"))
	if c2.baseURL != "https://explicit.example" {
		t.Errorf("reverse order baseURL = %q, want explicit", c2.baseURL)
	}

	// FromEnv alone still applies the env value.
	c3, _ := NewClient(FromEnv())
	if c3.baseURL != "https://env.example" {
		t.Errorf("FromEnv-only baseURL = %q, want env value", c3.baseURL)
	}
}

// A DecodeWarning carrying a parse error must survive a JSON round-trip so a
// model with warnings stays cacheable.
func TestDecodeWarningJSONRoundTrip(t *testing.T) {
	player := convertYahooPlayerToPlayer(mustPlayer(t, `{
		"player_key": "p1",
		"player_points": {"total": "bogus"}
	}`))
	if len(player.DecodeWarnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(player.DecodeWarnings))
	}

	blob, err := json.Marshal(player)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got Player
	if err := json.Unmarshal(blob, &got); err != nil {
		t.Fatalf("unmarshal (this used to fail on the error field): %v", err)
	}
	if len(got.DecodeWarnings) != 1 {
		t.Fatalf("round-tripped warnings = %d, want 1", len(got.DecodeWarnings))
	}
	w := got.DecodeWarnings[0]
	if w.Field != "player_points.total" || w.Value != "bogus" || w.Err == nil {
		t.Errorf("round-tripped warning = %+v, want field/value/err preserved", w)
	}
}

// fetchRoster must populate Roster.TeamID from the team key.
func TestFetchRosterPopulatesTeamID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"fantasy_content":{"team":{"roster":{"players":[
			{"player":{"player_key":"454.p.1","player_id":"1",
				"eligible_positions":[{"position":"PG"}],
				"selected_position":{"position":"PG"}}}
		]}}}}`))
	}))
	defer srv.Close()

	c, err := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}

	roster, err := c.fetchRoster(context.Background(), "454.l.1.t.7")
	if err != nil {
		t.Fatalf("fetchRoster: %v", err)
	}
	if len(roster) != 1 {
		t.Fatalf("got %d entries, want 1", len(roster))
	}
	if roster[0].TeamID != "7" {
		t.Errorf("TeamID = %q, want 7 (extracted from team key)", roster[0].TeamID)
	}
}

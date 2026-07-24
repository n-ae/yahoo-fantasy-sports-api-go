package yahoo

import (
	"encoding/json"
	"testing"
)

func TestDecoderDistinguishesZeroFromMalformed(t *testing.T) {
	var d decoder

	// A real "0" parses cleanly with no warning.
	if got := d.atoi("a", "0"); got != 0 {
		t.Fatalf("atoi(\"0\") = %d, want 0", got)
	}
	// An empty string is an absent field: zero value, no warning.
	if got := d.atoi("b", ""); got != 0 {
		t.Fatalf("atoi(\"\") = %d, want 0", got)
	}
	if len(d.warnings) != 0 {
		t.Fatalf("clean/empty input produced warnings: %v", d.warnings)
	}

	// A malformed value falls back to zero AND records a warning.
	if got := d.atoi("c", "not-a-number"); got != 0 {
		t.Fatalf("atoi(malformed) = %d, want 0", got)
	}
	if len(d.warnings) != 1 {
		t.Fatalf("malformed input produced %d warnings, want 1", len(d.warnings))
	}
	if d.warnings[0].Field != "c" || d.warnings[0].Value != "not-a-number" || d.warnings[0].Err == nil {
		t.Fatalf("unexpected warning: %+v", d.warnings[0])
	}
}

func mustPlayer(t *testing.T, jsonStr string) yahooPlayerData {
	t.Helper()
	var yp yahooPlayerData
	if err := json.Unmarshal([]byte(jsonStr), &yp); err != nil {
		t.Fatalf("unmarshal player fixture: %v", err)
	}
	return yp
}

func TestConvertPlayerRecordsMalformedPoints(t *testing.T) {
	player := convertYahooPlayerToPlayer(mustPlayer(t, `{
		"player_key": "p1",
		"player_points": {"total": "bogus"}
	}`))

	if player.PlayerPoints.Total != 0 {
		t.Errorf("malformed total = %v, want 0 fallback", player.PlayerPoints.Total)
	}
	if len(player.DecodeWarnings) != 1 {
		t.Fatalf("expected 1 decode warning, got %d: %v", len(player.DecodeWarnings), player.DecodeWarnings)
	}
	if player.DecodeWarnings[0].Field != "player_points.total" {
		t.Errorf("warning field = %q, want player_points.total", player.DecodeWarnings[0].Field)
	}

	// A legitimate zero must NOT produce a warning.
	clean := convertYahooPlayerToPlayer(mustPlayer(t, `{
		"player_key": "p2",
		"player_points": {"total": "0"}
	}`))
	if len(clean.DecodeWarnings) != 0 {
		t.Errorf("legitimate zero produced warnings: %v", clean.DecodeWarnings)
	}
}

func TestConvertMatchupBubblesTeamWarnings(t *testing.T) {
	var ym yahooMatchupData
	if err := json.Unmarshal([]byte(`{
		"week": "1",
		"teams": {"team": [
			{"team_key": "t0", "team_points": {"total": "10.5"}},
			{"team_key": "t1", "team_points": {"total": "oops"}}
		]}
	}`), &ym); err != nil {
		t.Fatalf("unmarshal matchup fixture: %v", err)
	}
	m := convertYahooMatchup(ym)

	if len(m.DecodeWarnings) != 1 {
		t.Fatalf("expected 1 bubbled warning, got %d: %v", len(m.DecodeWarnings), m.DecodeWarnings)
	}
	if got := m.DecodeWarnings[0].Field; got != "teams[1].team_points.total" {
		t.Errorf("warning field = %q, want teams[1].team_points.total", got)
	}
}

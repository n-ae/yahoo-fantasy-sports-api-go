package yahoo

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// endpointClient returns a client whose every request receives the given body.
func endpointClient(t *testing.T, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	c, err := NewClient(WithTokens("t", ""), WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatalf("construction failed: %v", err)
	}
	return c
}

func TestGetUserLeaguesParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"users":[{"user":[{"games":[{"game":[{"leagues":[
		{"league":{"league_id":"123","name":"My League","season":"2024","scoring_type":"head","num_teams":12,"current_week":5}}
	]}]}]}]}]}}`)
	leagues, err := c.GetUserLeagues(context.Background(), "454")
	if err != nil {
		t.Fatalf("GetUserLeagues: %v", err)
	}
	if len(leagues) != 1 {
		t.Fatalf("got %d leagues, want 1", len(leagues))
	}
	l := leagues[0]
	if l.LeagueName != "My League" || l.SeasonYear != 2024 || l.NumTeams != 12 || l.YahooGameKey != "454" {
		t.Errorf("league = %+v", l)
	}
}

func TestGetLeagueTeamsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"teams":[
		{"team":{"team_key":"454.l.1.t.1","team_id":"1","name":"Alpha","managers":[{"manager":{"nickname":"Ann"}}],
			"team_standings":{"rank":2,"outcome_totals":{"wins":7,"losses":3,"ties":0}}}}
	]}}}`)
	teams, err := c.GetLeagueTeams(context.Background(), "454.l.1")
	if err != nil {
		t.Fatalf("GetLeagueTeams: %v", err)
	}
	if len(teams) != 1 {
		t.Fatalf("got %d teams, want 1", len(teams))
	}
	if teams[0].TeamName != "Alpha" || teams[0].ManagerName != "Ann" || teams[0].Rank != 2 || teams[0].Wins != 7 {
		t.Errorf("team = %+v", teams[0])
	}
}

func TestGetLeaguePlayersParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"players":[
		{"player":{"player_key":"454.p.1","player_id":"1","name":{"full":"Jane Doe"},"display_position":"PG","eligible_positions":[{"position":"PG"}]}}
	]}}}`)
	players, err := c.GetLeaguePlayers(context.Background(), "454.l.1", PlayerStatusAll, 0, 25)
	if err != nil {
		t.Fatalf("GetLeaguePlayers: %v", err)
	}
	if len(players) != 1 || players[0].PlayerKey != "454.p.1" || players[0].Name.Full != "Jane Doe" {
		t.Errorf("players = %+v", players)
	}
}

func TestGetPlayerStatsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"players":{
		"player":{"player_key":"454.p.1","player_id":"1","name":{"full":"Jane Doe"},"player_points":{"total":"42.5"}}
	}}}}`)
	p, err := c.GetPlayerStats(context.Background(), "454.l.1", "454.p.1", 0)
	if err != nil {
		t.Fatalf("GetPlayerStats: %v", err)
	}
	if p.PlayerKey != "454.p.1" || p.PlayerPoints == nil || p.PlayerPoints.Total != 42.5 {
		t.Errorf("player = %+v points=%+v", p, p.PlayerPoints)
	}
}

func TestGetLeagueStandingsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"standings":{"teams":[
		{"team":{"team_key":"454.l.1.t.1","team_id":"1","name":"Alpha",
			"team_standings":{"rank":"1","outcome_totals":{"wins":"9","losses":"1","ties":"0","percentage":"0.9"},
			"points_for":"1200.5","points_against":"1100.0"}}}
	]}}}}`)
	s, err := c.GetLeagueStandings(context.Background(), "454.l.1")
	if err != nil {
		t.Fatalf("GetLeagueStandings: %v", err)
	}
	if len(s.Teams) != 1 {
		t.Fatalf("got %d standings teams, want 1", len(s.Teams))
	}
	st := s.Teams[0]
	if st.Name != "Alpha" || st.TeamStandings.Rank != 1 || st.TeamStandings.OutcomeTotals.Wins != 9 || st.TeamStandings.PointsFor != 1200.5 {
		t.Errorf("standings team = %+v", st)
	}
}

func TestGetLeagueMatchupsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"scoreboard":{"week":"1","matchups":[
		{"matchup":{"week":"1","status":"postevent","teams":{"team":[
			{"team_key":"t1","team_points":{"total":"110.5"}},
			{"team_key":"t2","team_points":{"total":"98.0"}}
		]}}}
	]}}}}`)
	matchups, err := c.GetLeagueMatchups(context.Background(), "454.l.1", 1)
	if err != nil {
		t.Fatalf("GetLeagueMatchups: %v", err)
	}
	if len(matchups) != 1 || matchups[0].Week != 1 || len(matchups[0].Teams) != 2 {
		t.Fatalf("matchups = %+v", matchups)
	}
	if matchups[0].Teams[0].Points != 110.5 {
		t.Errorf("team0 points = %v, want 110.5", matchups[0].Teams[0].Points)
	}
}

func TestGetLeagueDraftResultsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"draft_results":[
		{"draft_result":{"pick":"1","round":"1","team_key":"454.l.1.t.1","players":{"player":{"player_key":"454.p.1"}}}}
	]}}}`)
	results, err := c.GetLeagueDraftResults(context.Background(), "454.l.1")
	if err != nil {
		t.Fatalf("GetLeagueDraftResults: %v", err)
	}
	if len(results) != 1 || results[0].Pick != 1 || results[0].Round != 1 || results[0].TeamKey != "454.l.1.t.1" {
		t.Errorf("draft results = %+v", results)
	}
}

func TestGetLeagueTransactionsParsesFixture(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"transactions":[
		{"transaction":{"transaction_key":"454.l.1.tr.1","type":"add","status":"successful","timestamp":"1700000000"}}
	]}}}`)
	txns, err := c.GetLeagueTransactions(context.Background(), "454.l.1")
	if err != nil {
		t.Fatalf("GetLeagueTransactions: %v", err)
	}
	if len(txns) != 1 || txns[0].Type != "add" || txns[0].Timestamp != 1700000000 {
		t.Errorf("transactions = %+v", txns)
	}
}

// An empty collection decodes to an empty slice, not an error.
func TestFetchEmptyCollection(t *testing.T) {
	c := endpointClient(t, `{"fantasy_content":{"league":{"teams":[]}}}`)
	teams, err := c.GetLeagueTeams(context.Background(), "454.l.1")
	if err != nil {
		t.Fatalf("empty teams: %v", err)
	}
	if len(teams) != 0 {
		t.Errorf("got %d teams, want 0", len(teams))
	}
}

// A malformed payload surfaces a parse error.
func TestFetchMalformedPayload(t *testing.T) {
	c := endpointClient(t, `{not json`)
	if _, err := c.GetLeagueTeams(context.Background(), "454.l.1"); err == nil {
		t.Error("expected parse error for malformed payload")
	}
}

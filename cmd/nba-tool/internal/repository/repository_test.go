package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// schema is the minimal set of tables the repositories operate on, derived from
// their SQL. Timestamp columns use DATETIME so the driver scans them into
// time.Time.
const schema = `
CREATE TABLE fantasy_leagues (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	yahoo_league_id TEXT NOT NULL,
	yahoo_game_key TEXT,
	league_name TEXT,
	season_year INTEGER,
	scoring_type TEXT,
	scoring_settings TEXT,
	num_teams INTEGER,
	current_week INTEGER,
	start_week INTEGER,
	end_week INTEGER,
	last_synced_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE fantasy_teams (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	league_id INTEGER NOT NULL,
	yahoo_team_id TEXT,
	yahoo_team_key TEXT,
	team_name TEXT,
	manager_name TEXT,
	is_user_team INTEGER DEFAULT 0,
	wins INTEGER DEFAULT 0,
	losses INTEGER DEFAULT 0,
	ties INTEGER DEFAULT 0,
	rank INTEGER DEFAULT 0,
	points_for REAL DEFAULT 0,
	points_against REAL DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE fantasy_rosters (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	team_id INTEGER NOT NULL,
	player_id INTEGER NOT NULL,
	roster_position TEXT,
	selected_position TEXT,
	is_starting INTEGER DEFAULT 0,
	acquisition_type TEXT,
	acquisition_date DATETIME,
	added_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE players (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	yahoo_player_key TEXT NOT NULL
);
`

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	// A single connection so all statements hit the same in-memory database.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestLeagueRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewLeagueRepository(newTestDB(t))

	lg := &League{
		YahooLeagueID: "456", YahooGameKey: "454", LeagueName: "My League",
		SeasonYear: 2024, ScoringType: "head", ScoringSettings: `{"PTS":1}`,
		NumTeams: 12, CurrentWeek: 5,
	}
	if err := repo.Create(ctx, lg); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if lg.ID == 0 {
		t.Fatal("Create did not set ID")
	}

	got, err := repo.GetByYahooID(ctx, "456")
	if err != nil {
		t.Fatalf("GetByYahooID: %v", err)
	}
	if got.LeagueName != "My League" || got.YahooGameKey != "454" || got.NumTeams != 12 {
		t.Errorf("round-trip mismatch: %+v", got)
	}
	if got.LastSyncedAt != nil {
		t.Errorf("LastSyncedAt should be nil before sync, got %v", got.LastSyncedAt)
	}

	if err := repo.UpdateSyncTime(ctx, lg.ID); err != nil {
		t.Fatalf("UpdateSyncTime: %v", err)
	}
	got, _ = repo.GetByYahooID(ctx, "456")
	if got.LastSyncedAt == nil {
		t.Error("LastSyncedAt should be set after UpdateSyncTime")
	}

	all, err := repo.GetAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("GetAll = %d rows, %v; want 1", len(all), err)
	}

	if err := repo.Delete(ctx, lg.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetByYahooID(ctx, "456"); err != sql.ErrNoRows {
		t.Errorf("after Delete, GetByYahooID err = %v, want sql.ErrNoRows", err)
	}
}

func TestTeamRepositoryCRUD(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	teams := NewTeamRepository(db)

	a := &FantasyTeam{LeagueID: 1, YahooTeamID: "1", YahooTeamKey: "454.l.1.t.1", TeamName: "Alpha", Rank: 2, Wins: 3}
	b := &FantasyTeam{LeagueID: 1, YahooTeamID: "2", YahooTeamKey: "454.l.1.t.2", TeamName: "Bravo", Rank: 1, Wins: 5, IsUserTeam: true}
	for _, tm := range []*FantasyTeam{a, b} {
		if err := teams.Create(ctx, tm); err != nil {
			t.Fatalf("Create %s: %v", tm.TeamName, err)
		}
	}

	list, err := teams.GetByLeague(ctx, 1)
	if err != nil {
		t.Fatalf("GetByLeague: %v", err)
	}
	if len(list) != 2 || list[0].TeamName != "Bravo" { // ordered by rank
		t.Errorf("GetByLeague order wrong: %+v", list)
	}

	user, err := teams.GetUserTeam(ctx, 1)
	if err != nil {
		t.Fatalf("GetUserTeam: %v", err)
	}
	if user.TeamName != "Bravo" || !user.IsUserTeam {
		t.Errorf("GetUserTeam = %+v, want Bravo/user", user)
	}

	b.Wins = 6
	b.Rank = 1
	if err := teams.Update(ctx, b); err != nil {
		t.Fatalf("Update: %v", err)
	}
	user, _ = teams.GetUserTeam(ctx, 1)
	if user.Wins != 6 {
		t.Errorf("after Update, wins = %d, want 6", user.Wins)
	}
}

func TestRosterRepository(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	roster := NewRosterRepository(db)

	// GetPlayerIDByYahooKey against the players table.
	if _, err := db.Exec(`INSERT INTO players (yahoo_player_key) VALUES ('nba.p.1234')`); err != nil {
		t.Fatalf("seed player: %v", err)
	}
	pid, err := roster.GetPlayerIDByYahooKey(ctx, "nba.p.1234")
	if err != nil || pid != 1 {
		t.Fatalf("GetPlayerIDByYahooKey = %d, %v; want 1", pid, err)
	}
	if _, err := roster.GetPlayerIDByYahooKey(ctx, "nba.p.missing"); err != sql.ErrNoRows {
		t.Errorf("missing player err = %v, want sql.ErrNoRows", err)
	}

	starter := &RosterEntry{TeamID: 7, PlayerID: 1, RosterPosition: "PG", SelectedPosition: "PG", IsStarting: true}
	bench := &RosterEntry{TeamID: 7, PlayerID: 2, RosterPosition: "SG", SelectedPosition: "BN", IsStarting: false}
	for _, e := range []*RosterEntry{starter, bench} {
		if err := roster.Create(ctx, e); err != nil {
			t.Fatalf("Create roster entry: %v", err)
		}
	}

	entries, err := roster.GetByTeam(ctx, 7)
	if err != nil {
		t.Fatalf("GetByTeam: %v", err)
	}
	if len(entries) != 2 || !entries[0].IsStarting { // starters first
		t.Errorf("GetByTeam order/content wrong: %+v", entries)
	}

	if err := roster.DeleteByTeam(ctx, 7); err != nil {
		t.Fatalf("DeleteByTeam: %v", err)
	}
	entries, _ = roster.GetByTeam(ctx, 7)
	if len(entries) != 0 {
		t.Errorf("after DeleteByTeam, %d entries remain", len(entries))
	}
}

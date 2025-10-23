package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type LeagueRepository struct {
	db *sql.DB
}

type League struct {
	ID               int
	YahooLeagueID    string
	YahooGameKey     string
	LeagueName       string
	SeasonYear       int
	ScoringType      string
	ScoringSettings  string
	NumTeams         int
	CurrentWeek      int
	StartWeek        int
	EndWeek          int
	LastSyncedAt     *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewLeagueRepository(db *sql.DB) *LeagueRepository {
	return &LeagueRepository{db: db}
}

func (r *LeagueRepository) Create(ctx context.Context, league *League) error {
	query := `
		INSERT INTO fantasy_leagues (
			yahoo_league_id, yahoo_game_key, league_name, season_year,
			scoring_type, scoring_settings, num_teams, current_week,
			start_week, end_week
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		league.YahooLeagueID, league.YahooGameKey, league.LeagueName,
		league.SeasonYear, league.ScoringType, league.ScoringSettings,
		league.NumTeams, league.CurrentWeek, league.StartWeek, league.EndWeek,
	)
	if err != nil {
		return fmt.Errorf("failed to create league: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	league.ID = int(id)

	return nil
}

func (r *LeagueRepository) GetByYahooID(ctx context.Context, yahooLeagueID string) (*League, error) {
	query := `
		SELECT id, yahoo_league_id, yahoo_game_key, league_name, season_year,
		       scoring_type, scoring_settings, num_teams, current_week,
		       start_week, end_week, last_synced_at, created_at, updated_at
		FROM fantasy_leagues
		WHERE yahoo_league_id = ?
	`

	league := &League{}
	err := r.db.QueryRowContext(ctx, query, yahooLeagueID).Scan(
		&league.ID, &league.YahooLeagueID, &league.YahooGameKey,
		&league.LeagueName, &league.SeasonYear, &league.ScoringType,
		&league.ScoringSettings, &league.NumTeams, &league.CurrentWeek,
		&league.StartWeek, &league.EndWeek, &league.LastSyncedAt,
		&league.CreatedAt, &league.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return league, nil
}

func (r *LeagueRepository) GetAll(ctx context.Context) ([]*League, error) {
	query := `
		SELECT id, yahoo_league_id, yahoo_game_key, league_name, season_year,
		       scoring_type, scoring_settings, num_teams, current_week,
		       start_week, end_week, last_synced_at, created_at, updated_at
		FROM fantasy_leagues
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var leagues []*League
	for rows.Next() {
		league := &League{}
		err := rows.Scan(
			&league.ID, &league.YahooLeagueID, &league.YahooGameKey,
			&league.LeagueName, &league.SeasonYear, &league.ScoringType,
			&league.ScoringSettings, &league.NumTeams, &league.CurrentWeek,
			&league.StartWeek, &league.EndWeek, &league.LastSyncedAt,
			&league.CreatedAt, &league.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		leagues = append(leagues, league)
	}

	return leagues, nil
}

func (r *LeagueRepository) UpdateSyncTime(ctx context.Context, leagueID int) error {
	query := `UPDATE fantasy_leagues SET last_synced_at = ?, updated_at = ? WHERE id = ?`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, now, leagueID)
	return err
}

func (r *LeagueRepository) Delete(ctx context.Context, leagueID int) error {
	query := `DELETE FROM fantasy_leagues WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, leagueID)
	return err
}

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type TeamRepository struct {
	db *sql.DB
}

type FantasyTeam struct {
	ID            int
	LeagueID      int
	YahooTeamID   string
	YahooTeamKey  string
	TeamName      string
	ManagerName   string
	IsUserTeam    bool
	Wins          int
	Losses        int
	Ties          int
	Rank          int
	PointsFor     float64
	PointsAgainst float64
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, team *FantasyTeam) error {
	query := `
		INSERT INTO fantasy_teams (
			league_id, yahoo_team_id, yahoo_team_key, team_name, manager_name,
			is_user_team, wins, losses, ties, rank, points_for, points_against
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		team.LeagueID, team.YahooTeamID, team.YahooTeamKey, team.TeamName,
		team.ManagerName, team.IsUserTeam, team.Wins, team.Losses, team.Ties,
		team.Rank, team.PointsFor, team.PointsAgainst,
	)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	team.ID = int(id)

	return nil
}

func (r *TeamRepository) GetByLeague(ctx context.Context, leagueID int) ([]*FantasyTeam, error) {
	query := `
		SELECT id, league_id, yahoo_team_id, yahoo_team_key, team_name,
		       manager_name, is_user_team, wins, losses, ties, rank,
		       points_for, points_against, created_at, updated_at
		FROM fantasy_teams
		WHERE league_id = ?
		ORDER BY rank
	`

	rows, err := r.db.QueryContext(ctx, query, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*FantasyTeam
	for rows.Next() {
		team := &FantasyTeam{}
		err := rows.Scan(
			&team.ID, &team.LeagueID, &team.YahooTeamID, &team.YahooTeamKey,
			&team.TeamName, &team.ManagerName, &team.IsUserTeam, &team.Wins,
			&team.Losses, &team.Ties, &team.Rank, &team.PointsFor,
			&team.PointsAgainst, &team.CreatedAt, &team.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (r *TeamRepository) GetUserTeam(ctx context.Context, leagueID int) (*FantasyTeam, error) {
	query := `
		SELECT id, league_id, yahoo_team_id, yahoo_team_key, team_name,
		       manager_name, is_user_team, wins, losses, ties, rank,
		       points_for, points_against, created_at, updated_at
		FROM fantasy_teams
		WHERE league_id = ? AND is_user_team = 1
	`

	team := &FantasyTeam{}
	err := r.db.QueryRowContext(ctx, query, leagueID).Scan(
		&team.ID, &team.LeagueID, &team.YahooTeamID, &team.YahooTeamKey,
		&team.TeamName, &team.ManagerName, &team.IsUserTeam, &team.Wins,
		&team.Losses, &team.Ties, &team.Rank, &team.PointsFor,
		&team.PointsAgainst, &team.CreatedAt, &team.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (r *TeamRepository) Update(ctx context.Context, team *FantasyTeam) error {
	query := `
		UPDATE fantasy_teams
		SET team_name = ?, manager_name = ?, wins = ?, losses = ?, ties = ?,
		    rank = ?, points_for = ?, points_against = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query,
		team.TeamName, team.ManagerName, team.Wins, team.Losses, team.Ties,
		team.Rank, team.PointsFor, team.PointsAgainst, now, team.ID,
	)
	return err
}

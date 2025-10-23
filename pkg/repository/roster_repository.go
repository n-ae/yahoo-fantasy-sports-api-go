package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type RosterRepository struct {
	db *sql.DB
}

type RosterEntry struct {
	ID               int
	TeamID           int
	PlayerID         int
	RosterPosition   string
	SelectedPosition string
	IsStarting       bool
	AcquisitionType  string
	AcquisitionDate  *time.Time
	AddedAt          time.Time
	UpdatedAt        time.Time
}

func NewRosterRepository(db *sql.DB) *RosterRepository {
	return &RosterRepository{db: db}
}

func (r *RosterRepository) Create(ctx context.Context, entry *RosterEntry) error {
	query := `
		INSERT INTO fantasy_rosters (
			team_id, player_id, roster_position, selected_position,
			is_starting, acquisition_type, acquisition_date
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		entry.TeamID, entry.PlayerID, entry.RosterPosition,
		entry.SelectedPosition, entry.IsStarting, entry.AcquisitionType,
		entry.AcquisitionDate,
	)
	if err != nil {
		return fmt.Errorf("failed to create roster entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	entry.ID = int(id)

	return nil
}

func (r *RosterRepository) GetByTeam(ctx context.Context, teamID int) ([]*RosterEntry, error) {
	query := `
		SELECT id, team_id, player_id, roster_position, selected_position,
		       is_starting, acquisition_type, acquisition_date, added_at, updated_at
		FROM fantasy_rosters
		WHERE team_id = ?
		ORDER BY is_starting DESC, roster_position
	`

	rows, err := r.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*RosterEntry
	for rows.Next() {
		entry := &RosterEntry{}
		err := rows.Scan(
			&entry.ID, &entry.TeamID, &entry.PlayerID, &entry.RosterPosition,
			&entry.SelectedPosition, &entry.IsStarting, &entry.AcquisitionType,
			&entry.AcquisitionDate, &entry.AddedAt, &entry.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (r *RosterRepository) DeleteByTeam(ctx context.Context, teamID int) error {
	query := `DELETE FROM fantasy_rosters WHERE team_id = ?`
	_, err := r.db.ExecContext(ctx, query, teamID)
	return err
}

func (r *RosterRepository) GetPlayerIDByYahooKey(ctx context.Context, yahooPlayerKey string) (int, error) {
	query := `SELECT id FROM players WHERE yahoo_player_key = ?`
	var playerID int
	err := r.db.QueryRowContext(ctx, query, yahooPlayerKey).Scan(&playerID)
	return playerID, err
}

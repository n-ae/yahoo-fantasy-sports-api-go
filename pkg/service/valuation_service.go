package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
)

type ValuationService struct {
	db *sql.DB
}

type PlayerValue struct {
	PlayerID         int
	LeagueID         int
	FPG              float64
	ZScore           float64
	PositionRank     int
	OverallRank      int
	ScarcityMultiplier float64
	Projections      CategoryProjections
}

type CategoryProjections struct {
	PTS    float64
	REB    float64
	AST    float64
	STL    float64
	BLK    float64
	TO     float64
	FGPct  float64
	FTPct  float64
	TPM    float64
}

type ScoringSettings struct {
	PTS   float64 `json:"PTS"`
	REB   float64 `json:"REB"`
	AST   float64 `json:"AST"`
	STL   float64 `json:"STL"`
	BLK   float64 `json:"BLK"`
	TO    float64 `json:"TO"`
	TPM   float64 `json:"3PM"`
	FGPct float64 `json:"FG%"`
	FTPct float64 `json:"FT%"`
}

func NewValuationService(db *sql.DB) *ValuationService {
	return &ValuationService{db: db}
}

func (s *ValuationService) CalculateAllPlayerValues(ctx context.Context, leagueID int) error {
	league, err := s.getLeague(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("failed to get league: %w", err)
	}

	var scoringSettings ScoringSettings
	if err := json.Unmarshal([]byte(league.ScoringSettings), &scoringSettings); err != nil {
		return fmt.Errorf("failed to parse scoring settings: %w", err)
	}

	players, err := s.getActivePlayersWithStats(ctx)
	if err != nil {
		return fmt.Errorf("failed to get players: %w", err)
	}

	var playerValues []PlayerValue
	for _, player := range players {
		value := s.calculatePlayerValue(player, scoringSettings)
		value.LeagueID = leagueID
		playerValues = append(playerValues, value)
	}

	if err := s.calculateZScores(playerValues); err != nil {
		return fmt.Errorf("failed to calculate z-scores: %w", err)
	}

	s.applyPositionScarcity(ctx, playerValues)

	s.rankPlayers(playerValues)

	if err := s.savePlayerProjections(ctx, playerValues); err != nil {
		return fmt.Errorf("failed to save projections: %w", err)
	}

	return nil
}

type PlayerStats struct {
	PlayerID         int
	PrimaryPosition  string
	PointsPerGame    float64
	ReboundsPerGame  float64
	AssistsPerGame   float64
	StealsPerGame    float64
	BlocksPerGame    float64
	TurnoversPerGame float64
	FGPercentage     float64
	FTPercentage     float64
	ThreePointersMade float64
}

func (s *ValuationService) calculatePlayerValue(player PlayerStats, settings ScoringSettings) PlayerValue {
	fpg := (player.PointsPerGame * settings.PTS) +
		(player.ReboundsPerGame * settings.REB) +
		(player.AssistsPerGame * settings.AST) +
		(player.StealsPerGame * settings.STL) +
		(player.BlocksPerGame * settings.BLK) +
		(player.TurnoversPerGame * settings.TO) +
		(player.ThreePointersMade * settings.TPM)

	return PlayerValue{
		PlayerID: player.PlayerID,
		FPG:      fpg,
		Projections: CategoryProjections{
			PTS:   player.PointsPerGame,
			REB:   player.ReboundsPerGame,
			AST:   player.AssistsPerGame,
			STL:   player.StealsPerGame,
			BLK:   player.BlocksPerGame,
			TO:    player.TurnoversPerGame,
			FGPct: player.FGPercentage,
			FTPct: player.FTPercentage,
			TPM:   player.ThreePointersMade,
		},
	}
}

func (s *ValuationService) calculateZScores(players []PlayerValue) error {
	if len(players) == 0 {
		return nil
	}

	mean, stdDev := s.calculateStats(players)

	for i := range players {
		if stdDev > 0 {
			players[i].ZScore = (players[i].FPG - mean) / stdDev
		}
	}

	return nil
}

func (s *ValuationService) calculateStats(players []PlayerValue) (mean, stdDev float64) {
	if len(players) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, p := range players {
		sum += p.FPG
	}
	mean = sum / float64(len(players))

	variance := 0.0
	for _, p := range players {
		diff := p.FPG - mean
		variance += diff * diff
	}
	variance /= float64(len(players))
	stdDev = math.Sqrt(variance)

	return mean, stdDev
}

func (s *ValuationService) applyPositionScarcity(ctx context.Context, players []PlayerValue) {
	scarcityMap := map[string]float64{
		"PG": 1.0,
		"SG": 1.0,
		"SF": 1.1,
		"PF": 1.1,
		"C":  1.3,
	}

	for i := range players {
		position := s.getPlayerPosition(ctx, players[i].PlayerID)
		if multiplier, ok := scarcityMap[position]; ok {
			players[i].ScarcityMultiplier = multiplier
		} else {
			players[i].ScarcityMultiplier = 1.0
		}
	}
}

func (s *ValuationService) rankPlayers(players []PlayerValue) {
	for i := range players {
		rank := 1
		for j := range players {
			if players[j].FPG > players[i].FPG {
				rank++
			}
		}
		players[i].OverallRank = rank
	}
}

func (s *ValuationService) savePlayerProjections(ctx context.Context, players []PlayerValue) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	deleteQuery := `DELETE FROM player_projections WHERE league_id = ?`
	if _, err := tx.ExecContext(ctx, deleteQuery, players[0].LeagueID); err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO player_projections (
			player_id, league_id, fpg, proj_pts, proj_reb, proj_ast,
			proj_stl, proj_blk, proj_to, proj_fg_pct, proj_ft_pct, proj_3pm,
			z_score, overall_rank, scarcity_multiplier
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	for _, p := range players {
		_, err := tx.ExecContext(ctx, insertQuery,
			p.PlayerID, p.LeagueID, p.FPG,
			p.Projections.PTS, p.Projections.REB, p.Projections.AST,
			p.Projections.STL, p.Projections.BLK, p.Projections.TO,
			p.Projections.FGPct, p.Projections.FTPct, p.Projections.TPM,
			p.ZScore, p.OverallRank, p.ScarcityMultiplier,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *ValuationService) getLeague(ctx context.Context, leagueID int) (*struct {
	ScoringSettings string
}, error) {
	query := `SELECT scoring_settings FROM fantasy_leagues WHERE id = ?`
	var league struct {
		ScoringSettings string
	}
	err := s.db.QueryRowContext(ctx, query, leagueID).Scan(&league.ScoringSettings)
	return &league, err
}

func (s *ValuationService) getActivePlayersWithStats(ctx context.Context) ([]PlayerStats, error) {
	query := `
		SELECT p.id, COALESCE(pp.code, 'F') as primary_position,
		       COALESCE(s.points_per_game, 0) as ppg,
		       COALESCE(s.rebounds_per_game, 0) as rpg,
		       COALESCE(s.assists_per_game, 0) as apg,
		       COALESCE(s.steals_per_game, 0) as spg,
		       COALESCE(s.blocks_per_game, 0) as bpg,
		       COALESCE(s.turnovers_per_game, 0) as tpg,
		       COALESCE(s.field_goal_percentage, 0) as fgpct,
		       COALESCE(s.free_throw_percentage, 0) as ftpct,
		       COALESCE(s.three_pointers_made, 0) as tpm
		FROM players p
		LEFT JOIN player_positions plp ON p.id = plp.player_id AND plp.is_primary = 1
		LEFT JOIN positions pp ON plp.position_id = pp.id
		LEFT JOIN nba_player_stats s ON p.id = s.player_id AND s.stat_type = 'season' AND s.season = '2024-25'
		WHERE p.is_active = 1
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []PlayerStats
	for rows.Next() {
		var p PlayerStats
		err := rows.Scan(
			&p.PlayerID, &p.PrimaryPosition, &p.PointsPerGame,
			&p.ReboundsPerGame, &p.AssistsPerGame, &p.StealsPerGame,
			&p.BlocksPerGame, &p.TurnoversPerGame, &p.FGPercentage,
			&p.FTPercentage, &p.ThreePointersMade,
		)
		if err != nil {
			return nil, err
		}
		players = append(players, p)
	}

	return players, nil
}

func (s *ValuationService) getPlayerPosition(ctx context.Context, playerID int) string {
	query := `
		SELECT pos.code
		FROM player_positions pp
		JOIN positions pos ON pp.position_id = pos.id
		WHERE pp.player_id = ? AND pp.is_primary = 1
	`
	var position string
	s.db.QueryRowContext(ctx, query, playerID).Scan(&position)
	if position == "" {
		return "F"
	}
	return position
}

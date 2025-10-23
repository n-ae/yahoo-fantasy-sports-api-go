package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
)

type AnalysisService struct {
	db *sql.DB
}

type TeamAnalysis struct {
	TeamID           int
	CategoryScores   map[string]float64
	WeakCategories   []CategoryScore
	StrongCategories []CategoryScore
	PositionNeeds    []string
}

type CategoryScore struct {
	Category string
	ZScore   float64
}

type TeamCategoryTotals struct {
	PTS   float64
	REB   float64
	AST   float64
	STL   float64
	BLK   float64
	TO    float64
	FGPct float64
	FTPct float64
	TPM   float64
}

func NewAnalysisService(db *sql.DB) *AnalysisService {
	return &AnalysisService{db: db}
}

func (s *AnalysisService) AnalyzeAllTeams(ctx context.Context, leagueID int) error {
	teams, err := s.getLeagueTeams(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("failed to get teams: %w", err)
	}

	var teamTotals []struct {
		TeamID int
		Totals TeamCategoryTotals
	}

	for _, teamID := range teams {
		totals, err := s.calculateTeamCategoryTotals(ctx, teamID)
		if err != nil {
			return fmt.Errorf("failed to calculate totals for team %d: %w", teamID, err)
		}
		teamTotals = append(teamTotals, struct {
			TeamID int
			Totals TeamCategoryTotals
		}{teamID, totals})
	}

	for _, team := range teamTotals {
		analysis := s.analyzeTeam(team.TeamID, team.Totals, teamTotals)

		positionNeeds, err := s.analyzePositionNeeds(ctx, team.TeamID)
		if err != nil {
			return fmt.Errorf("failed to analyze position needs: %w", err)
		}
		analysis.PositionNeeds = positionNeeds

		if err := s.saveTeamAnalysis(ctx, analysis); err != nil {
			return fmt.Errorf("failed to save analysis for team %d: %w", team.TeamID, err)
		}
	}

	return nil
}

func (s *AnalysisService) calculateTeamCategoryTotals(ctx context.Context, teamID int) (TeamCategoryTotals, error) {
	query := `
		SELECT
			SUM(proj_pts) as total_pts,
			SUM(proj_reb) as total_reb,
			SUM(proj_ast) as total_ast,
			SUM(proj_stl) as total_stl,
			SUM(proj_blk) as total_blk,
			SUM(proj_to) as total_to,
			AVG(proj_fg_pct) as avg_fg_pct,
			AVG(proj_ft_pct) as avg_ft_pct,
			SUM(proj_3pm) as total_3pm
		FROM fantasy_rosters fr
		JOIN player_projections pp ON fr.player_id = pp.player_id
		WHERE fr.team_id = ? AND fr.is_starting = 1
	`

	var totals TeamCategoryTotals
	err := s.db.QueryRowContext(ctx, query, teamID).Scan(
		&totals.PTS, &totals.REB, &totals.AST, &totals.STL,
		&totals.BLK, &totals.TO, &totals.FGPct, &totals.FTPct, &totals.TPM,
	)

	return totals, err
}

func (s *AnalysisService) analyzeTeam(teamID int, totals TeamCategoryTotals, allTeams []struct {
	TeamID int
	Totals TeamCategoryTotals
}) TeamAnalysis {
	categories := map[string][]float64{
		"PTS":   {},
		"REB":   {},
		"AST":   {},
		"STL":   {},
		"BLK":   {},
		"TO":    {},
		"FG%":   {},
		"FT%":   {},
		"3PM":   {},
	}

	for _, team := range allTeams {
		categories["PTS"] = append(categories["PTS"], team.Totals.PTS)
		categories["REB"] = append(categories["REB"], team.Totals.REB)
		categories["AST"] = append(categories["AST"], team.Totals.AST)
		categories["STL"] = append(categories["STL"], team.Totals.STL)
		categories["BLK"] = append(categories["BLK"], team.Totals.BLK)
		categories["TO"] = append(categories["TO"], team.Totals.TO)
		categories["FG%"] = append(categories["FG%"], team.Totals.FGPct)
		categories["FT%"] = append(categories["FT%"], team.Totals.FTPct)
		categories["3PM"] = append(categories["3PM"], team.Totals.TPM)
	}

	zScores := make(map[string]float64)
	zScores["PTS"] = s.calculateZScore(totals.PTS, categories["PTS"])
	zScores["REB"] = s.calculateZScore(totals.REB, categories["REB"])
	zScores["AST"] = s.calculateZScore(totals.AST, categories["AST"])
	zScores["STL"] = s.calculateZScore(totals.STL, categories["STL"])
	zScores["BLK"] = s.calculateZScore(totals.BLK, categories["BLK"])
	zScores["TO"] = s.calculateZScore(totals.TO, categories["TO"]) * -1
	zScores["FG%"] = s.calculateZScore(totals.FGPct, categories["FG%"])
	zScores["FT%"] = s.calculateZScore(totals.FTPct, categories["FT%"])
	zScores["3PM"] = s.calculateZScore(totals.TPM, categories["3PM"])

	var scores []CategoryScore
	for cat, score := range zScores {
		scores = append(scores, CategoryScore{Category: cat, ZScore: score})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].ZScore < scores[j].ZScore
	})

	weak := scores[:3]
	strong := scores[len(scores)-3:]

	sort.Slice(strong, func(i, j int) bool {
		return strong[i].ZScore > strong[j].ZScore
	})

	return TeamAnalysis{
		TeamID:           teamID,
		CategoryScores:   zScores,
		WeakCategories:   weak,
		StrongCategories: strong,
	}
}

func (s *AnalysisService) calculateZScore(value float64, allValues []float64) float64 {
	if len(allValues) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range allValues {
		sum += v
	}
	mean := sum / float64(len(allValues))

	variance := 0.0
	for _, v := range allValues {
		diff := v - mean
		variance += diff * diff
	}
	stdDev := math.Sqrt(variance / float64(len(allValues)))

	if stdDev == 0 {
		return 0
	}

	return (value - mean) / stdDev
}

func (s *AnalysisService) analyzePositionNeeds(ctx context.Context, teamID int) ([]string, error) {
	query := `
		SELECT pos.code, COUNT(*) as count
		FROM fantasy_rosters fr
		JOIN players p ON fr.player_id = p.id
		JOIN player_positions pp ON p.id = pp.player_id AND pp.is_primary = 1
		JOIN positions pos ON pp.position_id = pos.id
		WHERE fr.team_id = ? AND fr.is_starting = 1
		GROUP BY pos.code
	`

	rows, err := s.db.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	positionCounts := make(map[string]int)
	for rows.Next() {
		var position string
		var count int
		if err := rows.Scan(&position, &count); err != nil {
			return nil, err
		}
		positionCounts[position] = count
	}

	var needs []string
	positions := []string{"PG", "SG", "SF", "PF", "C"}
	for _, pos := range positions {
		if positionCounts[pos] < 2 {
			needs = append(needs, pos)
		}
	}

	return needs, nil
}

func (s *AnalysisService) saveTeamAnalysis(ctx context.Context, analysis TeamAnalysis) error {
	query := `
		INSERT OR REPLACE INTO team_analysis (
			team_id, pts_zscore, reb_zscore, ast_zscore, stl_zscore, blk_zscore,
			to_zscore, fg_pct_zscore, ft_pct_zscore, tpm_zscore,
			weakest_cat_1, weakest_cat_2, weakest_cat_3,
			strongest_cat_1, strongest_cat_2, strongest_cat_3,
			needs_pg, needs_sg, needs_sf, needs_pf, needs_c
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		analysis.TeamID,
		analysis.CategoryScores["PTS"],
		analysis.CategoryScores["REB"],
		analysis.CategoryScores["AST"],
		analysis.CategoryScores["STL"],
		analysis.CategoryScores["BLK"],
		analysis.CategoryScores["TO"],
		analysis.CategoryScores["FG%"],
		analysis.CategoryScores["FT%"],
		analysis.CategoryScores["3PM"],
		analysis.WeakCategories[0].Category,
		analysis.WeakCategories[1].Category,
		analysis.WeakCategories[2].Category,
		analysis.StrongCategories[0].Category,
		analysis.StrongCategories[1].Category,
		analysis.StrongCategories[2].Category,
		contains(analysis.PositionNeeds, "PG"),
		contains(analysis.PositionNeeds, "SG"),
		contains(analysis.PositionNeeds, "SF"),
		contains(analysis.PositionNeeds, "PF"),
		contains(analysis.PositionNeeds, "C"),
	)

	return err
}

func (s *AnalysisService) getLeagueTeams(ctx context.Context, leagueID int) ([]int, error) {
	query := `SELECT id FROM fantasy_teams WHERE league_id = ?`
	rows, err := s.db.QueryContext(ctx, query, leagueID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []int
	for rows.Next() {
		var teamID int
		if err := rows.Scan(&teamID); err != nil {
			return nil, err
		}
		teams = append(teams, teamID)
	}

	return teams, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

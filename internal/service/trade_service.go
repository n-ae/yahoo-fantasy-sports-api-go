package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
)

type TradeService struct {
	db            *sql.DB
	evaluator     *EvaluationService
	analysisService *AnalysisService
}

type TradeSuggestion struct {
	ID               int
	LeagueID         int
	TeamAID          int
	TeamAName        string
	TeamAGives       []TradePlayer
	TeamBID          int
	TeamBName        string
	TeamBGives       []TradePlayer
	FairnessScore    float64
	TeamABenefit     string
	TeamBBenefit     string
	Recommendation   string
}

type TradePlayer struct {
	PlayerID   int
	PlayerName string
	Position   string
	FPG        float64
}

type TradeProposal struct {
	LeagueID         int
	TeamAID          int
	TeamBID          int
	TeamAGives       []int
	TeamBGives       []int
	FairnessScore    float64
	TeamAValueChange float64
	TeamBValueChange float64
	TeamABenefits    string
	TeamBBenefits    string
	Source           string
	Status           string
}

func NewTradeService(db *sql.DB, evaluator *EvaluationService, analysisService *AnalysisService) *TradeService {
	return &TradeService{
		db:              db,
		evaluator:       evaluator,
		analysisService: analysisService,
	}
}

func (s *TradeService) GenerateSuggestions(ctx context.Context, teamID int, limit int) ([]*TradeSuggestion, error) {
	leagueID, err := s.getLeagueIDByTeam(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get league ID: %w", err)
	}

	userAnalysis, err := s.getUserTeamAnalysis(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user team analysis: %w", err)
	}

	otherTeams, err := s.getOtherTeams(ctx, leagueID, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get other teams: %w", err)
	}

	var suggestions []*TradeSuggestion

	for _, otherTeam := range otherTeams {
		otherAnalysis, err := s.getUserTeamAnalysis(ctx, otherTeam.TeamID)
		if err != nil {
			continue
		}

		complementScore := s.calculateComplementaryScore(userAnalysis, otherAnalysis)
		if complementScore < 2 {
			continue
		}

		teamSuggestions, err := s.findTradesWithTeam(
			ctx,
			leagueID,
			teamID,
			otherTeam.TeamID,
			userAnalysis,
			otherAnalysis,
		)
		if err != nil {
			continue
		}

		suggestions = append(suggestions, teamSuggestions...)
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].FairnessScore > suggestions[j].FairnessScore
	})

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions, nil
}

func (s *TradeService) findTradesWithTeam(
	ctx context.Context,
	leagueID int,
	teamAID int,
	teamBID int,
	teamAAnalysis *TeamAnalysis,
	teamBAnalysis *TeamAnalysis,
) ([]*TradeSuggestion, error) {
	teamAPlayers, err := s.getRosterWithProjections(ctx, leagueID, teamAID)
	if err != nil {
		return nil, err
	}

	teamBPlayers, err := s.getRosterWithProjections(ctx, leagueID, teamBID)
	if err != nil {
		return nil, err
	}

	var suggestions []*TradeSuggestion

	for _, playerA := range teamAPlayers {
		for _, playerB := range teamBPlayers {
			if !s.isGoodFit(playerA, playerB, teamAAnalysis, teamBAnalysis) {
				continue
			}

			evaluation, err := s.evaluator.EvaluateTrade(
				ctx,
				leagueID,
				teamAID,
				[]int{playerB.PlayerID},
				teamBID,
				[]int{playerA.PlayerID},
			)
			if err != nil {
				continue
			}

			if !evaluation.IsFair {
				continue
			}

			teamAName, _ := s.getTeamName(ctx, teamAID)
			teamBName, _ := s.getTeamName(ctx, teamBID)

			suggestion := &TradeSuggestion{
				LeagueID:  leagueID,
				TeamAID:   teamAID,
				TeamAName: teamAName,
				TeamAGives: []TradePlayer{{
					PlayerID:   playerA.PlayerID,
					PlayerName: playerA.PlayerName,
					Position:   playerA.Position,
					FPG:        playerA.FPG,
				}},
				TeamBID:   teamBID,
				TeamBName: teamBName,
				TeamBGives: []TradePlayer{{
					PlayerID:   playerB.PlayerID,
					PlayerName: playerB.PlayerName,
					Position:   playerB.Position,
					FPG:        playerB.FPG,
				}},
				FairnessScore:  evaluation.FairnessScore,
				TeamABenefit:   s.formatBenefit(evaluation.TeamAImpact),
				TeamBBenefit:   s.formatBenefit(evaluation.TeamBImpact),
				Recommendation: evaluation.Recommendation,
			}

			suggestions = append(suggestions, suggestion)
		}
	}

	return suggestions, nil
}

func (s *TradeService) calculateComplementaryScore(
	teamA *TeamAnalysis,
	teamB *TeamAnalysis,
) int {
	score := 0

	teamAWeakMap := make(map[string]bool)
	teamAStrongMap := make(map[string]bool)
	for _, cat := range teamA.WeakCategories {
		teamAWeakMap[cat.Category] = true
	}
	for _, cat := range teamA.StrongCategories {
		teamAStrongMap[cat.Category] = true
	}

	for _, cat := range teamB.WeakCategories {
		if teamAStrongMap[cat.Category] {
			score++
		}
	}

	for _, cat := range teamB.StrongCategories {
		if teamAWeakMap[cat.Category] {
			score++
		}
	}

	return score
}

func (s *TradeService) isGoodFit(
	playerA RosterPlayer,
	playerB RosterPlayer,
	teamAAnalysis *TeamAnalysis,
	teamBAnalysis *TeamAnalysis,
) bool {
	valueDiff := playerA.FPG - playerB.FPG
	avgValue := (playerA.FPG + playerB.FPG) / 2.0

	if avgValue == 0 {
		return false
	}

	percentDiff := (valueDiff / avgValue) * 100.0
	if percentDiff < -15.0 || percentDiff > 15.0 {
		return false
	}

	return true
}

func (s *TradeService) formatBenefit(impact TradeImpact) string {
	if len(impact.CategoryImprovements) == 0 {
		return "No significant benefit"
	}

	benefits := "Improves: "
	for i, imp := range impact.CategoryImprovements {
		if i > 2 {
			break
		}
		if i > 0 {
			benefits += ", "
		}
		benefits += fmt.Sprintf("%s (+%.1f)", imp.Category, imp.Change)
	}

	if impact.PositionImpact != "" && impact.PositionImpact != "Neutral position impact" {
		benefits += fmt.Sprintf(" | %s", impact.PositionImpact)
	}

	return benefits
}

func (s *TradeService) SaveProposal(ctx context.Context, proposal *TradeProposal) error {
	tradeDetails := map[string][]int{
		"team_a_gives": proposal.TeamAGives,
		"team_b_gives": proposal.TeamBGives,
	}
	detailsJSON, err := json.Marshal(tradeDetails)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO trade_proposals (
			league_id, team_a_id, team_b_id, trade_details,
			fairness_score, team_a_value_change, team_b_value_change,
			team_a_benefits, team_b_benefits, source, status
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		proposal.LeagueID, proposal.TeamAID, proposal.TeamBID, string(detailsJSON),
		proposal.FairnessScore, proposal.TeamAValueChange, proposal.TeamBValueChange,
		proposal.TeamABenefits, proposal.TeamBBenefits, proposal.Source, proposal.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to save proposal: %w", err)
	}

	_, err = result.LastInsertId()
	return err
}

func (s *TradeService) GetProposalsByTeam(ctx context.Context, teamID int) ([]*TradeSuggestion, error) {
	query := `
		SELECT id, league_id, team_a_id, team_b_id, trade_details,
		       fairness_score, team_a_benefits, team_b_benefits
		FROM trade_proposals
		WHERE (team_a_id = ? OR team_b_id = ?) AND status != 'rejected'
		ORDER BY suggested_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, teamID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var suggestions []*TradeSuggestion
	for rows.Next() {
		var id, leagueID, teamAID, teamBID int
		var detailsJSON, teamABenefits, teamBBenefits string
		var fairnessScore float64

		err := rows.Scan(
			&id, &leagueID, &teamAID, &teamBID, &detailsJSON,
			&fairnessScore, &teamABenefits, &teamBBenefits,
		)
		if err != nil {
			continue
		}

		suggestion := &TradeSuggestion{
			ID:            id,
			LeagueID:      leagueID,
			TeamAID:       teamAID,
			TeamBID:       teamBID,
			FairnessScore: fairnessScore,
			TeamABenefit:  teamABenefits,
			TeamBBenefit:  teamBBenefits,
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

type RosterPlayer struct {
	PlayerID   int
	PlayerName string
	Position   string
	FPG        float64
	IsStarting bool
}

func (s *TradeService) getRosterWithProjections(
	ctx context.Context,
	leagueID int,
	teamID int,
) ([]RosterPlayer, error) {
	query := `
		SELECT p.id, p.full_name, COALESCE(pos.code, 'F') as position,
		       pp.fpg, fr.is_starting
		FROM fantasy_rosters fr
		JOIN players p ON fr.player_id = p.id
		JOIN player_projections pp ON p.id = pp.player_id AND pp.league_id = ?
		LEFT JOIN player_positions plp ON p.id = plp.player_id AND plp.is_primary = 1
		LEFT JOIN positions pos ON plp.position_id = pos.id
		WHERE fr.team_id = ? AND fr.is_starting = 1
	`

	rows, err := s.db.QueryContext(ctx, query, leagueID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var players []RosterPlayer
	for rows.Next() {
		var p RosterPlayer
		err := rows.Scan(&p.PlayerID, &p.PlayerName, &p.Position, &p.FPG, &p.IsStarting)
		if err != nil {
			continue
		}
		players = append(players, p)
	}

	return players, nil
}

func (s *TradeService) getUserTeamAnalysis(ctx context.Context, teamID int) (*TeamAnalysis, error) {
	query := `
		SELECT pts_zscore, reb_zscore, ast_zscore, stl_zscore, blk_zscore,
		       to_zscore, fg_pct_zscore, ft_pct_zscore, tpm_zscore,
		       weakest_cat_1, weakest_cat_2, weakest_cat_3,
		       strongest_cat_1, strongest_cat_2, strongest_cat_3
		FROM team_analysis
		WHERE team_id = ?
	`

	var analysis TeamAnalysis
	analysis.TeamID = teamID
	analysis.CategoryScores = make(map[string]float64)

	var weak1, weak2, weak3, strong1, strong2, strong3 string
	var pts, reb, ast, stl, blk, to, fgPct, ftPct, tpm float64

	err := s.db.QueryRowContext(ctx, query, teamID).Scan(
		&pts, &reb, &ast, &stl, &blk, &to, &fgPct, &ftPct, &tpm,
		&weak1, &weak2, &weak3,
		&strong1, &strong2, &strong3,
	)
	if err == nil {
		analysis.CategoryScores["PTS"] = pts
		analysis.CategoryScores["REB"] = reb
		analysis.CategoryScores["AST"] = ast
		analysis.CategoryScores["STL"] = stl
		analysis.CategoryScores["BLK"] = blk
		analysis.CategoryScores["TO"] = to
		analysis.CategoryScores["FG%"] = fgPct
		analysis.CategoryScores["FT%"] = ftPct
		analysis.CategoryScores["3PM"] = tpm
	}
	if err != nil {
		return nil, err
	}

	analysis.WeakCategories = []CategoryScore{
		{Category: weak1, ZScore: analysis.CategoryScores[weak1]},
		{Category: weak2, ZScore: analysis.CategoryScores[weak2]},
		{Category: weak3, ZScore: analysis.CategoryScores[weak3]},
	}

	analysis.StrongCategories = []CategoryScore{
		{Category: strong1, ZScore: analysis.CategoryScores[strong1]},
		{Category: strong2, ZScore: analysis.CategoryScores[strong2]},
		{Category: strong3, ZScore: analysis.CategoryScores[strong3]},
	}

	return &analysis, nil
}

func (s *TradeService) getLeagueIDByTeam(ctx context.Context, teamID int) (int, error) {
	query := `SELECT league_id FROM fantasy_teams WHERE id = ?`
	var leagueID int
	err := s.db.QueryRowContext(ctx, query, teamID).Scan(&leagueID)
	return leagueID, err
}

func (s *TradeService) getOtherTeams(ctx context.Context, leagueID int, excludeTeamID int) ([]struct {
	TeamID   int
	TeamName string
}, error) {
	query := `
		SELECT id, team_name
		FROM fantasy_teams
		WHERE league_id = ? AND id != ?
	`

	rows, err := s.db.QueryContext(ctx, query, leagueID, excludeTeamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []struct {
		TeamID   int
		TeamName string
	}
	for rows.Next() {
		var team struct {
			TeamID   int
			TeamName string
		}
		if err := rows.Scan(&team.TeamID, &team.TeamName); err != nil {
			continue
		}
		teams = append(teams, team)
	}

	return teams, nil
}

func (s *TradeService) getTeamName(ctx context.Context, teamID int) (string, error) {
	query := `SELECT team_name FROM fantasy_teams WHERE id = ?`
	var name string
	err := s.db.QueryRowContext(ctx, query, teamID).Scan(&name)
	return name, err
}

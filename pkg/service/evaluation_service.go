package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
)

type EvaluationService struct {
	db *sql.DB
}

type TradeImpact struct {
	TeamID               int
	ValueChange          float64
	CategoryImprovements []CategoryChange
	CategoryDeclines     []CategoryChange
	PositionImpact       string
	NetBenefit           float64
}

type CategoryChange struct {
	Category    string
	Change      float64
	PercentChange float64
}

type TradeEvaluation struct {
	TeamAImpact    TradeImpact
	TeamBImpact    TradeImpact
	FairnessScore  float64
	IsFair         bool
	Recommendation string
}

type PlayerProjection struct {
	PlayerID   int
	FPG        float64
	PTS        float64
	REB        float64
	AST        float64
	STL        float64
	BLK        float64
	TO         float64
	FGPct      float64
	FTPct      float64
	TPM        float64
	Position   string
}

func NewEvaluationService(db *sql.DB) *EvaluationService {
	return &EvaluationService{db: db}
}

func (s *EvaluationService) EvaluateTrade(
	ctx context.Context,
	leagueID int,
	teamAID int,
	teamAGives []int,
	teamBID int,
	teamBGives []int,
) (*TradeEvaluation, error) {
	teamAProjections, err := s.getPlayerProjections(ctx, leagueID, teamAGives)
	if err != nil {
		return nil, fmt.Errorf("failed to get team A projections: %w", err)
	}

	teamBProjections, err := s.getPlayerProjections(ctx, leagueID, teamBGives)
	if err != nil {
		return nil, fmt.Errorf("failed to get team B projections: %w", err)
	}

	fairnessScore := s.calculateFairnessScore(teamAProjections, teamBProjections)

	teamAImpact, err := s.calculateTeamImpact(ctx, leagueID, teamAID, teamBProjections, teamAProjections)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate team A impact: %w", err)
	}

	teamBImpact, err := s.calculateTeamImpact(ctx, leagueID, teamBID, teamAProjections, teamBProjections)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate team B impact: %w", err)
	}

	evaluation := &TradeEvaluation{
		TeamAImpact:   teamAImpact,
		TeamBImpact:   teamBImpact,
		FairnessScore: fairnessScore,
		IsFair:        fairnessScore >= 75.0,
	}

	evaluation.Recommendation = s.generateRecommendation(evaluation)

	return evaluation, nil
}

func (s *EvaluationService) calculateFairnessScore(
	teamAPlayers []PlayerProjection,
	teamBPlayers []PlayerProjection,
) float64 {
	teamAValue := s.sumFPG(teamAPlayers)
	teamBValue := s.sumFPG(teamBPlayers)

	if teamAValue == 0 && teamBValue == 0 {
		return 100.0
	}

	avgValue := (teamAValue + teamBValue) / 2.0
	if avgValue == 0 {
		return 0.0
	}

	valueDelta := math.Abs(teamAValue - teamBValue)
	fairnessScore := 100.0 - (valueDelta/avgValue)*100.0

	if fairnessScore < 0 {
		return 0.0
	}
	if fairnessScore > 100 {
		return 100.0
	}

	return fairnessScore
}

func (s *EvaluationService) calculateTeamImpact(
	ctx context.Context,
	leagueID int,
	teamID int,
	playersIn []PlayerProjection,
	playersOut []PlayerProjection,
) (TradeImpact, error) {
	currentTotals, err := s.getTeamCategoryTotals(ctx, teamID)
	if err != nil {
		return TradeImpact{}, err
	}

	afterTotals := s.simulateTrade(currentTotals, playersIn, playersOut)

	categoryChanges := s.calculateCategoryChanges(currentTotals, afterTotals)

	var improvements []CategoryChange
	var declines []CategoryChange

	for _, change := range categoryChanges {
		if change.Category == "TO" {
			if change.Change < 0 {
				improvements = append(improvements, change)
			} else if change.Change > 0 {
				declines = append(declines, change)
			}
		} else {
			if change.Change > 0 {
				improvements = append(improvements, change)
			} else if change.Change < 0 {
				declines = append(declines, change)
			}
		}
	}

	valueChange := s.sumFPG(playersIn) - s.sumFPG(playersOut)

	positionImpact := s.analyzePositionImpact(playersIn, playersOut)

	netBenefit := s.calculateNetBenefit(valueChange, improvements, declines)

	return TradeImpact{
		TeamID:               teamID,
		ValueChange:          valueChange,
		CategoryImprovements: improvements,
		CategoryDeclines:     declines,
		PositionImpact:       positionImpact,
		NetBenefit:           netBenefit,
	}, nil
}

func (s *EvaluationService) simulateTrade(
	current TeamCategoryTotals,
	playersIn []PlayerProjection,
	playersOut []PlayerProjection,
) TeamCategoryTotals {
	result := current

	for _, p := range playersOut {
		result.PTS -= p.PTS
		result.REB -= p.REB
		result.AST -= p.AST
		result.STL -= p.STL
		result.BLK -= p.BLK
		result.TO -= p.TO
		result.TPM -= p.TPM
	}

	for _, p := range playersIn {
		result.PTS += p.PTS
		result.REB += p.REB
		result.AST += p.AST
		result.STL += p.STL
		result.BLK += p.BLK
		result.TO += p.TO
		result.TPM += p.TPM
	}

	return result
}

func (s *EvaluationService) calculateCategoryChanges(
	before TeamCategoryTotals,
	after TeamCategoryTotals,
) []CategoryChange {
	categories := []struct {
		name   string
		before float64
		after  float64
	}{
		{"PTS", before.PTS, after.PTS},
		{"REB", before.REB, after.REB},
		{"AST", before.AST, after.AST},
		{"STL", before.STL, after.STL},
		{"BLK", before.BLK, after.BLK},
		{"TO", before.TO, after.TO},
		{"3PM", before.TPM, after.TPM},
	}

	var changes []CategoryChange
	for _, cat := range categories {
		change := cat.after - cat.before
		percentChange := 0.0
		if cat.before != 0 {
			percentChange = (change / cat.before) * 100.0
		}

		changes = append(changes, CategoryChange{
			Category:      cat.name,
			Change:        change,
			PercentChange: percentChange,
		})
	}

	return changes
}

func (s *EvaluationService) analyzePositionImpact(
	playersIn []PlayerProjection,
	playersOut []PlayerProjection,
) string {
	positionsOut := make(map[string]int)
	positionsIn := make(map[string]int)

	for _, p := range playersOut {
		positionsOut[p.Position]++
	}
	for _, p := range playersIn {
		positionsIn[p.Position]++
	}

	for pos, countOut := range positionsOut {
		countIn := positionsIn[pos]
		if countIn < countOut {
			return fmt.Sprintf("Creates %s gap", pos)
		}
	}

	for pos, countIn := range positionsIn {
		countOut := positionsOut[pos]
		if countIn > countOut {
			return fmt.Sprintf("Fills %s need", pos)
		}
	}

	return "Neutral position impact"
}

func (s *EvaluationService) calculateNetBenefit(
	valueChange float64,
	improvements []CategoryChange,
	declines []CategoryChange,
) float64 {
	benefit := valueChange

	for _, imp := range improvements {
		benefit += math.Abs(imp.Change) * 0.5
	}

	for _, dec := range declines {
		benefit -= math.Abs(dec.Change) * 0.5
	}

	return benefit
}

func (s *EvaluationService) generateRecommendation(eval *TradeEvaluation) string {
	if !eval.IsFair {
		return "Trade is imbalanced. Value difference too large."
	}

	if eval.TeamAImpact.NetBenefit > 2 && eval.TeamBImpact.NetBenefit > 2 {
		return "Strong mutual benefit. Both teams improve."
	}

	if eval.TeamAImpact.NetBenefit > 0 && eval.TeamBImpact.NetBenefit > 0 {
		return "Fair trade with mutual benefit."
	}

	if eval.TeamAImpact.NetBenefit < 0 || eval.TeamBImpact.NetBenefit < 0 {
		return "One team may not benefit sufficiently."
	}

	return "Even trade with minimal impact."
}

func (s *EvaluationService) sumFPG(players []PlayerProjection) float64 {
	total := 0.0
	for _, p := range players {
		total += p.FPG
	}
	return total
}

func (s *EvaluationService) getPlayerProjections(
	ctx context.Context,
	leagueID int,
	playerIDs []int,
) ([]PlayerProjection, error) {
	if len(playerIDs) == 0 {
		return []PlayerProjection{}, nil
	}

	query := `
		SELECT pp.player_id, pp.fpg, pp.proj_pts, pp.proj_reb, pp.proj_ast,
		       pp.proj_stl, pp.proj_blk, pp.proj_to, pp.proj_fg_pct,
		       pp.proj_ft_pct, pp.proj_3pm,
		       COALESCE(pos.code, 'F') as position
		FROM player_projections pp
		JOIN players p ON pp.player_id = p.id
		LEFT JOIN player_positions plp ON p.id = plp.player_id AND plp.is_primary = 1
		LEFT JOIN positions pos ON plp.position_id = pos.id
		WHERE pp.league_id = ? AND pp.player_id IN (` + s.placeholders(len(playerIDs)) + `)`

	args := []interface{}{leagueID}
	for _, id := range playerIDs {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projections []PlayerProjection
	for rows.Next() {
		var p PlayerProjection
		err := rows.Scan(
			&p.PlayerID, &p.FPG, &p.PTS, &p.REB, &p.AST,
			&p.STL, &p.BLK, &p.TO, &p.FGPct, &p.FTPct, &p.TPM, &p.Position,
		)
		if err != nil {
			return nil, err
		}
		projections = append(projections, p)
	}

	return projections, nil
}

func (s *EvaluationService) getTeamCategoryTotals(
	ctx context.Context,
	teamID int,
) (TeamCategoryTotals, error) {
	query := `
		SELECT
			COALESCE(SUM(pp.proj_pts), 0) as total_pts,
			COALESCE(SUM(pp.proj_reb), 0) as total_reb,
			COALESCE(SUM(pp.proj_ast), 0) as total_ast,
			COALESCE(SUM(pp.proj_stl), 0) as total_stl,
			COALESCE(SUM(pp.proj_blk), 0) as total_blk,
			COALESCE(SUM(pp.proj_to), 0) as total_to,
			COALESCE(AVG(pp.proj_fg_pct), 0) as avg_fg_pct,
			COALESCE(AVG(pp.proj_ft_pct), 0) as avg_ft_pct,
			COALESCE(SUM(pp.proj_3pm), 0) as total_3pm
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

func (s *EvaluationService) placeholders(count int) string {
	if count == 0 {
		return ""
	}
	result := "?"
	for i := 1; i < count; i++ {
		result += ",?"
	}
	return result
}

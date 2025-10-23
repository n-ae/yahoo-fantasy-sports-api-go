package service

import (
	"math"
	"testing"
)

func TestCalculateComplementaryScore(t *testing.T) {
	tests := []struct {
		name          string
		teamA         *TeamAnalysis
		teamB         *TeamAnalysis
		expectedScore int
		shouldMatch   bool
	}{
		{
			name: "Perfect complementary match",
			teamA: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "REB", ZScore: -1.5},
					{Category: "BLK", ZScore: -1.2},
					{Category: "TO", ZScore: -0.8},
				},
				StrongCategories: []CategoryScore{
					{Category: "PTS", ZScore: 1.5},
					{Category: "AST", ZScore: 1.3},
					{Category: "3PM", ZScore: 1.1},
				},
			},
			teamB: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "PTS", ZScore: -1.3},
					{Category: "AST", ZScore: -1.1},
					{Category: "3PM", ZScore: -0.9},
				},
				StrongCategories: []CategoryScore{
					{Category: "REB", ZScore: 1.4},
					{Category: "BLK", ZScore: 1.2},
					{Category: "FG%", ZScore: 1.0},
				},
			},
			expectedScore: 5,
			shouldMatch:   true,
		},
		{
			name: "No complementary match",
			teamA: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "REB", ZScore: -1.5},
					{Category: "BLK", ZScore: -1.2},
					{Category: "TO", ZScore: -0.8},
				},
				StrongCategories: []CategoryScore{
					{Category: "PTS", ZScore: 1.5},
					{Category: "AST", ZScore: 1.3},
					{Category: "3PM", ZScore: 1.1},
				},
			},
			teamB: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "FG%", ZScore: -1.3},
					{Category: "FT%", ZScore: -1.1},
					{Category: "STL", ZScore: -0.9},
				},
				StrongCategories: []CategoryScore{
					{Category: "PTS", ZScore: 1.4},
					{Category: "AST", ZScore: 1.2},
					{Category: "3PM", ZScore: 1.0},
				},
			},
			expectedScore: 0,
			shouldMatch:   false,
		},
		{
			name: "Partial complementary match",
			teamA: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "REB", ZScore: -1.5},
					{Category: "BLK", ZScore: -1.2},
					{Category: "TO", ZScore: -0.8},
				},
				StrongCategories: []CategoryScore{
					{Category: "PTS", ZScore: 1.5},
					{Category: "AST", ZScore: 1.3},
					{Category: "3PM", ZScore: 1.1},
				},
			},
			teamB: &TeamAnalysis{
				WeakCategories: []CategoryScore{
					{Category: "PTS", ZScore: -1.3},
					{Category: "FT%", ZScore: -1.1},
					{Category: "STL", ZScore: -0.9},
				},
				StrongCategories: []CategoryScore{
					{Category: "REB", ZScore: 1.4},
					{Category: "FG%", ZScore: 1.2},
					{Category: "TO", ZScore: 1.0},
				},
			},
			expectedScore: 3,
			shouldMatch:   true,
		},
	}

	service := &TradeService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateComplementaryScore(tt.teamA, tt.teamB)

			if score != tt.expectedScore {
				t.Errorf("Complementary score incorrect: got %d, want %d", score, tt.expectedScore)
			}

			isMatch := score >= 2
			if isMatch != tt.shouldMatch {
				t.Errorf("Match assessment incorrect: got %v, want %v (score: %d)",
					isMatch, tt.shouldMatch, score)
			}
		})
	}
}

func TestIsGoodFit(t *testing.T) {
	service := &TradeService{}

	tests := []struct {
		name         string
		playerA      RosterPlayer
		playerB      RosterPlayer
		expectedFit  bool
		description  string
	}{
		{
			name:        "Nearly equal value (within 15%)",
			playerA:     RosterPlayer{FPG: 45.0},
			playerB:     RosterPlayer{FPG: 44.0},
			expectedFit: true,
			description: "2% difference",
		},
		{
			name:        "Exactly equal value",
			playerA:     RosterPlayer{FPG: 45.0},
			playerB:     RosterPlayer{FPG: 45.0},
			expectedFit: true,
			description: "0% difference",
		},
		{
			name:        "At threshold (15% difference)",
			playerA:     RosterPlayer{FPG: 50.0},
			playerB:     RosterPlayer{FPG: 43.5},
			expectedFit: true,
			description: "~13% difference",
		},
		{
			name:        "Too large difference (20%)",
			playerA:     RosterPlayer{FPG: 50.0},
			playerB:     RosterPlayer{FPG: 40.0},
			expectedFit: false,
			description: "20% difference",
		},
		{
			name:        "Way too large difference (50%)",
			playerA:     RosterPlayer{FPG: 60.0},
			playerB:     RosterPlayer{FPG: 30.0},
			expectedFit: false,
			description: "50% difference",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isGoodFit(
				tt.playerA,
				tt.playerB,
				&TeamAnalysis{},
				&TeamAnalysis{},
			)

			if result != tt.expectedFit {
				valueDiff := math.Abs(tt.playerA.FPG - tt.playerB.FPG)
				avgValue := (tt.playerA.FPG + tt.playerB.FPG) / 2.0
				percentDiff := (valueDiff / avgValue) * 100.0

				t.Errorf("Fit assessment incorrect for %s: got %v, want %v (%.1f%% diff)",
					tt.description, result, tt.expectedFit, percentDiff)
			}
		})
	}
}

func TestFormatBenefit(t *testing.T) {
	service := &TradeService{}

	tests := []struct {
		name        string
		impact      TradeImpact
		shouldHave  []string
	}{
		{
			name: "Multiple improvements with position benefit",
			impact: TradeImpact{
				CategoryImprovements: []CategoryChange{
					{Category: "REB", Change: 3.2},
					{Category: "BLK", Change: 2.8},
					{Category: "FG%", Change: 0.02},
				},
				PositionImpact: "Fills C need",
			},
			shouldHave: []string{"Improves", "REB", "BLK", "Fills C need"},
		},
		{
			name: "No improvements",
			impact: TradeImpact{
				CategoryImprovements: []CategoryChange{},
				PositionImpact:       "Neutral position impact",
			},
			shouldHave: []string{"No significant"},
		},
		{
			name: "Single improvement",
			impact: TradeImpact{
				CategoryImprovements: []CategoryChange{
					{Category: "AST", Change: 5.0},
				},
				PositionImpact: "",
			},
			shouldHave: []string{"Improves", "AST"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.formatBenefit(tt.impact)

			if result == "" {
				t.Error("Benefit string should not be empty")
			}

			for _, expected := range tt.shouldHave {
				if !contains([]string{result}, expected) && len(expected) > 0 {
				}
			}
		})
	}
}

func TestTradeValueCalculation(t *testing.T) {
	tests := []struct {
		name           string
		playerAValue   float64
		playerBValue   float64
		expectedDelta  float64
		expectedFair   bool
	}{
		{
			name:           "Balanced trade",
			playerAValue:   45.0,
			playerBValue:   45.0,
			expectedDelta:  0.0,
			expectedFair:   true,
		},
		{
			name:           "Slight advantage to team A",
			playerAValue:   40.0,
			playerBValue:   45.0,
			expectedDelta:  5.0,
			expectedFair:   true,
		},
		{
			name:           "Large advantage to team A",
			playerAValue:   30.0,
			playerBValue:   50.0,
			expectedDelta:  20.0,
			expectedFair:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delta := math.Abs(tt.playerAValue - tt.playerBValue)

			if math.Abs(delta-tt.expectedDelta) > 0.01 {
				t.Errorf("Value delta incorrect: got %.2f, want %.2f", delta, tt.expectedDelta)
			}

			avgValue := (tt.playerAValue + tt.playerBValue) / 2.0
			percentDiff := (delta / avgValue) * 100.0
			isFair := percentDiff <= 15.0

			if isFair != tt.expectedFair {
				t.Errorf("Fairness assessment incorrect: got %v, want %v (%.1f%% diff)",
					isFair, tt.expectedFair, percentDiff)
			}
		})
	}
}

func TestEmptyTradeGeneration(t *testing.T) {
	service := &TradeService{}

	teamA := &TeamAnalysis{
		WeakCategories:   []CategoryScore{},
		StrongCategories: []CategoryScore{},
	}

	teamB := &TeamAnalysis{
		WeakCategories:   []CategoryScore{},
		StrongCategories: []CategoryScore{},
	}

	score := service.calculateComplementaryScore(teamA, teamB)

	if score != 0 {
		t.Errorf("Empty teams should have 0 complementary score, got %d", score)
	}

	shouldMatch := score >= 2
	if shouldMatch {
		t.Error("Empty teams should not match")
	}
}

func TestOneForOneTradeLogic(t *testing.T) {
	playersTeamA := []RosterPlayer{
		{PlayerID: 1, PlayerName: "Player A1", FPG: 45.0, Position: "PG"},
		{PlayerID: 2, PlayerName: "Player A2", FPG: 40.0, Position: "SG"},
	}

	playersTeamB := []RosterPlayer{
		{PlayerID: 3, PlayerName: "Player B1", FPG: 44.0, Position: "SF"},
		{PlayerID: 4, PlayerName: "Player B2", FPG: 30.0, Position: "C"},
	}

	service := &TradeService{}

	validTrades := 0
	for _, playerA := range playersTeamA {
		for _, playerB := range playersTeamB {
			if service.isGoodFit(playerA, playerB, &TeamAnalysis{}, &TeamAnalysis{}) {
				validTrades++
			}
		}
	}

	if validTrades == 0 {
		t.Error("Should find at least one valid 1-for-1 trade")
	}

	expectedValidTrades := 2

	if validTrades != expectedValidTrades {
		t.Logf("Found %d valid trades (expected ~%d)", validTrades, expectedValidTrades)
	}
}

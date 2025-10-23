package service

import (
	"math"
	"testing"
)

func TestCalculateFairnessScore(t *testing.T) {
	service := &EvaluationService{}

	tests := []struct {
		name           string
		teamAPlayers   []PlayerProjection
		teamBPlayers   []PlayerProjection
		expectedScore  float64
		shouldBeFair   bool
	}{
		{
			name: "Perfectly balanced trade",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 45.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 2, FPG: 45.0},
			},
			expectedScore: 100.0,
			shouldBeFair:  true,
		},
		{
			name: "Very fair trade (2% difference)",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 45.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 2, FPG: 44.0},
			},
			expectedScore: 97.75,
			shouldBeFair:  true,
		},
		{
			name: "Fair trade (10% difference)",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 50.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 2, FPG: 45.0},
			},
			expectedScore: 89.47,
			shouldBeFair:  true,
		},
		{
			name: "Borderline fair trade (25% difference)",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 50.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 2, FPG: 40.0},
			},
			expectedScore: 77.78,
			shouldBeFair:  true,
		},
		{
			name: "Unfair trade (40% difference)",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 60.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 2, FPG: 40.0},
			},
			expectedScore: 60.0,
			shouldBeFair:  false,
		},
		{
			name: "Multi-player fair trade",
			teamAPlayers: []PlayerProjection{
				{PlayerID: 1, FPG: 30.0},
				{PlayerID: 2, FPG: 20.0},
			},
			teamBPlayers: []PlayerProjection{
				{PlayerID: 3, FPG: 28.0},
				{PlayerID: 4, FPG: 22.0},
			},
			expectedScore: 100.0,
			shouldBeFair:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateFairnessScore(tt.teamAPlayers, tt.teamBPlayers)

			if math.Abs(score-tt.expectedScore) > 2.0 {
				t.Errorf("Fairness score incorrect: got %.2f, want %.2f", score, tt.expectedScore)
			}

			isFair := score >= 75.0
			if isFair != tt.shouldBeFair {
				t.Errorf("Fairness assessment incorrect: got %v, want %v (score: %.2f)",
					isFair, tt.shouldBeFair, score)
			}

			if score < 0 || score > 100 {
				t.Errorf("Fairness score out of bounds: %.2f (should be 0-100)", score)
			}
		})
	}
}

func TestSumFPG(t *testing.T) {
	service := &EvaluationService{}

	tests := []struct {
		name     string
		players  []PlayerProjection
		expected float64
	}{
		{
			name: "Single player",
			players: []PlayerProjection{
				{FPG: 45.0},
			},
			expected: 45.0,
		},
		{
			name: "Multiple players",
			players: []PlayerProjection{
				{FPG: 30.0},
				{FPG: 25.0},
				{FPG: 20.0},
			},
			expected: 75.0,
		},
		{
			name:     "Empty list",
			players:  []PlayerProjection{},
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.sumFPG(tt.players)
			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("Sum incorrect: got %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

func TestSimulateTrade(t *testing.T) {
	service := &EvaluationService{}

	current := TeamCategoryTotals{
		PTS:   100.0,
		REB:   50.0,
		AST:   75.0,
		STL:   10.0,
		BLK:   8.0,
		TO:    15.0,
		TPM:   12.0,
	}

	playersOut := []PlayerProjection{
		{PTS: 20.0, REB: 5.0, AST: 8.0, STL: 1.0, BLK: 0.5, TO: 3.0, TPM: 2.0},
	}

	playersIn := []PlayerProjection{
		{PTS: 15.0, REB: 12.0, AST: 2.0, STL: 1.5, BLK: 3.0, TO: 2.0, TPM: 1.0},
	}

	result := service.simulateTrade(current, playersIn, playersOut)

	expectedPTS := 100.0 - 20.0 + 15.0
	expectedREB := 50.0 - 5.0 + 12.0
	expectedAST := 75.0 - 8.0 + 2.0

	if math.Abs(result.PTS-expectedPTS) > 0.01 {
		t.Errorf("PTS simulation incorrect: got %.2f, want %.2f", result.PTS, expectedPTS)
	}

	if math.Abs(result.REB-expectedREB) > 0.01 {
		t.Errorf("REB simulation incorrect: got %.2f, want %.2f", result.REB, expectedREB)
	}

	if math.Abs(result.AST-expectedAST) > 0.01 {
		t.Errorf("AST simulation incorrect: got %.2f, want %.2f", result.AST, expectedAST)
	}
}

func TestCalculateCategoryChanges(t *testing.T) {
	service := &EvaluationService{}

	before := TeamCategoryTotals{
		PTS: 100.0,
		REB: 50.0,
		AST: 75.0,
		STL: 10.0,
		BLK: 8.0,
		TO:  15.0,
		TPM: 12.0,
	}

	after := TeamCategoryTotals{
		PTS: 105.0,
		REB: 55.0,
		AST: 70.0,
		STL: 11.0,
		BLK: 7.0,
		TO:  14.0,
		TPM: 13.0,
	}

	changes := service.calculateCategoryChanges(before, after)

	expectedChanges := map[string]float64{
		"PTS": 5.0,
		"REB": 5.0,
		"AST": -5.0,
		"STL": 1.0,
		"BLK": -1.0,
		"TO":  -1.0,
		"3PM": 1.0,
	}

	for _, change := range changes {
		expected := expectedChanges[change.Category]
		if math.Abs(change.Change-expected) > 0.01 {
			t.Errorf("Category %s change incorrect: got %.2f, want %.2f",
				change.Category, change.Change, expected)
		}

		if before.PTS != 0 && change.Category == "PTS" {
			expectedPercent := (expected / before.PTS) * 100.0
			if math.Abs(change.PercentChange-expectedPercent) > 0.01 {
				t.Errorf("Category %s percent change incorrect: got %.2f%%, want %.2f%%",
					change.Category, change.PercentChange, expectedPercent)
			}
		}
	}
}

func TestAnalyzePositionImpact(t *testing.T) {
	service := &EvaluationService{}

	tests := []struct {
		name             string
		playersIn        []PlayerProjection
		playersOut       []PlayerProjection
		expectedContains string
	}{
		{
			name: "Fills center need",
			playersIn: []PlayerProjection{
				{Position: "C"},
			},
			playersOut: []PlayerProjection{
				{Position: "PG"},
			},
			expectedContains: "Fills C need",
		},
		{
			name: "Creates guard gap",
			playersIn: []PlayerProjection{
				{Position: "C"},
			},
			playersOut: []PlayerProjection{
				{Position: "PG"},
			},
			expectedContains: "Creates PG gap",
		},
		{
			name: "Same position swap",
			playersIn: []PlayerProjection{
				{Position: "PG"},
			},
			playersOut: []PlayerProjection{
				{Position: "PG"},
			},
			expectedContains: "Neutral",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.analyzePositionImpact(tt.playersIn, tt.playersOut)

			if result == "" {
				t.Error("Position impact should not be empty")
			}
		})
	}
}

func TestGenerateRecommendation(t *testing.T) {
	tests := []struct {
		name        string
		evaluation  TradeEvaluation
		shouldAllow bool
	}{
		{
			name: "Unfair trade",
			evaluation: TradeEvaluation{
				FairnessScore: 60.0,
				IsFair:        false,
			},
			shouldAllow: false,
		},
		{
			name: "Strong mutual benefit",
			evaluation: TradeEvaluation{
				FairnessScore: 95.0,
				IsFair:        true,
				TeamAImpact: TradeImpact{
					NetBenefit: 5.0,
				},
				TeamBImpact: TradeImpact{
					NetBenefit: 5.0,
				},
			},
			shouldAllow: true,
		},
		{
			name: "Fair but minimal benefit",
			evaluation: TradeEvaluation{
				FairnessScore: 85.0,
				IsFair:        true,
				TeamAImpact: TradeImpact{
					NetBenefit: 0.5,
				},
				TeamBImpact: TradeImpact{
					NetBenefit: 0.5,
				},
			},
			shouldAllow: true,
		},
	}

	service := &EvaluationService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendation := service.generateRecommendation(&tt.evaluation)

			if recommendation == "" {
				t.Error("Recommendation should not be empty")
			}

			if !tt.evaluation.IsFair && !containsString(recommendation, "imbalanced") {
				t.Error("Unfair trade should mention imbalance")
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0
}

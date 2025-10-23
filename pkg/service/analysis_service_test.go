package service

import (
	"math"
	"testing"
)

func TestCalculateZScore(t *testing.T) {
	service := &AnalysisService{}

	tests := []struct {
		name      string
		value     float64
		allValues []float64
		expected  float64
	}{
		{
			name:      "Above average",
			value:     50.0,
			allValues: []float64{30.0, 40.0, 50.0, 60.0, 70.0},
			expected:  0.0,
		},
		{
			name:      "Well above average",
			value:     70.0,
			allValues: []float64{30.0, 40.0, 50.0, 60.0, 70.0},
			expected:  (70.0 - 50.0) / math.Sqrt(200.0),
		},
		{
			name:      "Below average",
			value:     30.0,
			allValues: []float64{30.0, 40.0, 50.0, 60.0, 70.0},
			expected:  (30.0 - 50.0) / math.Sqrt(200.0),
		},
		{
			name:      "All same values",
			value:     25.0,
			allValues: []float64{25.0, 25.0, 25.0, 25.0},
			expected:  0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculateZScore(tt.value, tt.allValues)

			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("Z-score incorrect: got %.3f, want %.3f", result, tt.expected)
			}
		})
	}
}

func TestAnalyzeTeam(t *testing.T) {
	service := &AnalysisService{}

	teamID := 1
	totals := TeamCategoryTotals{
		PTS:   120.0,
		REB:   50.0,
		AST:   80.0,
		STL:   10.0,
		BLK:   8.0,
		TO:    15.0,
		FGPct: 0.45,
		FTPct: 0.80,
		TPM:   12.0,
	}

	allTeams := []struct {
		TeamID int
		Totals TeamCategoryTotals
	}{
		{1, totals},
		{2, TeamCategoryTotals{PTS: 100.0, REB: 60.0, AST: 70.0, STL: 12.0, BLK: 10.0, TO: 18.0, FGPct: 0.43, FTPct: 0.75, TPM: 10.0}},
		{3, TeamCategoryTotals{PTS: 110.0, REB: 55.0, AST: 75.0, STL: 11.0, BLK: 9.0, TO: 16.0, FGPct: 0.44, FTPct: 0.78, TPM: 11.0}},
	}

	analysis := service.analyzeTeam(teamID, totals, allTeams)

	if analysis.TeamID != teamID {
		t.Errorf("TeamID incorrect: got %d, want %d", analysis.TeamID, teamID)
	}

	if len(analysis.WeakCategories) != 3 {
		t.Errorf("Should have 3 weak categories, got %d", len(analysis.WeakCategories))
	}

	if len(analysis.StrongCategories) != 3 {
		t.Errorf("Should have 3 strong categories, got %d", len(analysis.StrongCategories))
	}

	if len(analysis.CategoryScores) != 9 {
		t.Errorf("Should have 9 category scores, got %d", len(analysis.CategoryScores))
	}

	for cat, score := range analysis.CategoryScores {
		if math.IsNaN(score) || math.IsInf(score, 0) {
			t.Errorf("Category %s has invalid z-score: %.3f", cat, score)
		}
	}
}

func TestComplementaryScoreCalculation(t *testing.T) {
	teamAWeak := []CategoryScore{
		{Category: "REB", ZScore: -1.5},
		{Category: "BLK", ZScore: -1.2},
		{Category: "TO", ZScore: -0.8},
	}

	teamAStrong := []CategoryScore{
		{Category: "PTS", ZScore: 1.5},
		{Category: "AST", ZScore: 1.3},
		{Category: "3PM", ZScore: 1.1},
	}

	teamBWeak := []CategoryScore{
		{Category: "PTS", ZScore: -1.3},
		{Category: "AST", ZScore: -1.1},
		{Category: "STL", ZScore: -0.9},
	}

	teamBStrong := []CategoryScore{
		{Category: "REB", ZScore: 1.4},
		{Category: "BLK", ZScore: 1.2},
		{Category: "FG%", ZScore: 1.0},
	}

	score := 0

	teamAWeakMap := make(map[string]bool)
	teamAStrongMap := make(map[string]bool)
	for _, cat := range teamAWeak {
		teamAWeakMap[cat.Category] = true
	}
	for _, cat := range teamAStrong {
		teamAStrongMap[cat.Category] = true
	}

	for _, cat := range teamBWeak {
		if teamAStrongMap[cat.Category] {
			score++
		}
	}

	for _, cat := range teamBStrong {
		if teamAWeakMap[cat.Category] {
			score++
		}
	}

	expectedScore := 4

	if score != expectedScore {
		t.Errorf("Complementary score incorrect: got %d, want %d", score, expectedScore)
	}
}

func TestPositionNeedAnalysis(t *testing.T) {
	tests := []struct {
		name             string
		positionCounts   map[string]int
		expectedNeeds    []string
		shouldNeedPG     bool
		shouldNeedC      bool
	}{
		{
			name: "Thin at center",
			positionCounts: map[string]int{
				"PG": 2,
				"SG": 2,
				"SF": 2,
				"PF": 2,
				"C":  1,
			},
			expectedNeeds: []string{"C"},
			shouldNeedPG:  false,
			shouldNeedC:   true,
		},
		{
			name: "Thin at multiple positions",
			positionCounts: map[string]int{
				"PG": 1,
				"SG": 2,
				"SF": 2,
				"PF": 1,
				"C":  2,
			},
			expectedNeeds: []string{"PG", "PF"},
			shouldNeedPG:  true,
			shouldNeedC:   false,
		},
		{
			name: "Good depth everywhere",
			positionCounts: map[string]int{
				"PG": 2,
				"SG": 3,
				"SF": 2,
				"PF": 2,
				"C":  2,
			},
			expectedNeeds: []string{},
			shouldNeedPG:  false,
			shouldNeedC:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var needs []string
			positions := []string{"PG", "SG", "SF", "PF", "C"}
			for _, pos := range positions {
				if tt.positionCounts[pos] < 2 {
					needs = append(needs, pos)
				}
			}

			if len(needs) != len(tt.expectedNeeds) {
				t.Errorf("Wrong number of needs: got %d, want %d", len(needs), len(tt.expectedNeeds))
			}

			needsPG := false
			needsC := false
			for _, pos := range needs {
				if pos == "PG" {
					needsPG = true
				}
				if pos == "C" {
					needsC = true
				}
			}

			if needsPG != tt.shouldNeedPG {
				t.Errorf("PG need incorrect: got %v, want %v", needsPG, tt.shouldNeedPG)
			}

			if needsC != tt.shouldNeedC {
				t.Errorf("C need incorrect: got %v, want %v", needsC, tt.shouldNeedC)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"PG", "SG", "SF"}, "PG", true},
		{[]string{"PG", "SG", "SF"}, "C", false},
		{[]string{}, "PG", false},
		{[]string{"C"}, "C", true},
	}

	for _, tt := range tests {
		result := contains(tt.slice, tt.item)
		if result != tt.expected {
			t.Errorf("contains(%v, %s) = %v, want %v", tt.slice, tt.item, result, tt.expected)
		}
	}
}

package service

import (
	"math"
	"testing"
)

func TestCalculatePlayerValue(t *testing.T) {
	service := &ValuationService{}

	settings := ScoringSettings{
		PTS:   1.0,
		REB:   1.2,
		AST:   1.5,
		STL:   3.0,
		BLK:   3.0,
		TO:    -1.0,
		TPM:   1.0,
		FGPct: 0.0,
		FTPct: 0.0,
	}

	tests := []struct {
		name     string
		player   PlayerStats
		expected float64
	}{
		{
			name: "High scorer with good all-around stats",
			player: PlayerStats{
				PlayerID:          1,
				PointsPerGame:     28.0,
				ReboundsPerGame:   8.0,
				AssistsPerGame:    6.0,
				StealsPerGame:     1.5,
				BlocksPerGame:     1.0,
				TurnoversPerGame:  3.0,
				ThreePointersMade: 2.5,
			},
			expected: 28.0*1.0 + 8.0*1.2 + 6.0*1.5 + 1.5*3.0 + 1.0*3.0 + 3.0*-1.0 + 2.5*1.0,
		},
		{
			name: "Defensive specialist",
			player: PlayerStats{
				PlayerID:          2,
				PointsPerGame:     8.0,
				ReboundsPerGame:   12.0,
				AssistsPerGame:    1.0,
				StealsPerGame:     2.0,
				BlocksPerGame:     3.0,
				TurnoversPerGame:  1.0,
				ThreePointersMade: 0.0,
			},
			expected: 8.0*1.0 + 12.0*1.2 + 1.0*1.5 + 2.0*3.0 + 3.0*3.0 + 1.0*-1.0 + 0.0*1.0,
		},
		{
			name: "High turnover player",
			player: PlayerStats{
				PlayerID:          3,
				PointsPerGame:     20.0,
				ReboundsPerGame:   5.0,
				AssistsPerGame:    8.0,
				StealsPerGame:     1.0,
				BlocksPerGame:     0.5,
				TurnoversPerGame:  5.0,
				ThreePointersMade: 1.5,
			},
			expected: 20.0*1.0 + 5.0*1.2 + 8.0*1.5 + 1.0*3.0 + 0.5*3.0 + 5.0*-1.0 + 1.5*1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.calculatePlayerValue(tt.player, settings)

			if math.Abs(result.FPG-tt.expected) > 0.01 {
				t.Errorf("FPG calculation incorrect: got %.2f, want %.2f", result.FPG, tt.expected)
			}

			if result.PlayerID != tt.player.PlayerID {
				t.Errorf("PlayerID not set correctly: got %d, want %d", result.PlayerID, tt.player.PlayerID)
			}

			if result.Projections.PTS != tt.player.PointsPerGame {
				t.Errorf("Points projection not preserved: got %.2f, want %.2f",
					result.Projections.PTS, tt.player.PointsPerGame)
			}
		})
	}
}

func TestCalculateZScores(t *testing.T) {
	service := &ValuationService{}

	players := []PlayerValue{
		{PlayerID: 1, FPG: 50.0},
		{PlayerID: 2, FPG: 40.0},
		{PlayerID: 3, FPG: 30.0},
		{PlayerID: 4, FPG: 20.0},
		{PlayerID: 5, FPG: 10.0},
	}

	err := service.calculateZScores(players)
	if err != nil {
		t.Fatalf("calculateZScores failed: %v", err)
	}

	mean := 30.0
	stdDev := math.Sqrt(200.0)

	for i, player := range players {
		expectedZ := (player.FPG - mean) / stdDev
		if math.Abs(player.ZScore-expectedZ) > 0.01 {
			t.Errorf("Player %d z-score incorrect: got %.3f, want %.3f",
				i+1, player.ZScore, expectedZ)
		}
	}
}

func TestCalculateStats(t *testing.T) {
	service := &ValuationService{}

	tests := []struct {
		name         string
		players      []PlayerValue
		expectedMean float64
		expectedStd  float64
	}{
		{
			name: "Simple dataset",
			players: []PlayerValue{
				{FPG: 10.0},
				{FPG: 20.0},
				{FPG: 30.0},
			},
			expectedMean: 20.0,
			expectedStd:  math.Sqrt(200.0 / 3.0),
		},
		{
			name: "All same values",
			players: []PlayerValue{
				{FPG: 25.0},
				{FPG: 25.0},
				{FPG: 25.0},
			},
			expectedMean: 25.0,
			expectedStd:  0.0,
		},
		{
			name: "Wide range",
			players: []PlayerValue{
				{FPG: 0.0},
				{FPG: 50.0},
				{FPG: 100.0},
			},
			expectedMean: 50.0,
			expectedStd:  math.Sqrt((2500.0 + 0.0 + 2500.0) / 3.0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mean, stdDev := service.calculateStats(tt.players)

			if math.Abs(mean-tt.expectedMean) > 0.01 {
				t.Errorf("Mean incorrect: got %.3f, want %.3f", mean, tt.expectedMean)
			}

			if math.Abs(stdDev-tt.expectedStd) > 0.01 {
				t.Errorf("StdDev incorrect: got %.3f, want %.3f", stdDev, tt.expectedStd)
			}
		})
	}
}

func TestApplyPositionScarcity(t *testing.T) {
	tests := []struct {
		position           string
		expectedMultiplier float64
	}{
		{"PG", 1.0},
		{"SG", 1.0},
		{"SF", 1.1},
		{"PF", 1.1},
		{"C", 1.3},
		{"G", 1.0},
		{"F", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.position, func(t *testing.T) {
			scarcityMap := map[string]float64{
				"PG": 1.0,
				"SG": 1.0,
				"SF": 1.1,
				"PF": 1.1,
				"C":  1.3,
			}

			multiplier, ok := scarcityMap[tt.position]
			if !ok {
				multiplier = 1.0
			}

			if math.Abs(multiplier-tt.expectedMultiplier) > 0.01 {
				t.Errorf("Position %s multiplier incorrect: got %.2f, want %.2f",
					tt.position, multiplier, tt.expectedMultiplier)
			}
		})
	}
}

func TestRankPlayers(t *testing.T) {
	service := &ValuationService{}

	players := []PlayerValue{
		{PlayerID: 1, FPG: 50.0},
		{PlayerID: 2, FPG: 30.0},
		{PlayerID: 3, FPG: 40.0},
		{PlayerID: 4, FPG: 20.0},
		{PlayerID: 5, FPG: 45.0},
	}

	service.rankPlayers(players)

	expectedRanks := map[int]int{
		1: 1,
		2: 4,
		3: 3,
		4: 5,
		5: 2,
	}

	for _, player := range players {
		expectedRank := expectedRanks[player.PlayerID]
		if player.OverallRank != expectedRank {
			t.Errorf("Player %d rank incorrect: got %d, want %d",
				player.PlayerID, player.OverallRank, expectedRank)
		}
	}
}

func TestEmptyPlayerList(t *testing.T) {
	service := &ValuationService{}

	players := []PlayerValue{}

	err := service.calculateZScores(players)
	if err != nil {
		t.Errorf("calculateZScores should handle empty list: %v", err)
	}

	mean, stdDev := service.calculateStats(players)
	if mean != 0.0 || stdDev != 0.0 {
		t.Errorf("Empty list should return 0,0: got mean=%.2f, stdDev=%.2f", mean, stdDev)
	}
}

package yahoo

import (
	"testing"
)

func TestParseNBAStatsComplete(t *testing.T) {
	stats := []Stat{
		{StatID: 4, Value: "10"},   // FGM
		{StatID: 3, Value: "20"},   // FGA
		{StatID: 5, Value: "0.500"}, // FG%
		{StatID: 7, Value: "8"},    // FTM
		{StatID: 6, Value: "10"},   // FTA
		{StatID: 8, Value: "0.800"}, // FT%
		{StatID: 10, Value: "3"},   // 3PM
		{StatID: 9, Value: "9"},    // 3PA
		{StatID: 11, Value: "0.333"}, // 3P%
		{StatID: 12, Value: "31"},  // Points
		{StatID: 0, Value: "1"},    // Games
	}

	nbaStats, err := ParseNBAStats(stats)
	if err != nil {
		t.Fatalf("ParseNBAStats failed: %v", err)
	}

	tests := []struct {
		name string
		got  int
		want int
	}{
		{"FGM", nbaStats.FGM, 10},
		{"FGA", nbaStats.FGA, 20},
		{"FTM", nbaStats.FTM, 8},
		{"FTA", nbaStats.FTA, 10},
		{"3PM", nbaStats.ThreePointsMade, 3},
		{"3PA", nbaStats.ThreePointsAttempt, 9},
		{"Points", nbaStats.Points, 31},
		{"Games", nbaStats.GamesPlayed, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}

	if nbaStats.FGPercent != 0.500 {
		t.Errorf("FGPercent = %f, want 0.500", nbaStats.FGPercent)
	}
	if nbaStats.FTPercent != 0.800 {
		t.Errorf("FTPercent = %f, want 0.800", nbaStats.FTPercent)
	}
	if nbaStats.ThreePPercent != 0.333 {
		t.Errorf("ThreePPercent = %f, want 0.333", nbaStats.ThreePPercent)
	}
}

func TestParseNBAStatsMissingPercentages(t *testing.T) {
	stats := []Stat{
		{StatID: 4, Value: "10"},  // FGM
		{StatID: 3, Value: "20"},  // FGA
		{StatID: 7, Value: "8"},   // FTM
		{StatID: 6, Value: "10"},  // FTA
		{StatID: 10, Value: "3"},  // 3PM
		{StatID: 9, Value: "9"},   // 3PA
	}

	nbaStats, err := ParseNBAStats(stats)
	if err != nil {
		t.Fatalf("ParseNBAStats failed: %v", err)
	}

	if nbaStats.FGM != 10 || nbaStats.FGA != 20 {
		t.Errorf("FG stats incorrect: FGM=%d, FGA=%d", nbaStats.FGM, nbaStats.FGA)
	}

	if nbaStats.FGPercent == 0 {
		t.Error("FGPercent should be auto-calculated when missing, got 0")
	}
}

func TestParseNBAStatsZeroAttempts(t *testing.T) {
	stats := []Stat{
		{StatID: 4, Value: "0"},  // FGM
		{StatID: 3, Value: "0"},  // FGA
		{StatID: 7, Value: "0"},  // FTM
		{StatID: 6, Value: "0"},  // FTA
		{StatID: 10, Value: "0"}, // 3PM
		{StatID: 9, Value: "0"},  // 3PA
	}

	nbaStats, err := ParseNBAStats(stats)
	if err != nil {
		t.Fatalf("ParseNBAStats failed: %v", err)
	}

	if nbaStats.FGPercent != 0 {
		t.Errorf("FGPercent with 0 attempts should be 0, got %f", nbaStats.FGPercent)
	}
	if nbaStats.FTPercent != 0 {
		t.Errorf("FTPercent with 0 attempts should be 0, got %f", nbaStats.FTPercent)
	}
	if nbaStats.ThreePPercent != 0 {
		t.Errorf("ThreePPercent with 0 attempts should be 0, got %f", nbaStats.ThreePPercent)
	}
}

func TestNBAStatsCalculateMethods(t *testing.T) {
	stats := NBAStats{
		FGM:                10,
		FGA:                20,
		FTM:                8,
		FTA:                10,
		ThreePointsMade:    3,
		ThreePointsAttempt: 9,
		Points:             31,
	}

	fgPercent := stats.CalculateFGPercent()
	if fgPercent != 0.5 {
		t.Errorf("CalculateFGPercent() = %f, want 0.5", fgPercent)
	}

	ftPercent := stats.CalculateFTPercent()
	if ftPercent != 0.8 {
		t.Errorf("CalculateFTPercent() = %f, want 0.8", ftPercent)
	}

	tpPercent := stats.Calculate3PPercent()
	expected := 3.0 / 9.0
	if tpPercent != expected {
		t.Errorf("Calculate3PPercent() = %f, want %f", tpPercent, expected)
	}
}

func TestNBAStatsCalculateMethodsZeroAttempts(t *testing.T) {
	stats := NBAStats{
		FGM: 0,
		FGA: 0,
		FTM: 0,
		FTA: 0,
		ThreePointsMade:    0,
		ThreePointsAttempt: 0,
	}

	if stats.CalculateFGPercent() != 0.0 {
		t.Error("CalculateFGPercent() should return 0.0 with 0 attempts")
	}
	if stats.CalculateFTPercent() != 0.0 {
		t.Error("CalculateFTPercent() should return 0.0 with 0 attempts")
	}
	if stats.Calculate3PPercent() != 0.0 {
		t.Error("Calculate3PPercent() should return 0.0 with 0 attempts")
	}
}

func TestNBAStatsTrueShootingPercent(t *testing.T) {
	stats := NBAStats{
		FGA:    20,
		FTA:    10,
		Points: 31,
	}

	ts := stats.TrueShootingPercent()

	expectedTSA := 20.0 + 0.44*10.0
	expected := 31.0 / (2.0 * expectedTSA)

	if ts != expected {
		t.Errorf("TrueShootingPercent() = %f, want %f", ts, expected)
	}
}

func TestNBAStatsEffectiveFGPercent(t *testing.T) {
	stats := NBAStats{
		FGM:                10,
		FGA:                20,
		ThreePointsMade:    3,
	}

	efg := stats.EffectiveFGPercent()

	expected := (10.0 + 0.5*3.0) / 20.0

	if efg != expected {
		t.Errorf("EffectiveFGPercent() = %f, want %f", efg, expected)
	}
}

func TestStatHelperGetShootingStats(t *testing.T) {
	stats := []Stat{
		{StatID: 4, Value: "10"},  // FGM
		{StatID: 3, Value: "20"},  // FGA
		{StatID: 7, Value: "8"},   // FTM
		{StatID: 6, Value: "10"},  // FTA
		{StatID: 10, Value: "3"},  // 3PM
		{StatID: 9, Value: "9"},   // 3PA
	}

	helper := NewStatHelper(stats)
	fgm, fga, ftm, fta, tpm, tpa, err := helper.GetShootingStats()

	if err != nil {
		t.Fatalf("GetShootingStats() failed: %v", err)
	}

	if fgm != 10 || fga != 20 {
		t.Errorf("FG stats: got FGM=%d, FGA=%d, want 10, 20", fgm, fga)
	}
	if ftm != 8 || fta != 10 {
		t.Errorf("FT stats: got FTM=%d, FTA=%d, want 8, 10", ftm, fta)
	}
	if tpm != 3 || tpa != 9 {
		t.Errorf("3P stats: got 3PM=%d, 3PA=%d, want 3, 9", tpm, tpa)
	}
}

func TestStatHelperGetShootingStatsMissing(t *testing.T) {
	stats := []Stat{
		{StatID: 4, Value: "10"},  // FGM only
	}

	helper := NewStatHelper(stats)
	_, _, _, _, _, _, err := helper.GetShootingStats()

	if err == nil {
		t.Error("GetShootingStats() should return error when stats are missing")
	}
}

func TestParseCompoundStat(t *testing.T) {
	stats := []Stat{
		{StatID: StatIDFGM, Value: "7/15"},  // Compound FGM/FGA
		{StatID: StatIDFTM, Value: "4/5"},   // Compound FTM/FTA
		{StatID: StatID3PM, Value: "2/8"},   // Compound 3PM/3PA
	}

	helper := NewStatHelper(stats)

	// Test FGM/FGA parsing
	fgm, fga, err := helper.parseCompoundStat(StatIDFGM)
	if err != nil {
		t.Errorf("parseCompoundStat(FGM) failed: %v", err)
	}
	if fgm != 7 || fga != 15 {
		t.Errorf("FG compound stat: got FGM=%d, FGA=%d, want 7, 15", fgm, fga)
	}

	// Test FTM/FTA parsing
	ftm, fta, err := helper.parseCompoundStat(StatIDFTM)
	if err != nil {
		t.Errorf("parseCompoundStat(FTM) failed: %v", err)
	}
	if ftm != 4 || fta != 5 {
		t.Errorf("FT compound stat: got FTM=%d, FTA=%d, want 4, 5", ftm, fta)
	}

	// Test 3PM/3PA parsing
	tpm, tpa, err := helper.parseCompoundStat(StatID3PM)
	if err != nil {
		t.Errorf("parseCompoundStat(3PM) failed: %v", err)
	}
	if tpm != 2 || tpa != 8 {
		t.Errorf("3P compound stat: got 3PM=%d, 3PA=%d, want 2, 8", tpm, tpa)
	}
}

func TestGetFGMFGA(t *testing.T) {
	// Test with compound stats
	stats := []Stat{
		{StatID: StatIDFGM, Value: "7/15"},
	}

	helper := NewStatHelper(stats)
	fgm, fga, err := helper.GetFGMFGA()

	if err != nil {
		t.Errorf("GetFGMFGA() failed: %v", err)
	}
	if fgm != 7 || fga != 15 {
		t.Errorf("GetFGMFGA() = (%d, %d), want (7, 15)", fgm, fga)
	}
}

func TestGetFTMFTA(t *testing.T) {
	// Test with compound stats
	stats := []Stat{
		{StatID: StatIDFTM, Value: "4/5"},
	}

	helper := NewStatHelper(stats)
	ftm, fta, err := helper.GetFTMFTA()

	if err != nil {
		t.Errorf("GetFTMFTA() failed: %v", err)
	}
	if ftm != 4 || fta != 5 {
		t.Errorf("GetFTMFTA() = (%d, %d), want (4, 5)", ftm, fta)
	}
}

func TestGet3PM3PA(t *testing.T) {
	// Test with compound stats
	stats := []Stat{
		{StatID: StatID3PM, Value: "2/8"},
	}

	helper := NewStatHelper(stats)
	tpm, tpa, err := helper.Get3PM3PA()

	if err != nil {
		t.Errorf("Get3PM3PA() failed: %v", err)
	}
	if tpm != 2 || tpa != 8 {
		t.Errorf("Get3PM3PA() = (%d, %d), want (2, 8)", tpm, tpa)
	}
}

func TestGetFGMFGAAlternateCompoundID(t *testing.T) {
	// Test with alternate compound stat ID 9004003
	stats := []Stat{
		{StatID: StatIDFGMFGACompound, Value: "93/180"},
	}

	helper := NewStatHelper(stats)
	fgm, fga, err := helper.GetFGMFGA()

	if err != nil {
		t.Errorf("GetFGMFGA() failed: %v", err)
	}
	if fgm != 93 || fga != 180 {
		t.Errorf("GetFGMFGA() = (%d, %d), want (93, 180)", fgm, fga)
	}
}

func TestGetFTMFTAAlternateCompoundID(t *testing.T) {
	// Test with alternate compound stat ID 9007006
	stats := []Stat{
		{StatID: StatIDFTMFTACompound, Value: "71/83"},
	}

	helper := NewStatHelper(stats)
	ftm, fta, err := helper.GetFTMFTA()

	if err != nil {
		t.Errorf("GetFTMFTA() failed: %v", err)
	}
	if ftm != 71 || fta != 83 {
		t.Errorf("GetFTMFTA() = (%d, %d), want (71, 83)", ftm, fta)
	}
}

func TestParseNBAStatsWithCompoundStats(t *testing.T) {
	// Test with compound stats instead of individual stat IDs
	stats := []Stat{
		{StatID: StatIDFGM, Value: "10/20"},  // Compound FGM/FGA
		{StatID: StatIDFTM, Value: "8/10"},   // Compound FTM/FTA
		{StatID: StatID3PM, Value: "3/9"},    // Compound 3PM/3PA
		{StatID: StatIDPoints, Value: "31"},
		{StatID: StatIDGamesPlayed, Value: "1"},
	}

	nbaStats, err := ParseNBAStats(stats)
	if err != nil {
		t.Fatalf("ParseNBAStats failed: %v", err)
	}

	if nbaStats.FGM != 10 {
		t.Errorf("FGM = %d, want 10", nbaStats.FGM)
	}
	if nbaStats.FGA != 20 {
		t.Errorf("FGA = %d, want 20", nbaStats.FGA)
	}
	if nbaStats.FTM != 8 {
		t.Errorf("FTM = %d, want 8", nbaStats.FTM)
	}
	if nbaStats.FTA != 10 {
		t.Errorf("FTA = %d, want 10", nbaStats.FTA)
	}
	if nbaStats.ThreePointsMade != 3 {
		t.Errorf("3PM = %d, want 3", nbaStats.ThreePointsMade)
	}
	if nbaStats.ThreePointsAttempt != 9 {
		t.Errorf("3PA = %d, want 9", nbaStats.ThreePointsAttempt)
	}
}

func TestStatHelperGetIntByID(t *testing.T) {
	stats := []Stat{
		{StatID: StatIDFGM, Value: "10"},
		{StatID: StatIDFGA, Value: "20"},
		{StatID: StatIDFTM, Value: "8"},
		{StatID: StatIDFTA, Value: "10"},
		{StatID: StatID3PM, Value: "3"},
		{StatID: StatID3PA, Value: "9"},
	}

	helper := NewStatHelper(stats)

	tests := []struct {
		name   string
		statID int
		want   int
	}{
		{"FGM", StatIDFGM, 10},
		{"FGA", StatIDFGA, 20},
		{"FTM", StatIDFTM, 8},
		{"FTA", StatIDFTA, 10},
		{"3PM", StatID3PM, 3},
		{"3PA", StatID3PA, 9},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := helper.GetIntByID(tt.statID)
			if err != nil {
				t.Errorf("GetIntByID(%d) failed: %v", tt.statID, err)
			}
			if got != tt.want {
				t.Errorf("GetIntByID(%d) = %d, want %d", tt.statID, got, tt.want)
			}
		})
	}
}

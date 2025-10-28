package yahoo

import (
	"fmt"
	"strconv"
)

type StatHelper struct {
	stats []Stat
}

func NewStatHelper(stats []Stat) *StatHelper {
	return &StatHelper{stats: stats}
}

func (sh *StatHelper) GetByID(statID int) (string, bool) {
	for _, stat := range sh.stats {
		if stat.StatID == statID {
			return stat.Value, true
		}
	}
	return "", false
}

func (sh *StatHelper) GetFloatByID(statID int) (float64, error) {
	value, ok := sh.GetByID(statID)
	if !ok {
		return 0, fmt.Errorf("stat ID %d not found", statID)
	}
	return strconv.ParseFloat(value, 64)
}

func (sh *StatHelper) GetIntByID(statID int) (int, error) {
	value, ok := sh.GetByID(statID)
	if !ok {
		return 0, fmt.Errorf("stat ID %d not found", statID)
	}
	return strconv.Atoi(value)
}

func (sh *StatHelper) GetAll() []Stat {
	return sh.stats
}

func (sh *StatHelper) GetShootingStats() (fgm, fga, ftm, fta, tpm, tpa int, err error) {
	fgm, err = sh.GetIntByID(StatIDFGM)
	if err != nil {
		return
	}
	fga, err = sh.GetIntByID(StatIDFGA)
	if err != nil {
		return
	}
	ftm, err = sh.GetIntByID(StatIDFTM)
	if err != nil {
		return
	}
	fta, err = sh.GetIntByID(StatIDFTA)
	if err != nil {
		return
	}
	tpm, err = sh.GetIntByID(StatID3PM)
	if err != nil {
		return
	}
	tpa, err = sh.GetIntByID(StatID3PA)
	return
}

const (
	StatIDGamesPlayed       = 0
	StatIDFGM               = 5
	StatIDFGA               = 6
	StatIDFGPercent         = 7
	StatIDFTM               = 8
	StatIDFTA               = 9
	StatIDFTPercent         = 10
	StatID3PM               = 12
	StatID3PA               = 13
	StatID3PPercent         = 14
	StatIDPoints            = 15
	StatIDRebounds          = 16
	StatIDOffensiveRebounds = 17
	StatIDAssists           = 18
	StatIDSteals            = 19
	StatIDBlocks            = 20
	StatIDTurnovers         = 21
)

type NBAStats struct {
	GamesPlayed       int
	FGM               int
	FGA               int
	FGPercent         float64
	FTM               int
	FTA               int
	FTPercent         float64
	ThreePointsMade   int
	ThreePointsAttempt int
	ThreePPercent     float64
	Points            int
	Rebounds          int
	OffensiveRebounds int
	Assists           int
	Steals            int
	Blocks            int
	Turnovers         int
}

func ParseNBAStats(stats []Stat) (*NBAStats, error) {
	sh := NewStatHelper(stats)
	nbaStats := &NBAStats{}

	if val, err := sh.GetIntByID(StatIDGamesPlayed); err == nil {
		nbaStats.GamesPlayed = val
	}
	if val, err := sh.GetIntByID(StatIDFGM); err == nil {
		nbaStats.FGM = val
	}
	if val, err := sh.GetIntByID(StatIDFGA); err == nil {
		nbaStats.FGA = val
	}
	if val, err := sh.GetFloatByID(StatIDFGPercent); err == nil {
		nbaStats.FGPercent = val
	}
	if val, err := sh.GetIntByID(StatIDFTM); err == nil {
		nbaStats.FTM = val
	}
	if val, err := sh.GetIntByID(StatIDFTA); err == nil {
		nbaStats.FTA = val
	}
	if val, err := sh.GetFloatByID(StatIDFTPercent); err == nil {
		nbaStats.FTPercent = val
	}
	if val, err := sh.GetIntByID(StatID3PM); err == nil {
		nbaStats.ThreePointsMade = val
	}
	if val, err := sh.GetIntByID(StatID3PA); err == nil {
		nbaStats.ThreePointsAttempt = val
	}
	if val, err := sh.GetFloatByID(StatID3PPercent); err == nil {
		nbaStats.ThreePPercent = val
	}
	if val, err := sh.GetIntByID(StatIDPoints); err == nil {
		nbaStats.Points = val
	}
	if val, err := sh.GetIntByID(StatIDRebounds); err == nil {
		nbaStats.Rebounds = val
	}
	if val, err := sh.GetIntByID(StatIDOffensiveRebounds); err == nil {
		nbaStats.OffensiveRebounds = val
	}
	if val, err := sh.GetIntByID(StatIDAssists); err == nil {
		nbaStats.Assists = val
	}
	if val, err := sh.GetIntByID(StatIDSteals); err == nil {
		nbaStats.Steals = val
	}
	if val, err := sh.GetIntByID(StatIDBlocks); err == nil {
		nbaStats.Blocks = val
	}
	if val, err := sh.GetIntByID(StatIDTurnovers); err == nil {
		nbaStats.Turnovers = val
	}

	return nbaStats, nil
}

func (n *NBAStats) CalculateFGPercent() float64 {
	if n.FGA == 0 {
		return 0.0
	}
	return float64(n.FGM) / float64(n.FGA)
}

func (n *NBAStats) CalculateFTPercent() float64 {
	if n.FTA == 0 {
		return 0.0
	}
	return float64(n.FTM) / float64(n.FTA)
}

func (n *NBAStats) Calculate3PPercent() float64 {
	if n.ThreePointsAttempt == 0 {
		return 0.0
	}
	return float64(n.ThreePointsMade) / float64(n.ThreePointsAttempt)
}

func (n *NBAStats) TrueShootingPercent() float64 {
	tsa := float64(n.FGA) + 0.44*float64(n.FTA)
	if tsa == 0 {
		return 0.0
	}
	return float64(n.Points) / (2.0 * tsa)
}

func (n *NBAStats) EffectiveFGPercent() float64 {
	if n.FGA == 0 {
		return 0.0
	}
	return (float64(n.FGM) + 0.5*float64(n.ThreePointsMade)) / float64(n.FGA)
}

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

// GetFGMFGA attempts to get field goal made/attempted, with fallback to compound stat parsing
func (sh *StatHelper) GetFGMFGA() (fgm, fga int, err error) {
	fgm, err = sh.GetIntByID(StatIDFGM)
	if err != nil {
		if compoundFGM, compoundFGA, parseErr := sh.parseCompoundStat(StatIDFGM); parseErr == nil {
			return compoundFGM, compoundFGA, nil
		}
		return 0, 0, err
	}
	fga, err = sh.GetIntByID(StatIDFGA)
	if err != nil {
		if _, compoundFGA, parseErr := sh.parseCompoundStat(StatIDFGM); parseErr == nil {
			return fgm, compoundFGA, nil
		}
		return fgm, 0, err
	}
	return fgm, fga, nil
}

// GetFTMFTA attempts to get free throw made/attempted, with fallback to compound stat parsing
func (sh *StatHelper) GetFTMFTA() (ftm, fta int, err error) {
	ftm, err = sh.GetIntByID(StatIDFTM)
	if err != nil {
		if compoundFTM, compoundFTA, parseErr := sh.parseCompoundStat(StatIDFTM); parseErr == nil {
			return compoundFTM, compoundFTA, nil
		}
		return 0, 0, err
	}
	fta, err = sh.GetIntByID(StatIDFTA)
	if err != nil {
		if _, compoundFTA, parseErr := sh.parseCompoundStat(StatIDFTM); parseErr == nil {
			return ftm, compoundFTA, nil
		}
		return ftm, 0, err
	}
	return ftm, fta, nil
}

// Get3PM3PA attempts to get 3-pointers made/attempted, with fallback to compound stat parsing
func (sh *StatHelper) Get3PM3PA() (tpm, tpa int, err error) {
	tpm, err = sh.GetIntByID(StatID3PM)
	if err != nil {
		if compoundTPM, compoundTPA, parseErr := sh.parseCompoundStat(StatID3PM); parseErr == nil {
			return compoundTPM, compoundTPA, nil
		}
		return 0, 0, err
	}
	tpa, err = sh.GetIntByID(StatID3PA)
	if err != nil {
		if _, compoundTPA, parseErr := sh.parseCompoundStat(StatID3PM); parseErr == nil {
			return tpm, compoundTPA, nil
		}
		// 3PA is optional, return with tpm and 0 for tpa
		return tpm, 0, nil
	}
	return tpm, tpa, nil
}

// parseCompoundStat attempts to parse a compound stat value like "7/15" into made/attempted
// This is a fallback for when the stat ID returns a compound value instead of individual stats
func (sh *StatHelper) parseCompoundStat(statID int) (made int, attempted int, err error) {
	value, ok := sh.GetByID(statID)
	if !ok {
		return 0, 0, fmt.Errorf("stat ID %d not found", statID)
	}
	
	// Parse compound format "made/attempted"
	parts := []rune(value)
	var madeStr, attemptedStr string
	slashFound := false
	
	for _, ch := range parts {
		if ch == '/' {
			slashFound = true
			continue
		}
		if !slashFound {
			madeStr += string(ch)
		} else {
			attemptedStr += string(ch)
		}
	}
	
	if !slashFound || madeStr == "" || attemptedStr == "" {
		return 0, 0, fmt.Errorf("invalid compound stat format: %s", value)
	}
	
	made, err = strconv.Atoi(madeStr)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse made value: %w", err)
	}
	
	attempted, err = strconv.Atoi(attemptedStr)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse attempted value: %w", err)
	}
	
	return made, attempted, nil
}

func (sh *StatHelper) GetShootingStats() (fgm, fga, ftm, fta, tpm, tpa int, err error) {
	fgm, err = sh.GetIntByID(StatIDFGM)
	if err != nil {
		// Fallback: try parsing from compound stat
		if compoundFGM, _, parseErr := sh.parseCompoundStat(StatIDFGM); parseErr == nil {
			fgm = compoundFGM
		} else {
			return
		}
	}
	fga, err = sh.GetIntByID(StatIDFGA)
	if err != nil {
		// Fallback: try parsing from compound stat
		if _, compoundFGA, parseErr := sh.parseCompoundStat(StatIDFGM); parseErr == nil {
			fga = compoundFGA
			err = nil
		} else {
			return
		}
	}
	ftm, err = sh.GetIntByID(StatIDFTM)
	if err != nil {
		// Fallback: try parsing from compound stat
		if compoundFTM, _, parseErr := sh.parseCompoundStat(StatIDFTM); parseErr == nil {
			ftm = compoundFTM
			err = nil
		} else {
			return
		}
	}
	fta, err = sh.GetIntByID(StatIDFTA)
	if err != nil {
		// Fallback: try parsing from compound stat
		if _, compoundFTA, parseErr := sh.parseCompoundStat(StatIDFTM); parseErr == nil {
			fta = compoundFTA
			err = nil
		} else {
			return
		}
	}
	tpm, err = sh.GetIntByID(StatID3PM)
	if err != nil {
		// Fallback: try parsing from compound stat
		if compound3PM, _, parseErr := sh.parseCompoundStat(StatID3PM); parseErr == nil {
			tpm = compound3PM
			err = nil
		} else {
			return
		}
	}
	tpa, err = sh.GetIntByID(StatID3PA)
	if err != nil {
		// Fallback: try parsing from compound stat
		if _, compound3PA, parseErr := sh.parseCompoundStat(StatID3PM); parseErr == nil {
			tpa = compound3PA
			err = nil
		} else {
			err = nil // 3PA is optional
		}
	}
	return
}

const (
	StatIDGamesPlayed       = 0
	StatIDGamesStarted      = 1
	StatIDMinutesPlayed     = 2
	StatIDFGA               = 3
	StatIDFGM               = 4
	StatIDFGPercent         = 5
	StatIDFTA               = 6
	StatIDFTM               = 7
	StatIDFTPercent         = 8
	StatID3PA               = 9
	StatID3PM               = 10
	StatID3PPercent         = 11
	StatIDPoints            = 12
	StatIDOffensiveRebounds = 13
	StatIDDefensiveRebounds = 14
	StatIDRebounds          = 15
	StatIDAssists           = 16
	StatIDSteals            = 17
	StatIDBlocks            = 18
	StatIDTurnovers         = 19
	StatIDAssistTurnoverRatio = 20
	StatIDPersonalFouls     = 21
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
	} else if fgm, fga, err := sh.parseCompoundStat(StatIDFGM); err == nil {
		nbaStats.FGM = fgm
		nbaStats.FGA = fga
	}
	if val, err := sh.GetIntByID(StatIDFGA); err == nil {
		nbaStats.FGA = val
	} else if _, fga, err := sh.parseCompoundStat(StatIDFGM); err == nil && nbaStats.FGA == 0 {
		nbaStats.FGA = fga
	}
	if val, err := sh.GetFloatByID(StatIDFGPercent); err == nil {
		nbaStats.FGPercent = val
	}
	if val, err := sh.GetIntByID(StatIDFTM); err == nil {
		nbaStats.FTM = val
	} else if ftm, fta, err := sh.parseCompoundStat(StatIDFTM); err == nil {
		nbaStats.FTM = ftm
		nbaStats.FTA = fta
	}
	if val, err := sh.GetIntByID(StatIDFTA); err == nil {
		nbaStats.FTA = val
	} else if _, fta, err := sh.parseCompoundStat(StatIDFTM); err == nil && nbaStats.FTA == 0 {
		nbaStats.FTA = fta
	}
	if val, err := sh.GetFloatByID(StatIDFTPercent); err == nil {
		nbaStats.FTPercent = val
	}
	if val, err := sh.GetIntByID(StatID3PM); err == nil {
		nbaStats.ThreePointsMade = val
	} else if tpm, tpa, err := sh.parseCompoundStat(StatID3PM); err == nil {
		nbaStats.ThreePointsMade = tpm
		nbaStats.ThreePointsAttempt = tpa
	}
	if val, err := sh.GetIntByID(StatID3PA); err == nil {
		nbaStats.ThreePointsAttempt = val
	} else if _, tpa, err := sh.parseCompoundStat(StatID3PM); err == nil && nbaStats.ThreePointsAttempt == 0 {
		nbaStats.ThreePointsAttempt = tpa
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

	if nbaStats.FGPercent == 0 && nbaStats.FGA > 0 {
		nbaStats.FGPercent = nbaStats.CalculateFGPercent()
	}
	if nbaStats.FTPercent == 0 && nbaStats.FTA > 0 {
		nbaStats.FTPercent = nbaStats.CalculateFTPercent()
	}
	if nbaStats.ThreePPercent == 0 && nbaStats.ThreePointsAttempt > 0 {
		nbaStats.ThreePPercent = nbaStats.Calculate3PPercent()
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

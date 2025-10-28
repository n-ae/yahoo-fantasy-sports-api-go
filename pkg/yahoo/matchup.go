package yahoo

type Week struct {
	WeekNum   int       `json:"week"`
	StartDate string    `json:"start"`
	EndDate   string    `json:"end"`
	Matchups  []Matchup `json:"matchups"`
}

type Matchup struct {
	Week              int           `json:"week"`
	WeekStart         string        `json:"week_start"`
	WeekEnd           string        `json:"week_end"`
	Status            string        `json:"status"`
	IsPlayoffs        bool          `json:"is_playoffs"`
	IsConsolation     bool          `json:"is_consolation"`
	IsTied            bool          `json:"is_tied"`
	WinnerTeamKey     string        `json:"winner_team_key,omitempty"`
	Teams             []MatchupTeam `json:"teams"`
}

type MatchupTeam struct {
	TeamKey            string        `json:"team_key"`
	TeamID             string        `json:"team_id"`
	Name               string        `json:"name"`
	Points             float64       `json:"points"`
	ProjectedPoints    float64       `json:"projected_points"`
	IsWinner           bool          `json:"is_winner"`
	Stats              []Stat        `json:"stats,omitempty"`
	TeamPoints         TeamPoints    `json:"team_points"`
	TeamProjectedPoints TeamProjectedPoints `json:"team_projected_points"`
}

type TeamPoints struct {
	CoverageType string  `json:"coverage_type"`
	Week         int     `json:"week,omitempty"`
	Total        float64 `json:"total"`
}

type TeamProjectedPoints struct {
	CoverageType string  `json:"coverage_type"`
	Week         int     `json:"week,omitempty"`
	Total        float64 `json:"total"`
}

type yahooScoreboardResponse struct {
	FantasyContent struct {
		League struct {
			Scoreboard struct {
				Week     string `json:"week"`
				Matchups []struct {
					Matchup yahooMatchupData `json:"matchup"`
				} `json:"matchups"`
			} `json:"scoreboard"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooMatchupData struct {
	Week      string `json:"week"`
	WeekStart string `json:"week_start"`
	WeekEnd   string `json:"week_end"`
	Status    string `json:"status"`
	IsPlayoffs string `json:"is_playoffs"`
	IsConsolation string `json:"is_consolation"`
	IsTied    string `json:"is_tied"`
	WinnerTeamKey string `json:"winner_team_key,omitempty"`
	Teams     struct {
		Team []struct {
			TeamKey  string `json:"team_key"`
			TeamID   string `json:"team_id"`
			Name     string `json:"name"`
			WinProbability string `json:"win_probability,omitempty"`
			TeamPoints struct {
				CoverageType string `json:"coverage_type"`
				Week         string `json:"week,omitempty"`
				Total        string `json:"total"`
			} `json:"team_points"`
			TeamProjectedPoints struct {
				CoverageType string `json:"coverage_type"`
				Week         string `json:"week,omitempty"`
				Total        string `json:"total"`
			} `json:"team_projected_points"`
			TeamStats *struct {
				CoverageType string `json:"coverage_type"`
				Week         string `json:"week,omitempty"`
				Stats        struct {
					Stat []struct {
						StatID int    `json:"stat_id"`
						Value  string `json:"value"`
					} `json:"stat"`
				} `json:"stats"`
			} `json:"team_stats,omitempty"`
		} `json:"team"`
	} `json:"teams"`
}

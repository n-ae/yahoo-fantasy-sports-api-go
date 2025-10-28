package yahoo

type Standings struct {
	Teams []StandingsTeam `json:"teams"`
}

type StandingsTeam struct {
	TeamKey        string         `json:"team_key"`
	TeamID         string         `json:"team_id"`
	Name           string         `json:"name"`
	TeamStandings  TeamStandings  `json:"team_standings"`
	ManagerNickname string        `json:"manager_nickname,omitempty"`
	Managers       []Manager      `json:"managers,omitempty"`
}

type TeamStandings struct {
	Rank            int            `json:"rank"`
	PlayoffSeed     int            `json:"playoff_seed,omitempty"`
	OutcomeTotals   OutcomeTotals  `json:"outcome_totals"`
	PointsFor       float64        `json:"points_for"`
	PointsAgainst   float64        `json:"points_against"`
	GamesBack       string         `json:"games_back,omitempty"`
	Streak          *Streak        `json:"streak,omitempty"`
}

type OutcomeTotals struct {
	Wins       int     `json:"wins"`
	Losses     int     `json:"losses"`
	Ties       int     `json:"ties"`
	Percentage float64 `json:"percentage"`
}

type Streak struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

type Manager struct {
	ManagerID        string `json:"manager_id"`
	Nickname         string `json:"nickname"`
	GUID             string `json:"guid"`
	IsCommissioner   bool   `json:"is_commissioner"`
	IsCurrentLogin   bool   `json:"is_current_login"`
	Email            string `json:"email,omitempty"`
	ImageURL         string `json:"image_url,omitempty"`
}

type yahooStandingsResponse struct {
	FantasyContent struct {
		League struct {
			Standings struct {
				Teams []struct {
					Team yahooStandingsTeamData `json:"team"`
				} `json:"teams"`
			} `json:"standings"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooStandingsTeamData struct {
	TeamKey  string `json:"team_key"`
	TeamID   string `json:"team_id"`
	Name     string `json:"name"`
	Managers []struct {
		Manager struct {
			ManagerID      string `json:"manager_id"`
			Nickname       string `json:"nickname"`
			GUID           string `json:"guid"`
			IsCommissioner string `json:"is_commissioner"`
			IsCurrentLogin string `json:"is_current_login"`
		} `json:"manager"`
	} `json:"managers"`
	TeamStandings struct {
		Rank          string `json:"rank"`
		PlayoffSeed   string `json:"playoff_seed,omitempty"`
		OutcomeTotals struct {
			Wins       string `json:"wins"`
			Losses     string `json:"losses"`
			Ties       string `json:"ties"`
			Percentage string `json:"percentage"`
		} `json:"outcome_totals"`
		PointsFor     string `json:"points_for"`
		PointsAgainst string `json:"points_against"`
		GamesBack     string `json:"games_back,omitempty"`
		Streak        *struct {
			Type  string `json:"type"`
			Value string `json:"value"`
		} `json:"streak,omitempty"`
	} `json:"team_standings"`
}

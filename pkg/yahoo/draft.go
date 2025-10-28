package yahoo

type DraftResult struct {
	Pick      int    `json:"pick"`
	Round     int    `json:"round"`
	TeamKey   string `json:"team_key"`
	TeamName  string `json:"team_name,omitempty"`
	PlayerKey string `json:"player_key"`
	Player    Player `json:"player"`
}

type yahooDraftResultsResponse struct {
	FantasyContent struct {
		League struct {
			DraftResults []struct {
				DraftResult yahooDraftResultData `json:"draft_result"`
			} `json:"draft_results"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooTeamDraftResultsResponse struct {
	FantasyContent struct {
		Team struct {
			DraftResults []struct {
				DraftResult yahooDraftResultData `json:"draft_result"`
			} `json:"draft_results"`
		} `json:"team"`
	} `json:"fantasy_content"`
}

type yahooDraftResultData struct {
	Pick    string `json:"pick"`
	Round   string `json:"round"`
	TeamKey string `json:"team_key"`
	Players struct {
		Player yahooPlayerData `json:"player"`
	} `json:"players"`
}

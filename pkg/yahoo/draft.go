package yahoo

type DraftResult struct {
	Pick      int    `json:"pick"`
	Round     int    `json:"round"`
	TeamKey   string `json:"team_key"`
	TeamName  string `json:"team_name,omitempty"`
	PlayerKey string `json:"player_key"`
	Player    Player `json:"player"`
	// DecodeWarnings lists non-fatal numeric parse failures for this draft
	// result's own fields (the nested Player carries its own warnings).
	DecodeWarnings []DecodeWarning `json:"decode_warnings,omitempty"`
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

type yahooDraftResultData struct {
	Pick    string `json:"pick"`
	Round   string `json:"round"`
	TeamKey string `json:"team_key"`
	Players struct {
		Player yahooPlayerData `json:"player"`
	} `json:"players"`
}

package yahoo

type Stat struct {
	StatID  int     `json:"stat_id"`
	Value   string  `json:"value"`
	Display string  `json:"-"`
	Name    string  `json:"-"`
	Order   int     `json:"-"`
}

type StatCategory struct {
	StatID      int    `json:"stat_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	SortOrder   int    `json:"sort_order"`
	PositionType string `json:"position_type"`
}

type Player struct {
	PlayerKey             string                 `json:"player_key"`
	PlayerID              string                 `json:"player_id"`
	Name                  PlayerName             `json:"name"`
	EditorialTeamKey      string                 `json:"editorial_team_key"`
	EditorialTeamFullName string                 `json:"editorial_team_full_name"`
	EditorialTeamAbbr     string                 `json:"editorial_team_abbr"`
	DisplayPosition       string                 `json:"display_position"`
	EligiblePositions     []string               `json:"eligible_positions"`
	SelectedPosition      SelectedPosition       `json:"selected_position"`
	PlayerStats           *PlayerStats           `json:"player_stats,omitempty"`
	PlayerPoints          *PlayerPoints          `json:"player_points,omitempty"`
	Ownership             *Ownership             `json:"ownership,omitempty"`
	PercentOwned          *PercentOwned          `json:"percent_owned,omitempty"`
	Status                string                 `json:"status,omitempty"`
	StatusFull            string                 `json:"status_full,omitempty"`
	InjuryNote            string                 `json:"injury_note,omitempty"`
	UniformNumber         string                 `json:"uniform_number,omitempty"`
	ImageURL              string                 `json:"image_url,omitempty"`
	Headshot              map[string]string      `json:"headshot,omitempty"`
	ByeWeeks              map[string]int         `json:"bye_weeks,omitempty"`
}

type PlayerName struct {
	Full       string `json:"full"`
	First      string `json:"first"`
	Last       string `json:"last"`
	ASCIIFirst string `json:"ascii_first"`
	ASCIILast  string `json:"ascii_last"`
}

type SelectedPosition struct {
	Position      string `json:"position"`
	CoverageType  string `json:"coverage_type,omitempty"`
	Date          string `json:"date,omitempty"`
	Week          int    `json:"week,omitempty"`
	IsFlexPosition bool   `json:"is_flex_position,omitempty"`
}

type PlayerStats struct {
	CoverageType string `json:"coverage_type"`
	Week         int    `json:"week,omitempty"`
	Date         string `json:"date,omitempty"`
	Season       int    `json:"season,omitempty"`
	Stats        []Stat `json:"stats"`
}

type PlayerPoints struct {
	CoverageType string  `json:"coverage_type"`
	Week         int     `json:"week,omitempty"`
	Season       int     `json:"season,omitempty"`
	Total        float64 `json:"total"`
}

type Ownership struct {
	OwnershipType string `json:"ownership_type"`
	OwnerTeamKey  string `json:"owner_team_key,omitempty"`
	OwnerTeamName string `json:"owner_team_name,omitempty"`
}

type PercentOwned struct {
	CoverageType string  `json:"coverage_type"`
	Week         int     `json:"week,omitempty"`
	Value        float64 `json:"value"`
	Delta        float64 `json:"delta,omitempty"`
}

type PlayerStatus string

const (
	PlayerStatusAll           PlayerStatus = "A"
	PlayerStatusFreeAgents    PlayerStatus = "FA"
	PlayerStatusWaivers       PlayerStatus = "W"
	PlayerStatusTaken         PlayerStatus = "T"
	PlayerStatusKeepers       PlayerStatus = "K"
)

type yahooPlayerResponse struct {
	FantasyContent struct {
		League struct {
			Players []struct {
				Player yahooPlayerData `json:"player"`
			} `json:"players"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooSinglePlayerResponse struct {
	FantasyContent struct {
		League struct {
			Players struct {
				Player yahooPlayerData `json:"player"`
			} `json:"players"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooPlayerData struct {
	PlayerKey        string `json:"player_key"`
	PlayerID         string `json:"player_id"`
	Name             struct {
		Full       string `json:"full"`
		First      string `json:"first"`
		Last       string `json:"last"`
		ASCIIFirst string `json:"ascii_first"`
		ASCIILast  string `json:"ascii_last"`
	} `json:"name"`
	EditorialTeamKey      string `json:"editorial_team_key"`
	EditorialTeamFullName string `json:"editorial_team_full_name"`
	EditorialTeamAbbr     string `json:"editorial_team_abbr"`
	DisplayPosition       string `json:"display_position"`
	EligiblePositions     []struct {
		Position string `json:"position"`
	} `json:"eligible_positions"`
	SelectedPosition *struct {
		Position string `json:"position"`
	} `json:"selected_position,omitempty"`
	PlayerStats *struct {
		CoverageType string `json:"coverage_type"`
		Week         string `json:"week,omitempty"`
		Stats        struct {
			Stat []struct {
				StatID int    `json:"stat_id"`
				Value  string `json:"value"`
			} `json:"stat"`
		} `json:"stats"`
	} `json:"player_stats,omitempty"`
	PlayerPoints *struct {
		CoverageType string `json:"coverage_type"`
		Week         string `json:"week,omitempty"`
		Total        string `json:"total"`
	} `json:"player_points,omitempty"`
}

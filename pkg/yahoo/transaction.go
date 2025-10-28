package yahoo

type Transaction struct {
	TransactionKey string               `json:"transaction_key"`
	TransactionID  string               `json:"transaction_id"`
	Type           string               `json:"type"`
	Status         string               `json:"status"`
	Timestamp      int64                `json:"timestamp"`
	FAABBid        int                  `json:"faab_bid,omitempty"`
	Players        []TransactionPlayer  `json:"players"`
}

type TransactionPlayer struct {
	PlayerKey         string `json:"player_key"`
	PlayerID          string `json:"player_id"`
	Name              PlayerName `json:"name"`
	TransactionData   TransactionData `json:"transaction_data"`
}

type TransactionData struct {
	Type               string `json:"type"`
	SourceType         string `json:"source_type"`
	SourceTeamKey      string `json:"source_team_key,omitempty"`
	SourceTeamName     string `json:"source_team_name,omitempty"`
	DestinationType    string `json:"destination_type"`
	DestinationTeamKey string `json:"destination_team_key,omitempty"`
	DestinationTeamName string `json:"destination_team_name,omitempty"`
}

type yahooTransactionsResponse struct {
	FantasyContent struct {
		League struct {
			Transactions []struct {
				Transaction yahooTransactionData `json:"transaction"`
			} `json:"transactions"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooTransactionData struct {
	TransactionKey string `json:"transaction_key"`
	TransactionID  string `json:"transaction_id"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
	FAABBid        string `json:"faab_bid,omitempty"`
	Players        []struct {
		Player struct {
			PlayerKey string `json:"player_key"`
			PlayerID  string `json:"player_id"`
			Name      struct {
				Full       string `json:"full"`
				First      string `json:"first"`
				Last       string `json:"last"`
				ASCIIFirst string `json:"ascii_first"`
				ASCIILast  string `json:"ascii_last"`
			} `json:"name"`
			TransactionData struct {
				Type                string `json:"type"`
				SourceType          string `json:"source_type"`
				SourceTeamKey       string `json:"source_team_key,omitempty"`
				SourceTeamName      string `json:"source_team_name,omitempty"`
				DestinationType     string `json:"destination_type"`
				DestinationTeamKey  string `json:"destination_team_key,omitempty"`
				DestinationTeamName string `json:"destination_team_name,omitempty"`
			} `json:"transaction_data"`
		} `json:"player"`
	} `json:"players"`
}

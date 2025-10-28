package yahoo

import (
	"testing"
)

func TestConvertYahooPlayerToPlayer(t *testing.T) {
	yahooPlayer := yahooPlayerData{
		PlayerKey:             "423.p.12345",
		PlayerID:              "12345",
		EditorialTeamKey:      "nfl.team.1",
		EditorialTeamFullName: "Kansas City Chiefs",
		EditorialTeamAbbr:     "KC",
		DisplayPosition:       "QB",
	}
	yahooPlayer.Name.Full = "Patrick Mahomes"
	yahooPlayer.Name.First = "Patrick"
	yahooPlayer.Name.Last = "Mahomes"

	player := convertYahooPlayerToPlayer(yahooPlayer)

	if player.PlayerKey != "423.p.12345" {
		t.Errorf("PlayerKey = %v, want %v", player.PlayerKey, "423.p.12345")
	}
	if player.Name.Full != "Patrick Mahomes" {
		t.Errorf("Name.Full = %v, want %v", player.Name.Full, "Patrick Mahomes")
	}
	if player.DisplayPosition != "QB" {
		t.Errorf("DisplayPosition = %v, want %v", player.DisplayPosition, "QB")
	}
}

func TestConvertYahooStandingsTeam(t *testing.T) {
	yahooTeam := yahooStandingsTeamData{
		TeamKey: "423.l.12345.t.1",
		TeamID:  "1",
		Name:    "Test Team",
	}
	yahooTeam.TeamStandings.Rank = "1"
	yahooTeam.TeamStandings.OutcomeTotals.Wins = "10"
	yahooTeam.TeamStandings.OutcomeTotals.Losses = "3"
	yahooTeam.TeamStandings.OutcomeTotals.Ties = "0"
	yahooTeam.TeamStandings.OutcomeTotals.Percentage = "0.769"
	yahooTeam.TeamStandings.PointsFor = "1250.50"
	yahooTeam.TeamStandings.PointsAgainst = "1100.25"

	team := convertYahooStandingsTeam(yahooTeam)

	if team.TeamStandings.Rank != 1 {
		t.Errorf("Rank = %v, want %v", team.TeamStandings.Rank, 1)
	}
	if team.TeamStandings.OutcomeTotals.Wins != 10 {
		t.Errorf("Wins = %v, want %v", team.TeamStandings.OutcomeTotals.Wins, 10)
	}
	if team.TeamStandings.OutcomeTotals.Losses != 3 {
		t.Errorf("Losses = %v, want %v", team.TeamStandings.OutcomeTotals.Losses, 3)
	}
	if team.TeamStandings.PointsFor != 1250.50 {
		t.Errorf("PointsFor = %v, want %v", team.TeamStandings.PointsFor, 1250.50)
	}
}

func TestConvertYahooDraftResult(t *testing.T) {
	yahooDraft := yahooDraftResultData{
		Pick:    "1",
		Round:   "1",
		TeamKey: "423.l.12345.t.1",
	}
	yahooDraft.Players.Player.PlayerKey = "423.p.12345"
	yahooDraft.Players.Player.PlayerID = "12345"
	yahooDraft.Players.Player.Name.Full = "Christian McCaffrey"

	result := convertYahooDraftResult(yahooDraft)

	if result.Pick != 1 {
		t.Errorf("Pick = %v, want %v", result.Pick, 1)
	}
	if result.Round != 1 {
		t.Errorf("Round = %v, want %v", result.Round, 1)
	}
	if result.Player.Name.Full != "Christian McCaffrey" {
		t.Errorf("Player.Name.Full = %v, want %v", result.Player.Name.Full, "Christian McCaffrey")
	}
}

func TestConvertYahooTransaction(t *testing.T) {
	yahooTrans := yahooTransactionData{
		TransactionKey: "423.l.12345.tr.1",
		TransactionID:  "1",
		Type:           "add/drop",
		Status:         "successful",
		Timestamp:      "1609459200",
		FAABBid:        "25",
	}

	trans := convertYahooTransaction(yahooTrans)

	if trans.Type != "add/drop" {
		t.Errorf("Type = %v, want %v", trans.Type, "add/drop")
	}
	if trans.Status != "successful" {
		t.Errorf("Status = %v, want %v", trans.Status, "successful")
	}
	if trans.FAABBid != 25 {
		t.Errorf("FAABBid = %v, want %v", trans.FAABBid, 25)
	}
	if trans.Timestamp != 1609459200 {
		t.Errorf("Timestamp = %v, want %v", trans.Timestamp, 1609459200)
	}
}

func TestConvertYahooMatchup(t *testing.T) {
	yahooMatchup := yahooMatchupData{
		Week:          "1",
		WeekStart:     "2024-09-05",
		WeekEnd:       "2024-09-11",
		Status:        "postevent",
		IsPlayoffs:    "0",
		IsConsolation: "0",
		IsTied:        "0",
		WinnerTeamKey: "423.l.12345.t.1",
	}

	matchup := convertYahooMatchup(yahooMatchup)

	if matchup.Week != 1 {
		t.Errorf("Week = %v, want %v", matchup.Week, 1)
	}
	if matchup.Status != "postevent" {
		t.Errorf("Status = %v, want %v", matchup.Status, "postevent")
	}
	if matchup.IsPlayoffs {
		t.Errorf("IsPlayoffs = %v, want %v", matchup.IsPlayoffs, false)
	}
	if matchup.WinnerTeamKey != "423.l.12345.t.1" {
		t.Errorf("WinnerTeamKey = %v, want %v", matchup.WinnerTeamKey, "423.l.12345.t.1")
	}
}

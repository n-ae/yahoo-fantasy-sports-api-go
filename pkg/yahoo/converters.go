package yahoo

import (
	"strconv"
)

func convertYahooPlayerToPlayer(yp yahooPlayerData) Player {
	player := Player{
		PlayerKey:             yp.PlayerKey,
		PlayerID:              yp.PlayerID,
		Name:                  PlayerName{
			Full:       yp.Name.Full,
			First:      yp.Name.First,
			Last:       yp.Name.Last,
			ASCIIFirst: yp.Name.ASCIIFirst,
			ASCIILast:  yp.Name.ASCIILast,
		},
		EditorialTeamKey:      yp.EditorialTeamKey,
		EditorialTeamFullName: yp.EditorialTeamFullName,
		EditorialTeamAbbr:     yp.EditorialTeamAbbr,
		DisplayPosition:       yp.DisplayPosition,
	}

	for _, pos := range yp.EligiblePositions {
		player.EligiblePositions = append(player.EligiblePositions, pos.Position)
	}

	if yp.SelectedPosition != nil {
		player.SelectedPosition = SelectedPosition{
			Position: yp.SelectedPosition.Position,
		}
	}

	if yp.PlayerStats != nil {
		weekNum := 0
		if yp.PlayerStats.Week != "" {
			weekNum, _ = strconv.Atoi(yp.PlayerStats.Week)
		}

		var stats []Stat
		for _, s := range yp.PlayerStats.Stats.Stat {
			stats = append(stats, Stat{
				StatID: s.StatID,
				Value:  s.Value,
			})
		}

		player.PlayerStats = &PlayerStats{
			CoverageType: yp.PlayerStats.CoverageType,
			Week:         weekNum,
			Stats:        stats,
		}
	}

	if yp.PlayerPoints != nil {
		weekNum := 0
		if yp.PlayerPoints.Week != "" {
			weekNum, _ = strconv.Atoi(yp.PlayerPoints.Week)
		}

		total := 0.0
		if yp.PlayerPoints.Total != "" {
			total, _ = strconv.ParseFloat(yp.PlayerPoints.Total, 64)
		}

		player.PlayerPoints = &PlayerPoints{
			CoverageType: yp.PlayerPoints.CoverageType,
			Week:         weekNum,
			Total:        total,
		}
	}

	return player
}

func convertYahooStandingsTeam(yt yahooStandingsTeamData) StandingsTeam {
	rank, _ := strconv.Atoi(yt.TeamStandings.Rank)
	playoffSeed := 0
	if yt.TeamStandings.PlayoffSeed != "" {
		playoffSeed, _ = strconv.Atoi(yt.TeamStandings.PlayoffSeed)
	}

	wins, _ := strconv.Atoi(yt.TeamStandings.OutcomeTotals.Wins)
	losses, _ := strconv.Atoi(yt.TeamStandings.OutcomeTotals.Losses)
	ties, _ := strconv.Atoi(yt.TeamStandings.OutcomeTotals.Ties)
	percentage, _ := strconv.ParseFloat(yt.TeamStandings.OutcomeTotals.Percentage, 64)

	pointsFor, _ := strconv.ParseFloat(yt.TeamStandings.PointsFor, 64)
	pointsAgainst, _ := strconv.ParseFloat(yt.TeamStandings.PointsAgainst, 64)

	team := StandingsTeam{
		TeamKey: yt.TeamKey,
		TeamID:  yt.TeamID,
		Name:    yt.Name,
		TeamStandings: TeamStandings{
			Rank:        rank,
			PlayoffSeed: playoffSeed,
			OutcomeTotals: OutcomeTotals{
				Wins:       wins,
				Losses:     losses,
				Ties:       ties,
				Percentage: percentage,
			},
			PointsFor:     pointsFor,
			PointsAgainst: pointsAgainst,
			GamesBack:     yt.TeamStandings.GamesBack,
		},
	}

	if yt.TeamStandings.Streak != nil {
		streakVal, _ := strconv.Atoi(yt.TeamStandings.Streak.Value)
		team.TeamStandings.Streak = &Streak{
			Type:  yt.TeamStandings.Streak.Type,
			Value: streakVal,
		}
	}

	for _, m := range yt.Managers {
		isComm := m.Manager.IsCommissioner == "1"
		isCurrent := m.Manager.IsCurrentLogin == "1"
		team.Managers = append(team.Managers, Manager{
			ManagerID:      m.Manager.ManagerID,
			Nickname:       m.Manager.Nickname,
			GUID:           m.Manager.GUID,
			IsCommissioner: isComm,
			IsCurrentLogin: isCurrent,
		})
	}

	if len(team.Managers) > 0 {
		team.ManagerNickname = team.Managers[0].Nickname
	}

	return team
}

func convertYahooMatchup(ym yahooMatchupData) Matchup {
	weekNum, _ := strconv.Atoi(ym.Week)
	isPlayoffs := ym.IsPlayoffs == "1"
	isConsolation := ym.IsConsolation == "1"
	isTied := ym.IsTied == "1"

	matchup := Matchup{
		Week:          weekNum,
		WeekStart:     ym.WeekStart,
		WeekEnd:       ym.WeekEnd,
		Status:        ym.Status,
		IsPlayoffs:    isPlayoffs,
		IsConsolation: isConsolation,
		IsTied:        isTied,
		WinnerTeamKey: ym.WinnerTeamKey,
	}

	for _, t := range ym.Teams.Team {
		weekNum, _ := strconv.Atoi(t.TeamPoints.Week)
		points, _ := strconv.ParseFloat(t.TeamPoints.Total, 64)
		projPoints, _ := strconv.ParseFloat(t.TeamProjectedPoints.Total, 64)

		team := MatchupTeam{
			TeamKey: t.TeamKey,
			TeamID:  t.TeamID,
			Name:    t.Name,
			Points:  points,
			ProjectedPoints: projPoints,
			IsWinner: t.TeamKey == ym.WinnerTeamKey,
			TeamPoints: TeamPoints{
				CoverageType: t.TeamPoints.CoverageType,
				Week:         weekNum,
				Total:        points,
			},
			TeamProjectedPoints: TeamProjectedPoints{
				CoverageType: t.TeamProjectedPoints.CoverageType,
				Week:         weekNum,
				Total:        projPoints,
			},
		}

		if t.TeamStats != nil {
			var stats []Stat
			for _, s := range t.TeamStats.Stats.Stat {
				stats = append(stats, Stat{
					StatID: s.StatID,
					Value:  s.Value,
				})
			}
			team.Stats = stats
		}

		matchup.Teams = append(matchup.Teams, team)
	}

	return matchup
}

func convertYahooDraftResult(ydr yahooDraftResultData) DraftResult {
	pick, _ := strconv.Atoi(ydr.Pick)
	round, _ := strconv.Atoi(ydr.Round)

	return DraftResult{
		Pick:      pick,
		Round:     round,
		TeamKey:   ydr.TeamKey,
		PlayerKey: ydr.Players.Player.PlayerKey,
		Player:    convertYahooPlayerToPlayer(ydr.Players.Player),
	}
}

func convertYahooTransaction(yt yahooTransactionData) Transaction {
	timestamp, _ := strconv.ParseInt(yt.Timestamp, 10, 64)
	faabBid := 0
	if yt.FAABBid != "" {
		faabBid, _ = strconv.Atoi(yt.FAABBid)
	}

	trans := Transaction{
		TransactionKey: yt.TransactionKey,
		TransactionID:  yt.TransactionID,
		Type:           yt.Type,
		Status:         yt.Status,
		Timestamp:      timestamp,
		FAABBid:        faabBid,
	}

	for _, p := range yt.Players {
		trans.Players = append(trans.Players, TransactionPlayer{
			PlayerKey: p.Player.PlayerKey,
			PlayerID:  p.Player.PlayerID,
			Name: PlayerName{
				Full:       p.Player.Name.Full,
				First:      p.Player.Name.First,
				Last:       p.Player.Name.Last,
				ASCIIFirst: p.Player.Name.ASCIIFirst,
				ASCIILast:  p.Player.Name.ASCIILast,
			},
			TransactionData: TransactionData{
				Type:                p.Player.TransactionData.Type,
				SourceType:          p.Player.TransactionData.SourceType,
				SourceTeamKey:       p.Player.TransactionData.SourceTeamKey,
				SourceTeamName:      p.Player.TransactionData.SourceTeamName,
				DestinationType:     p.Player.TransactionData.DestinationType,
				DestinationTeamKey:  p.Player.TransactionData.DestinationTeamKey,
				DestinationTeamName: p.Player.TransactionData.DestinationTeamName,
			},
		})
	}

	return trans
}

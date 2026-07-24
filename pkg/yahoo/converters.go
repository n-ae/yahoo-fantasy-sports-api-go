package yahoo

import "fmt"

func convertYahooPlayerToPlayer(yp yahooPlayerData) Player {
	var d decoder
	player := convertYahooPlayerWith(&d, yp)
	player.DecodeWarnings = d.warnings
	return player
}

// convertYahooPlayerWith converts a player, recording warnings into the shared
// decoder d. Used directly by callers (draft, transaction) that want the
// warnings merged into their own container instead of onto the nested Player.
func convertYahooPlayerWith(d *decoder, yp yahooPlayerData) Player {
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
		var stats []Stat
		for _, s := range yp.PlayerStats.Stats.Stat {
			stats = append(stats, Stat{
				StatID: s.StatID,
				Value:  s.Value,
			})
		}

		player.PlayerStats = &PlayerStats{
			CoverageType: yp.PlayerStats.CoverageType,
			Week:         d.atoi("player_stats.week", yp.PlayerStats.Week),
			Stats:        stats,
		}
	}

	if yp.PlayerPoints != nil {
		player.PlayerPoints = &PlayerPoints{
			CoverageType: yp.PlayerPoints.CoverageType,
			Week:         d.atoi("player_points.week", yp.PlayerPoints.Week),
			Total:        d.parseFloat("player_points.total", yp.PlayerPoints.Total),
		}
	}

	return player
}

func convertYahooStandingsTeam(yt yahooStandingsTeamData) StandingsTeam {
	var d decoder
	rank := d.atoi("rank", yt.TeamStandings.Rank)
	playoffSeed := d.atoi("playoff_seed", yt.TeamStandings.PlayoffSeed)

	wins := d.atoi("outcome_totals.wins", yt.TeamStandings.OutcomeTotals.Wins)
	losses := d.atoi("outcome_totals.losses", yt.TeamStandings.OutcomeTotals.Losses)
	ties := d.atoi("outcome_totals.ties", yt.TeamStandings.OutcomeTotals.Ties)
	percentage := d.parseFloat("outcome_totals.percentage", yt.TeamStandings.OutcomeTotals.Percentage)

	pointsFor := d.parseFloat("points_for", yt.TeamStandings.PointsFor)
	pointsAgainst := d.parseFloat("points_against", yt.TeamStandings.PointsAgainst)

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
		team.TeamStandings.Streak = &Streak{
			Type:  yt.TeamStandings.Streak.Type,
			Value: d.atoi("streak.value", yt.TeamStandings.Streak.Value),
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

	team.DecodeWarnings = d.warnings
	return team
}

func convertYahooMatchup(ym yahooMatchupData) Matchup {
	var d decoder
	isPlayoffs := ym.IsPlayoffs == "1"
	isConsolation := ym.IsConsolation == "1"
	isTied := ym.IsTied == "1"

	matchup := Matchup{
		Week:          d.atoi("week", ym.Week),
		WeekStart:     ym.WeekStart,
		WeekEnd:       ym.WeekEnd,
		Status:        ym.Status,
		IsPlayoffs:    isPlayoffs,
		IsConsolation: isConsolation,
		IsTied:        isTied,
		WinnerTeamKey: ym.WinnerTeamKey,
	}

	for i, t := range ym.Teams.Team {
		var td decoder
		prefix := fmt.Sprintf("teams[%d]", i)
		weekNum := td.atoi("team_points.week", t.TeamPoints.Week)
		points := td.parseFloat("team_points.total", t.TeamPoints.Total)
		projPoints := td.parseFloat("team_projected_points.total", t.TeamProjectedPoints.Total)

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

		d.merge(prefix, td.warnings)
		matchup.Teams = append(matchup.Teams, team)
	}

	matchup.DecodeWarnings = d.warnings
	return matchup
}

func convertYahooDraftResult(ydr yahooDraftResultData) DraftResult {
	var d decoder
	return DraftResult{
		Pick:           d.atoi("pick", ydr.Pick),
		Round:          d.atoi("round", ydr.Round),
		TeamKey:        ydr.TeamKey,
		PlayerKey:      ydr.Players.Player.PlayerKey,
		Player:         convertYahooPlayerToPlayer(ydr.Players.Player),
		DecodeWarnings: d.warnings,
	}
}

func convertYahooTransaction(yt yahooTransactionData) Transaction {
	var d decoder
	trans := Transaction{
		TransactionKey: yt.TransactionKey,
		TransactionID:  yt.TransactionID,
		Type:           yt.Type,
		Status:         yt.Status,
		Timestamp:      d.parseInt64("timestamp", yt.Timestamp),
		FAABBid:        d.atoi("faab_bid", yt.FAABBid),
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

	trans.DecodeWarnings = d.warnings
	return trans
}

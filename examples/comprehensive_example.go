package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/yahoo"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./fantasy.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	client := yahoo.NewClient("", "", db)
	ctx := context.Background()

	fmt.Println("=== Yahoo Fantasy API - Comprehensive Example ===")
	fmt.Println()

	gameKey, err := yahoo.GetGameKey("nfl", 2024)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("NFL 2024 Game Key: %s\n", gameKey)
	fmt.Println()

	fmt.Println("1. Fetching User Leagues...")
	leagues, err := client.GetUserLeagues(ctx, gameKey)
	if err != nil {
		log.Printf("Error fetching leagues: %v\n", err)
		os.Exit(1)
	}

	if len(leagues) == 0 {
		fmt.Println("No leagues found for this game/season")
		return
	}

	league := leagues[0]
	leagueKey := fmt.Sprintf("%s.l.%s", league.YahooGameKey, league.YahooLeagueID)
	fmt.Printf("   League: %s (%s, %d teams)\n", league.LeagueName, league.ScoringType, league.NumTeams)
	fmt.Println()

	fmt.Println("2. Fetching League Standings...")
	standings, err := client.GetLeagueStandings(ctx, leagueKey)
	if err != nil {
		log.Printf("Error fetching standings: %v\n", err)
	} else {
		for i, team := range standings.Teams {
			if i >= 3 {
				break
			}
			fmt.Printf("   #%d %s (%d-%d-%d) - %.2f PF, %.2f PA\n",
				team.TeamStandings.Rank,
				team.Name,
				team.TeamStandings.OutcomeTotals.Wins,
				team.TeamStandings.OutcomeTotals.Losses,
				team.TeamStandings.OutcomeTotals.Ties,
				team.TeamStandings.PointsFor,
				team.TeamStandings.PointsAgainst)
		}
	}
	fmt.Println()

	fmt.Println("3. Fetching League Teams...")
	teams, err := client.GetLeagueTeams(ctx, leagueKey)
	if err != nil {
		log.Printf("Error fetching teams: %v\n", err)
	} else {
		fmt.Printf("   Found %d teams\n", len(teams))
		if len(teams) > 0 {
			fmt.Printf("   First team: %s (Manager: %s)\n", teams[0].TeamName, teams[0].ManagerName)
		}
	}
	fmt.Println()

	fmt.Println("4. Fetching Current Week Matchups...")
	if league.CurrentWeek > 0 {
		matchups, err := client.GetLeagueMatchups(ctx, leagueKey, league.CurrentWeek)
		if err != nil {
			log.Printf("Error fetching matchups: %v\n", err)
		} else {
			for i, matchup := range matchups {
				if i >= 2 {
					break
				}
				if len(matchup.Teams) >= 2 {
					team1 := matchup.Teams[0]
					team2 := matchup.Teams[1]
					fmt.Printf("   %s (%.2f) vs %s (%.2f)",
						team1.Name, team1.Points,
						team2.Name, team2.Points)
					if matchup.Status == "postevent" {
						if team1.IsWinner {
							fmt.Printf(" - Winner: %s\n", team1.Name)
						} else if team2.IsWinner {
							fmt.Printf(" - Winner: %s\n", team2.Name)
						} else {
							fmt.Printf(" - Tie\n")
						}
					} else {
						fmt.Printf(" - Status: %s\n", matchup.Status)
					}
				}
			}
		}
	}
	fmt.Println()

	fmt.Println("5. Fetching League Players (Free Agents)...")
	players, err := client.GetLeaguePlayers(ctx, leagueKey, yahoo.PlayerStatusFreeAgents, 0, 5)
	if err != nil {
		log.Printf("Error fetching players: %v\n", err)
	} else {
		fmt.Printf("   Found %d free agents\n", len(players))
		for i, player := range players {
			if i >= 3 {
				break
			}
			fmt.Printf("   - %s (%s - %s)\n",
				player.Name.Full,
				player.DisplayPosition,
				player.EditorialTeamAbbr)
		}
	}
	fmt.Println()

	fmt.Println("6. Fetching Draft Results...")
	draftResults, err := client.GetLeagueDraftResults(ctx, leagueKey)
	if err != nil {
		log.Printf("Error fetching draft results: %v\n", err)
	} else {
		fmt.Printf("   Total picks: %d\n", len(draftResults))
		for i, result := range draftResults {
			if i >= 5 {
				break
			}
			fmt.Printf("   Round %d, Pick %d: %s - %s\n",
				result.Round,
				result.Pick,
				result.Player.Name.Full,
				result.Player.DisplayPosition)
		}
	}
	fmt.Println()

	fmt.Println("7. Fetching Recent Transactions...")
	transactions, err := client.GetLeagueTransactions(ctx, leagueKey)
	if err != nil {
		log.Printf("Error fetching transactions: %v\n", err)
	} else {
		fmt.Printf("   Total transactions: %d\n", len(transactions))
		for i, trans := range transactions {
			if i >= 3 {
				break
			}
			fmt.Printf("   %s - %s", trans.Type, trans.Status)
			if trans.FAABBid > 0 {
				fmt.Printf(" ($%d FAAB)", trans.FAABBid)
			}
			fmt.Println()
			for _, player := range trans.Players {
				fmt.Printf("     %s: %s -> %s\n",
					player.Name.Full,
					player.TransactionData.SourceType,
					player.TransactionData.DestinationType)
			}
		}
	}
	fmt.Println()

	if len(teams) > 0 {
		teamKey := teams[0].YahooTeamKey
		fmt.Printf("8. Fetching Roster for Team: %s...\n", teams[0].TeamName)
		roster, err := client.GetTeamRoster(ctx, teamKey)
		if err != nil {
			log.Printf("Error fetching roster: %v\n", err)
		} else {
			fmt.Printf("   Roster size: %d players\n", len(roster))
			activeCount := 0
			for _, player := range roster {
				if player.IsStarting {
					activeCount++
				}
			}
			fmt.Printf("   Active players: %d\n", activeCount)
			fmt.Printf("   Bench players: %d\n", len(roster)-activeCount)
		}
	}

	fmt.Println()
	fmt.Println("=== Example Complete ===")
	fmt.Println()
	fmt.Println("Note: Some API calls may fail if you don't have proper Yahoo API credentials configured.")
	fmt.Println("Set YAHOO_CONSUMER_KEY, YAHOO_CONSUMER_SECRET, YAHOO_ACCESS_TOKEN, and YAHOO_REFRESH_TOKEN")
}

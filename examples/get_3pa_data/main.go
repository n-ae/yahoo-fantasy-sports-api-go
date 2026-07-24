package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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

	gameKey, err := yahoo.GetGameKey("nba", 2025)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("=== Getting NBA 3-Point Attempt Data ===")
	fmt.Println()

	leagues, err := client.GetUserLeagues(ctx, gameKey)
	if err != nil {
		log.Fatalf("Error fetching leagues: %v", err)
	}

	if len(leagues) == 0 {
		fmt.Println("No NBA leagues found for 2025 season")
		return
	}

	league := leagues[0]
	leagueKey := fmt.Sprintf("%s.l.%s", league.YahooGameKey, league.YahooLeagueID)
	fmt.Printf("League: %s\n", league.LeagueName)
	fmt.Println()

	fmt.Println("Fetching players...")
	players, err := client.GetLeaguePlayers(ctx, leagueKey, yahoo.PlayerStatusAll, 0, 10)
	if err != nil {
		log.Fatalf("Error fetching players: %v", err)
	}

	fmt.Printf("Found %d players. Getting stats...\n", len(players))
	fmt.Println()

	for i, player := range players {
		if i >= 5 {
			break
		}

		playerWithStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
		if err != nil {
			fmt.Printf("Error getting stats for %s: %v\n", player.Name.Full, err)
			continue
		}

		if playerWithStats.PlayerStats == nil {
			fmt.Printf("%s - No stats available\n", player.Name.Full)
			continue
		}

		nbaStats, err := yahoo.ParseNBAStats(playerWithStats.PlayerStats.Stats)
		if err != nil {
			fmt.Printf("Error parsing stats for %s: %v\n", player.Name.Full, err)
			continue
		}

		fmt.Printf("%-25s (%s)\n", player.Name.Full, player.EditorialTeamAbbr)
		fmt.Printf("  3-Pointers Made:      %d\n", nbaStats.ThreePointsMade)
		fmt.Printf("  3-Pointers Attempted: %d\n", nbaStats.ThreePointsAttempt)
		if nbaStats.ThreePointsAttempt > 0 {
			fmt.Printf("  3-Point %%:            %.1f%%\n", nbaStats.ThreePPercent*100)
		}
		fmt.Printf("  Total Points:         %d\n", nbaStats.Points)
		fmt.Printf("  Games Played:         %d\n", nbaStats.GamesPlayed)
		fmt.Println()
	}

	fmt.Println("=== Alternative: Direct Stat ID Access ===")
	fmt.Println()

	if len(players) > 0 {
		player := players[0]
		playerWithStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
		if err == nil && playerWithStats.PlayerStats != nil {
			helper := yahoo.NewStatHelper(playerWithStats.PlayerStats.Stats)

			if threePA, ok := helper.GetByID(yahoo.StatID3PA); ok {
				fmt.Printf("Player: %s\n", player.Name.Full)
				fmt.Printf("3PA (Stat ID 13): %s\n", threePA)
			}

			if threePM, ok := helper.GetByID(yahoo.StatID3PM); ok {
				fmt.Printf("3PM (Stat ID 12): %s\n", threePM)
			}

			if threePA, err := helper.GetIntByID(yahoo.StatID3PA); err == nil {
				fmt.Printf("3PA as int: %d\n", threePA)
			}
		}
	}

	fmt.Println()
	fmt.Println("=== Getting Weekly Stats ===")
	fmt.Println()

	if league.CurrentWeek > 0 && len(players) > 0 {
		player := players[0]
		fmt.Printf("Getting Week %d stats for %s...\n", league.CurrentWeek, player.Name.Full)

		weeklyStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, league.CurrentWeek)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else if weeklyStats.PlayerStats != nil {
			nbaStats, _ := yahoo.ParseNBAStats(weeklyStats.PlayerStats.Stats)
			fmt.Printf("  3-Pointers Made: %d\n", nbaStats.ThreePointsMade)
			fmt.Printf("  3-Pointers Attempted: %d\n", nbaStats.ThreePointsAttempt)
			fmt.Printf("  Points: %d\n", nbaStats.Points)
		}
	}

	fmt.Println()
	fmt.Println("=== Tip: Finding Your League's Stat IDs ===")
	fmt.Println("If the default stat IDs don't work, run this to see all stat IDs:")
	fmt.Println("  go run examples/get_player_stats.go <league_key> <player_key> 0")
	fmt.Println()
	fmt.Println("Common NBA Stat IDs:")
	fmt.Println("  12 = 3-Pointers Made (3PM)")
	fmt.Println("  13 = 3-Pointers Attempted (3PA)")
	fmt.Println("  14 = 3-Point % (3P%)")
}

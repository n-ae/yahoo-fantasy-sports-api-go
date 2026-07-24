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

	fmt.Println("=== Complete Shooting Stats Example ===")
	fmt.Println("Demonstrating FGA, FGM, FTA, FTM, 3PA, 3PM access")
	fmt.Println()

	gameKey, err := yahoo.GetGameKey("nba", 2025)
	if err != nil {
		log.Fatal(err)
	}

	leagues, err := client.GetUserLeagues(ctx, gameKey)
	if err != nil {
		log.Fatalf("Error fetching leagues: %v", err)
	}

	if len(leagues) == 0 {
		fmt.Println("No NBA leagues found. Please update with your league key.")
		return
	}

	league := leagues[0]
	leagueKey := fmt.Sprintf("%s.l.%s", league.YahooGameKey, league.YahooLeagueID)
	fmt.Printf("League: %s\n", league.LeagueName)
	fmt.Println()

	players, err := client.GetLeaguePlayers(ctx, leagueKey, yahoo.PlayerStatusAll, 0, 5)
	if err != nil {
		log.Fatalf("Error fetching players: %v", err)
	}

	if len(players) == 0 {
		fmt.Println("No players found")
		return
	}

	fmt.Println("=== Method 1: Parsed Structs (Recommended) ===")
	fmt.Println()

	for i, player := range players {
		if i >= 3 {
			break
		}

		playerWithStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
		if err != nil || playerWithStats.PlayerStats == nil {
			continue
		}

		nbaStats, err := yahoo.ParseNBAStats(playerWithStats.PlayerStats.Stats)
		if err != nil {
			continue
		}

		fmt.Printf("%-25s (%s - %s)\n",
			player.Name.Full,
			player.DisplayPosition,
			player.EditorialTeamAbbr)

		fmt.Printf("  FG:  %3d / %-3d  (%.1f%%)  [Stat IDs: 4, 3]\n",
			nbaStats.FGM, nbaStats.FGA, nbaStats.FGPercent*100)
		fmt.Printf("  FT:  %3d / %-3d  (%.1f%%)  [Stat IDs: 7, 6]\n",
			nbaStats.FTM, nbaStats.FTA, nbaStats.FTPercent*100)
		fmt.Printf("  3P:  %3d / %-3d  (%.1f%%)  [Stat IDs: 10, 9]\n",
			nbaStats.ThreePointsMade, nbaStats.ThreePointsAttempt, nbaStats.ThreePPercent*100)

		fmt.Printf("  Advanced: TS%% = %.1f%%, eFG%% = %.1f%%\n",
			nbaStats.TrueShootingPercent()*100,
			nbaStats.EffectiveFGPercent()*100)
		fmt.Printf("  Scoring: %d points in %d games\n", nbaStats.Points, nbaStats.GamesPlayed)
		fmt.Println()
	}

	fmt.Println("=== Method 2: Stat Helper with Constants ===")
	fmt.Println()

	if len(players) > 0 {
		player := players[0]
		playerWithStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
		if err == nil && playerWithStats.PlayerStats != nil {
			helper := yahoo.NewStatHelper(playerWithStats.PlayerStats.Stats)

			fmt.Printf("Player: %s\n", player.Name.Full)
			fmt.Println()

			fgm, _ := helper.GetIntByID(yahoo.StatIDFGM)
			fga, _ := helper.GetIntByID(yahoo.StatIDFGA)
			fmt.Printf("Field Goals:  FGM=%d (ID 4), FGA=%d (ID 3)\n", fgm, fga)

			ftm, _ := helper.GetIntByID(yahoo.StatIDFTM)
			fta, _ := helper.GetIntByID(yahoo.StatIDFTA)
			fmt.Printf("Free Throws:  FTM=%d (ID 7), FTA=%d (ID 6)\n", ftm, fta)

			tpm, _ := helper.GetIntByID(yahoo.StatID3PM)
			tpa, _ := helper.GetIntByID(yahoo.StatID3PA)
			fmt.Printf("3-Pointers:   3PM=%d (ID 10), 3PA=%d (ID 9)\n", tpm, tpa)
			fmt.Println()

			fmt.Println("Bulk access using GetShootingStats():")
			fgm2, fga2, ftm2, fta2, tpm2, tpa2, err := helper.GetShootingStats()
			if err == nil {
				fmt.Printf("  All shooting stats: FG=%d/%d, FT=%d/%d, 3P=%d/%d\n",
					fgm2, fga2, ftm2, fta2, tpm2, tpa2)
			}
		}
	}
	fmt.Println()

	fmt.Println("=== Method 3: Raw Access ===")
	fmt.Println()

	if len(players) > 0 {
		player := players[0]
		playerWithStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
		if err == nil && playerWithStats.PlayerStats != nil {
			fmt.Printf("Player: %s\n", player.Name.Full)
			fmt.Println("Raw stat array:")

			var fgm, fga, ftm, fta, tpm, tpa int
			for _, stat := range playerWithStats.PlayerStats.Stats {
				switch stat.StatID {
				case 4:
					fmt.Printf("  Stat ID  4 (FGM): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &fgm)
				case 3:
					fmt.Printf("  Stat ID  3 (FGA): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &fga)
				case 7:
					fmt.Printf("  Stat ID  7 (FTM): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &ftm)
				case 6:
					fmt.Printf("  Stat ID  6 (FTA): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &fta)
				case 10:
					fmt.Printf("  Stat ID 10 (3PM): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &tpm)
				case 9:
					fmt.Printf("  Stat ID  9 (3PA): %s\n", stat.Value)
					fmt.Sscanf(stat.Value, "%d", &tpa)
				}
			}
			fmt.Println()
			fmt.Printf("Manual calculation: FG=%d/%d, FT=%d/%d, 3P=%d/%d\n",
				fgm, fga, ftm, fta, tpm, tpa)
		}
	}
	fmt.Println()

	fmt.Println("=== Weekly Stats Example ===")
	fmt.Println()

	if league.CurrentWeek > 0 && len(players) > 0 {
		player := players[0]
		fmt.Printf("Getting Week %d stats for %s...\n", league.CurrentWeek, player.Name.Full)

		weeklyStats, err := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, league.CurrentWeek)
		if err == nil && weeklyStats.PlayerStats != nil {
			nbaStats, err := yahoo.ParseNBAStats(weeklyStats.PlayerStats.Stats)
			if err == nil {
				fmt.Printf("  FG:  %d/%d (%.1f%%)\n",
					nbaStats.FGM, nbaStats.FGA, nbaStats.FGPercent*100)
				fmt.Printf("  FT:  %d/%d (%.1f%%)\n",
					nbaStats.FTM, nbaStats.FTA, nbaStats.FTPercent*100)
				fmt.Printf("  3P:  %d/%d (%.1f%%)\n",
					nbaStats.ThreePointsMade, nbaStats.ThreePointsAttempt, nbaStats.ThreePPercent*100)
				fmt.Printf("  Points: %d\n", nbaStats.Points)
			}
		}
	}
	fmt.Println()

	fmt.Println("=== Summary ===")
	fmt.Println()
	fmt.Println("All Six Core Shooting Stats Supported:")
	fmt.Println("  ✓ FGA (Field Goals Attempted) - Stat ID 3")
	fmt.Println("  ✓ FGM (Field Goals Made) - Stat ID 4")
	fmt.Println("  ✓ FTA (Free Throws Attempted) - Stat ID 6")
	fmt.Println("  ✓ FTM (Free Throws Made) - Stat ID 7")
	fmt.Println("  ✓ 3PA (3-Point Attempts) - Stat ID 9")
	fmt.Println("  ✓ 3PM (3-Pointers Made) - Stat ID 10")
	fmt.Println()
	fmt.Println("Automatic Percentage Calculations:")
	fmt.Println("  ✓ FG% = FGM / FGA")
	fmt.Println("  ✓ FT% = FTM / FTA")
	fmt.Println("  ✓ 3P% = 3PM / 3PA")
	fmt.Println()
	fmt.Println("Advanced Metrics:")
	fmt.Println("  ✓ True Shooting % (TS%)")
	fmt.Println("  ✓ Effective Field Goal % (eFG%)")
}

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

	if len(os.Args) < 4 {
		fmt.Println("Usage: go run get_player_stats.go <league_key> <player_key> <week_num>")
		fmt.Println("Example: go run get_player_stats.go 466.l.12345 466.p.5479 0")
		fmt.Println("(Use week_num=0 for season stats)")
		return
	}

	leagueKey := os.Args[1]
	playerKey := os.Args[2]
	weekNum := 0
	fmt.Sscanf(os.Args[3], "%d", &weekNum)

	player, err := client.GetPlayerStats(ctx, leagueKey, playerKey, weekNum)
	if err != nil {
		log.Fatalf("Error fetching player stats: %v", err)
	}

	fmt.Printf("Player: %s (%s - %s)\n",
		player.Name.Full,
		player.DisplayPosition,
		player.EditorialTeamAbbr)
	fmt.Println()

	if player.PlayerStats != nil {
		fmt.Printf("Coverage: %s", player.PlayerStats.CoverageType)
		if player.PlayerStats.Week > 0 {
			fmt.Printf(" (Week %d)", player.PlayerStats.Week)
		}
		fmt.Println()
		fmt.Println()

		fmt.Println("All Stats:")
		for _, stat := range player.PlayerStats.Stats {
			fmt.Printf("  Stat ID %2d: %s\n", stat.StatID, stat.Value)
		}
	}

	if player.PlayerPoints != nil {
		fmt.Printf("\nTotal Fantasy Points: %.2f\n", player.PlayerPoints.Total)
	}

	fmt.Println("\n=== Common NBA Stat IDs ===")
	fmt.Println("These are typical NBA stat IDs (may vary by league):")
	fmt.Println("  0: Games Played")
	fmt.Println("  5: Field Goals Made (FGM)")
	fmt.Println("  6: Field Goals Attempted (FGA)")
	fmt.Println("  7: Field Goal % (FG%)")
	fmt.Println("  8: Free Throws Made (FTM)")
	fmt.Println("  9: Free Throws Attempted (FTA)")
	fmt.Println("  10: Free Throw % (FT%)")
	fmt.Println("  12: 3-Pointers Made (3PM)")
	fmt.Println("  13: 3-Pointers Attempted (3PA)")
	fmt.Println("  14: 3-Point % (3P%)")
	fmt.Println("  15: Points (PTS)")
	fmt.Println("  16: Total Rebounds (REB)")
	fmt.Println("  17: Offensive Rebounds (OREB)")
	fmt.Println("  18: Assists (AST)")
	fmt.Println("  19: Steals (STL)")
	fmt.Println("  20: Blocks (BLK)")
	fmt.Println("  21: Turnovers (TO)")
	fmt.Println()
	fmt.Println("Note: Stat IDs may vary based on your league's custom settings.")
	fmt.Println("Use the output above to identify the correct stat ID for your league.")
}

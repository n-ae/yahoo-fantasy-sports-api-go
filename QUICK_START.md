# Quick Start Guide

## Getting 3-Point Attempt Data

### Simple Example

```go
package main

import (
    "context"
    "fmt"
    "github.com/n-ae/yahoo-fantasy-sports-api-go/pkg/yahoo"
)

func main() {
    client := yahoo.NewClient("", "", db)
    ctx := context.Background()

    // Get player stats
    player, _ := client.GetPlayerStats(ctx,
        "466.l.12345",  // Your NBA league key
        "466.p.5479",   // Player key (e.g., Stephen Curry)
        0)              // 0 = season stats, or week number

    // Method 1: Use the NBA stats parser (easiest)
    if player.PlayerStats != nil {
        nbaStats, _ := yahoo.ParseNBAStats(player.PlayerStats.Stats)
        fmt.Printf("3PA: %d\n", nbaStats.ThreePointsAttempt)
        fmt.Printf("3PM: %d\n", nbaStats.ThreePointsMade)
    }

    // Method 2: Use the stat helper
    helper := yahoo.NewStatHelper(player.PlayerStats.Stats)
    threePA, _ := helper.GetIntByID(yahoo.StatID3PA)
    fmt.Printf("3PA: %d\n", threePA)
}
```

### Run the Examples

```bash
# See all available stats for a player
go run examples/get_player_stats.go 466.l.12345 466.p.5479 0

# Get 3PA data for multiple players
go run examples/get_3pa_data.go
```

## Finding Your League and Player Keys

### 1. Get Your League Key

```go
gameKey, _ := yahoo.GetGameKey("nba", 2025)
leagues, _ := client.GetUserLeagues(ctx, gameKey)

for _, league := range leagues {
    leagueKey := fmt.Sprintf("%s.l.%s",
        league.YahooGameKey,
        league.YahooLeagueID)
    fmt.Println(leagueKey) // e.g., "466.l.12345"
}
```

### 2. Get Player Keys

```go
// Get all players in league
players, _ := client.GetLeaguePlayers(ctx, leagueKey,
    yahoo.PlayerStatusAll, 0, 25)

for _, player := range players {
    fmt.Printf("%s: %s\n", player.Name.Full, player.PlayerKey)
}
```

## Common NBA Stat IDs

| Stat ID | Stat Name | Description |
|---------|-----------|-------------|
| 0 | GP | Games Played |
| 5 | FGM | Field Goals Made |
| 6 | FGA | Field Goals Attempted |
| 12 | 3PM | 3-Pointers Made |
| **13** | **3PA** | **3-Pointers Attempted** |
| 14 | 3P% | 3-Point Percentage |
| 15 | PTS | Points |
| 16 | REB | Total Rebounds |
| 18 | AST | Assists |
| 19 | STL | Steals |
| 20 | BLK | Blocks |
| 21 | TO | Turnovers |

## Weekly Stats

```go
// Get current week stats
weekStats, _ := client.GetPlayerStats(ctx, leagueKey, playerKey,
    league.CurrentWeek)

// Get specific week
week5Stats, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, 5)
```

## Troubleshooting

### "Stat ID not found"

Your league may use custom stat IDs. Run this to see all available stat IDs:

```bash
go run examples/get_player_stats.go <your_league_key> <player_key> 0
```

### "No stats available"

- Player may not have played any games yet
- Try a different player who has active stats
- Verify the league key is correct

### "401 Unauthorized"

Set your Yahoo API credentials:

```bash
export YAHOO_CONSUMER_KEY="your_key"
export YAHOO_CONSUMER_SECRET="your_secret"
export YAHOO_ACCESS_TOKEN="your_token"
export YAHOO_REFRESH_TOKEN="your_refresh_token"
```

## Next Steps

- See [README.md](README.md) for complete API documentation
- See [examples/](examples/) for more usage examples
- See [CHANGELOG.md](CHANGELOG.md) for all available features

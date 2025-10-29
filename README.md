# Yahoo Fantasy Sports API - Go SDK

A comprehensive Go SDK for the Yahoo Fantasy Sports API with support for NFL, MLB, NBA, and NHL fantasy leagues.

## Features

This SDK provides complete feature parity with the Python `yahoofantasy` package and includes:

- ✅ OAuth 2.0 authentication with automatic token refresh
- ✅ League, team, and player data retrieval
- ✅ Player statistics (weekly and season-long)
- ✅ Weekly matchups and scoring
- ✅ League standings with detailed outcomes
- ✅ Draft results tracking
- ✅ Transaction history (adds, drops, trades)
- ✅ Roster management (active vs bench)
- ✅ Built-in caching with configurable TTL
- ✅ Support for all four major sports (NFL, MLB, NBA, NHL)
- ✅ Game ID mapping for seasons 2001-2025

## Installation

```bash
go get github.com/n-ae/yahoo-fantasy-sports-api-go
```

## Quick Start

### Authentication

Set up your Yahoo Developer application credentials:

```bash
export YAHOO_CONSUMER_KEY="your_consumer_key"
export YAHOO_CONSUMER_SECRET="your_consumer_secret"
export YAHOO_ACCESS_TOKEN="your_access_token"
export YAHOO_REFRESH_TOKEN="your_refresh_token"
```

Enable caching (optional):

```bash
export YAHOO_ENABLE_CACHE="true"
```

### Basic Usage

```go
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

    gameKey, _ := yahoo.GetGameKey("nfl", 2024)
    leagues, err := client.GetUserLeagues(ctx, gameKey)
    if err != nil {
        log.Fatal(err)
    }

    for _, league := range leagues {
        fmt.Printf("%s - %s (%d teams)\n",
            league.LeagueName,
            league.ScoringType,
            league.NumTeams)
    }
}
```

## API Reference

### Game ID Mapping

Get Yahoo game IDs for different sports and seasons:

```go
gameID, err := yahoo.GetGameID("nfl", 2024)  // Returns: 449
gameKey, err := yahoo.GetGameKey("mlb", 2023) // Returns: "422"
```

Supported game codes: `nfl`, `mlb`, `nba`, `nhl`
Supported seasons: 2001-2025

### Leagues

#### Get User Leagues

```go
gameKey := "449" // NFL 2024
leagues, err := client.GetUserLeagues(ctx, gameKey)
```

#### Get League Teams

```go
leagueKey := "449.l.12345"
teams, err := client.GetLeagueTeams(ctx, leagueKey)
```

#### Get League Standings

```go
standings, err := client.GetLeagueStandings(ctx, leagueKey)
for _, team := range standings.Teams {
    fmt.Printf("#%d %s (%d-%d-%d)\n",
        team.TeamStandings.Rank,
        team.Name,
        team.TeamStandings.OutcomeTotals.Wins,
        team.TeamStandings.OutcomeTotals.Losses,
        team.TeamStandings.OutcomeTotals.Ties)
}
```

### Players

#### Get League Players

Retrieve all players in a league with optional status filtering:

```go
status := yahoo.PlayerStatusFreeAgents
players, err := client.GetLeaguePlayers(ctx, leagueKey, status, 0, 25)
```

Player status options:
- `PlayerStatusAll` - All players
- `PlayerStatusFreeAgents` - Free agents only
- `PlayerStatusWaivers` - Waivers only
- `PlayerStatusTaken` - Taken players only
- `PlayerStatusKeepers` - Keepers only

#### Get Player Stats

Get player statistics for a specific week or entire season:

```go
player, err := client.GetPlayerStats(ctx, leagueKey, playerKey, weekNum)
if player.PlayerStats != nil {
    for _, stat := range player.PlayerStats.Stats {
        fmt.Printf("Stat ID %d: %s\n", stat.StatID, stat.Value)
    }
}

if player.PlayerPoints != nil {
    fmt.Printf("Total Points: %.2f\n", player.PlayerPoints.Total)
}
```

Set `weekNum` to `0` for season-long stats.

### Matchups

#### Get Weekly Matchups

```go
weekNum := 1
matchups, err := client.GetLeagueMatchups(ctx, leagueKey, weekNum)

for _, matchup := range matchups {
    team1 := matchup.Teams[0]
    team2 := matchup.Teams[1]

    fmt.Printf("%s (%.2f) vs %s (%.2f)\n",
        team1.Name, team1.Points,
        team2.Name, team2.Points)

    if team1.IsWinner {
        fmt.Printf("Winner: %s\n", team1.Name)
    } else if team2.IsWinner {
        fmt.Printf("Winner: %s\n", team2.Name)
    }
}
```

### Rosters

#### Get Team Roster

```go
teamKey := "449.l.12345.t.1"
roster, err := client.GetTeamRoster(ctx, teamKey)

for _, player := range roster {
    fmt.Printf("%s - %s (%s)\n",
        player.PlayerKey,
        player.Position,
        player.SelectedPos)
}
```

### Draft Results

#### Get League Draft Results

```go
results, err := client.GetLeagueDraftResults(ctx, leagueKey)

for _, result := range results {
    fmt.Printf("Round %d, Pick %d: %s - %s\n",
        result.Round,
        result.Pick,
        result.Player.Name.Full,
        result.Player.DisplayPosition)
}
```

### Transactions

#### Get League Transactions

```go
transactions, err := client.GetLeagueTransactions(ctx, leagueKey)

for _, trans := range transactions {
    fmt.Printf("Type: %s, Status: %s\n", trans.Type, trans.Status)

    if trans.FAABBid > 0 {
        fmt.Printf("FAAB Bid: $%d\n", trans.FAABBid)
    }

    for _, player := range trans.Players {
        fmt.Printf("  %s: %s -> %s\n",
            player.Name.Full,
            player.TransactionData.SourceType,
            player.TransactionData.DestinationType)
    }
}
```

## Data Structures

### League

```go
type League struct {
    YahooLeagueID string
    YahooGameKey  string
    LeagueName    string
    SeasonYear    int
    ScoringType   string
    NumTeams      int
    CurrentWeek   int
}
```

### Player

```go
type Player struct {
    PlayerKey             string
    PlayerID              string
    Name                  PlayerName
    EditorialTeamKey      string
    EditorialTeamAbbr     string
    DisplayPosition       string
    EligiblePositions     []string
    SelectedPosition      SelectedPosition
    PlayerStats           *PlayerStats
    PlayerPoints          *PlayerPoints
}
```

### Matchup

```go
type Matchup struct {
    Week              int
    Status            string
    IsPlayoffs        bool
    IsConsolation     bool
    WinnerTeamKey     string
    Teams             []MatchupTeam
}

type MatchupTeam struct {
    TeamKey            string
    Name               string
    Points             float64
    ProjectedPoints    float64
    IsWinner           bool
    Stats              []Stat
}
```

### Standings

```go
type Standings struct {
    Teams []StandingsTeam
}

type StandingsTeam struct {
    TeamKey        string
    Name           string
    TeamStandings  TeamStandings
    Managers       []Manager
}

type TeamStandings struct {
    Rank            int
    OutcomeTotals   OutcomeTotals
    PointsFor       float64
    PointsAgainst   float64
    Streak          *Streak
}
```

### DraftResult

```go
type DraftResult struct {
    Pick      int
    Round     int
    TeamKey   string
    PlayerKey string
    Player    Player
}
```

### Transaction

```go
type Transaction struct {
    TransactionKey string
    Type           string
    Status         string
    Timestamp      int64
    FAABBid        int
    Players        []TransactionPlayer
}
```

## Caching

The SDK includes built-in caching to reduce API calls:

```bash
export YAHOO_ENABLE_CACHE="true"
```

Cache TTLs:
- User leagues: 24 hours
- League teams: 6 hours
- Team rosters: 1 hour
- Player stats: 2 hours
- Matchups: 1 hour
- Standings: 6 hours
- Draft results: 24 hours
- Transactions: 30 minutes

## Error Handling

All API methods return errors. Always check for errors:

```go
leagues, err := client.GetUserLeagues(ctx, gameKey)
if err != nil {
    log.Printf("Error fetching leagues: %v", err)
    return
}
```

The SDK automatically handles token refresh when the access token expires.

## Working with Player Stats

### Getting Specific Stats (e.g., 3-Point Attempts)

Yahoo uses Stat IDs to identify different statistics. Here are three ways to access them:

#### Method 1: Using the Stats Helper (Recommended for NBA)

```go
player, err := client.GetPlayerStats(ctx, leagueKey, playerKey, 0) // 0 = season stats

if player.PlayerStats != nil {
    nbaStats, err := yahoo.ParseNBAStats(player.PlayerStats.Stats)
    if err == nil {
        fmt.Printf("3-Point Attempts: %d\n", nbaStats.ThreePointsAttempt)
        fmt.Printf("3-Pointers Made: %d\n", nbaStats.ThreePointsMade)
        fmt.Printf("3-Point %%: %.1f%%\n", nbaStats.ThreePPercent * 100)
    }
}
```

#### Method 2: Direct Stat ID Access

```go
player, err := client.GetPlayerStats(ctx, leagueKey, playerKey, 0)

if player.PlayerStats != nil {
    helper := yahoo.NewStatHelper(player.PlayerStats.Stats)

    // Get as string
    if threePA, ok := helper.GetByID(yahoo.StatID3PA); ok {
        fmt.Printf("3PA: %s\n", threePA)
    }

    // Get as int
    if threePA, err := helper.GetIntByID(yahoo.StatID3PA); err == nil {
        fmt.Printf("3PA: %d\n", threePA)
    }
}
```

#### Method 3: Manual Loop Through Stats

```go
player, err := client.GetPlayerStats(ctx, leagueKey, playerKey, 0)

if player.PlayerStats != nil {
    for _, stat := range player.PlayerStats.Stats {
        if stat.StatID == 9 { // 9 = 3PA in NBA leagues
            fmt.Printf("3-Point Attempts: %s\n", stat.Value)
        }
    }
}
```

### Common NBA Stat IDs

```go
const (
    StatIDGamesPlayed       = 0   // GP
    StatIDGamesStarted      = 1   // GS
    StatIDMinutesPlayed     = 2   // MIN
    StatIDFGA               = 3   // Field Goals Attempted
    StatIDFGM               = 4   // Field Goals Made
    StatIDFGPercent         = 5   // Field Goal %
    StatIDFTA               = 6   // Free Throws Attempted
    StatIDFTM               = 7   // Free Throws Made
    StatIDFTPercent         = 8   // Free Throw %
    StatID3PA               = 9   // 3-Pointers Attempted
    StatID3PM               = 10  // 3-Pointers Made
    StatID3PPercent         = 11  // 3-Point %
    StatIDPoints            = 12  // Points
    StatIDOffensiveRebounds = 13  // Offensive Rebounds
    StatIDDefensiveRebounds = 14  // Defensive Rebounds
    StatIDRebounds          = 15  // Total Rebounds
    StatIDAssists           = 16  // Assists
    StatIDSteals            = 17  // Steals
    StatIDBlocks            = 18  // Blocks
    StatIDTurnovers         = 19  // Turnovers
    StatIDAssistTurnoverRatio = 20 // Assist/Turnover Ratio
    StatIDPersonalFouls     = 21  // Personal Fouls
)
```

**Note:** Stat IDs may vary if your league has custom scoring settings. To find your league's stat IDs:

```bash
go run examples/get_player_stats.go <league_key> <player_key> 0
```

### Weekly vs Season Stats

```go
// Get season stats
seasonStats, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, 0)

// Get week 5 stats
weekStats, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, 5)
```

### Complete Example

See `examples/get_3pa_data.go` for a complete example of retrieving and working with 3-point attempt data.

## Testing

Run the test suite:

```bash
go test ./pkg/yahoo/... -v
```

Run with coverage:

```bash
go test ./pkg/yahoo/... -cover
```

## Compatibility

This SDK provides complete feature parity with the Python `yahoofantasy` package (v1.4.9), including:

- ✅ All resource types (League, Team, Player, Week, Matchup, Standings, etc.)
- ✅ All API methods (get_leagues, players, standings, weeks, draft_results, transactions)
- ✅ Player filtering by status
- ✅ Weekly and season stats
- ✅ Draft results and transaction history
- ✅ Game ID mapping for all sports (2001-2025)
- ✅ Caching with configurable TTL

## Contributing

Issues and pull requests are welcome! Please ensure all tests pass before submitting.

## License

MIT License

## Credits

Based on the Python `yahoofantasy` package by Matt Dodge.

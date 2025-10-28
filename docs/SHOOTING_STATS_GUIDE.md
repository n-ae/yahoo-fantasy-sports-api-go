# Complete Guide: Accessing Shooting Stats (FGA, FGM, FTA, FTM, 3PA, 3PM)

## Quick Reference

All six core NBA shooting statistics are fully supported with three complementary access methods.

### The Six Core Stats

| Stat | Full Name | Stat ID | Type |
|------|-----------|---------|------|
| **FGM** | Field Goals Made | 5 | int |
| **FGA** | Field Goals Attempted | 6 | int |
| **FTM** | Free Throws Made | 8 | int |
| **FTA** | Free Throws Attempted | 9 | int |
| **3PM** | 3-Pointers Made | 12 | int |
| **3PA** | 3-Point Attempts | 13 | int |

### Three Ways to Access

## Method 1: Parsed Structs ⭐ Recommended

**Best for**: Standard leagues, clean code, automatic calculations

```go
player, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, 0)
nbaStats, _ := yahoo.ParseNBAStats(player.PlayerStats.Stats)

// All six stats with proper types
fmt.Printf("FG:  %d/%d (%.1f%%)\n",
    nbaStats.FGM, nbaStats.FGA, nbaStats.FGPercent*100)
fmt.Printf("FT:  %d/%d (%.1f%%)\n",
    nbaStats.FTM, nbaStats.FTA, nbaStats.FTPercent*100)
fmt.Printf("3P:  %d/%d (%.1f%%)\n",
    nbaStats.ThreePointsMade,
    nbaStats.ThreePointsAttempt,
    nbaStats.ThreePPercent*100)
```

**Features:**
- ✅ Type-safe field access
- ✅ Automatic percentage calculation if missing from API
- ✅ Zero-division protection
- ✅ Advanced metrics (TS%, eFG%)
- ✅ All stats in one struct

## Method 2: Stat Helper

**Best for**: Custom leagues, selective stat access, flexibility

```go
helper := yahoo.NewStatHelper(player.PlayerStats.Stats)

// Individual stat access
fgm, _ := helper.GetIntByID(yahoo.StatIDFGM)  // 5
fga, _ := helper.GetIntByID(yahoo.StatIDFGA)  // 6
ftm, _ := helper.GetIntByID(yahoo.StatIDFTM)  // 8
fta, _ := helper.GetIntByID(yahoo.StatIDFTA)  // 9
tpm, _ := helper.GetIntByID(yahoo.StatID3PM)  // 12
tpa, _ := helper.GetIntByID(yahoo.StatID3PA)  // 13

// Or bulk access
fgm, fga, ftm, fta, tpm, tpa, _ := helper.GetShootingStats()
```

**Features:**
- ✅ Named constants (no magic numbers)
- ✅ Type conversion (string → int/float)
- ✅ Works with custom stat IDs
- ✅ Bulk helper method
- ✅ Flexible for any stat

## Method 3: Raw Access

**Best for**: Debugging, unknown stat IDs, maximum control

```go
var fgm, fga, ftm, fta, tpm, tpa int

for _, stat := range player.PlayerStats.Stats {
    switch stat.StatID {
    case 5:  // FGM
        fmt.Sscanf(stat.Value, "%d", &fgm)
    case 6:  // FGA
        fmt.Sscanf(stat.Value, "%d", &fga)
    case 8:  // FTM
        fmt.Sscanf(stat.Value, "%d", &ftm)
    case 9:  // FTA
        fmt.Sscanf(stat.Value, "%d", &fta)
    case 12: // 3PM
        fmt.Sscanf(stat.Value, "%d", &tpm)
    case 13: // 3PA
        fmt.Sscanf(stat.Value, "%d", &tpa)
    }
}
```

**Features:**
- ✅ Direct API access
- ✅ No abstraction overhead
- ✅ Discover unknown stats
- ✅ Full control

## Complete Examples

### Get All Six Stats (Season)

```go
player, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, 0)
nbaStats, _ := yahoo.ParseNBAStats(player.PlayerStats.Stats)

fmt.Printf("%-20s %s\n", "Stat", "Value")
fmt.Printf("%-20s %d\n", "FGM", nbaStats.FGM)
fmt.Printf("%-20s %d\n", "FGA", nbaStats.FGA)
fmt.Printf("%-20s %d\n", "FTM", nbaStats.FTM)
fmt.Printf("%-20s %d\n", "FTA", nbaStats.FTA)
fmt.Printf("%-20s %d\n", "3PM", nbaStats.ThreePointsMade)
fmt.Printf("%-20s %d\n", "3PA", nbaStats.ThreePointsAttempt)
```

### Get All Six Stats (Weekly)

```go
weekStats, _ := client.GetPlayerStats(ctx, leagueKey, playerKey, weekNum)
nbaStats, _ := yahoo.ParseNBAStats(weekStats.PlayerStats.Stats)

// Same access as season stats
```

### Compare Multiple Players

```go
players, _ := client.GetLeaguePlayers(ctx, leagueKey,
    yahoo.PlayerStatusAll, 0, 10)

for _, player := range players {
    p, _ := client.GetPlayerStats(ctx, leagueKey, player.PlayerKey, 0)
    stats, _ := yahoo.ParseNBAStats(p.PlayerStats.Stats)

    fmt.Printf("%-25s FG:%d/%d FT:%d/%d 3P:%d/%d\n",
        player.Name.Full,
        stats.FGM, stats.FGA,
        stats.FTM, stats.FTA,
        stats.ThreePointsMade, stats.ThreePointsAttempt)
}
```

## Advanced Features

### Automatic Percentage Calculation

```go
nbaStats, _ := yahoo.ParseNBAStats(player.PlayerStats.Stats)

// Percentages automatically calculated if missing from API
fmt.Printf("FG%%: %.1f%%\n", nbaStats.FGPercent*100)
fmt.Printf("FT%%: %.1f%%\n", nbaStats.FTPercent*100)
fmt.Printf("3P%%: %.1f%%\n", nbaStats.ThreePPercent*100)
```

### Advanced Shooting Metrics

```go
// True Shooting Percentage
ts := nbaStats.TrueShootingPercent()
fmt.Printf("TS%%: %.1f%%\n", ts*100)

// Effective Field Goal Percentage
efg := nbaStats.EffectiveFGPercent()
fmt.Printf("eFG%%: %.1f%%\n", efg*100)
```

### Manual Calculation Methods

```go
// Recalculate percentages from raw values
fgPercent := nbaStats.CalculateFGPercent()
ftPercent := nbaStats.CalculateFTPercent()
tpPercent := nbaStats.Calculate3PPercent()

// All methods handle zero division safely
```

## Troubleshooting

### Custom League Stat IDs

If your league uses custom stat IDs:

```bash
# Discover actual stat IDs
go run examples/get_player_stats.go <league_key> <player_key> 0
```

Then use StatHelper:
```go
helper := yahoo.NewStatHelper(player.PlayerStats.Stats)
customFGA, _ := helper.GetIntByID(42) // Your custom ID
```

### Missing Stats

```go
// ParseNBAStats handles missing stats gracefully
nbaStats, _ := yahoo.ParseNBAStats(player.PlayerStats.Stats)

// Missing stats default to zero
if nbaStats.FGA == 0 {
    fmt.Println("Player has no field goal attempts")
}
```

### Zero Division Errors

All percentage methods handle zero attempts:

```go
// Returns 0.0 if FGA is 0 (no panic)
percent := nbaStats.CalculateFGPercent()
```

## Decision Tree

**Which method should I use?**

```
Are you using a standard Yahoo league?
├─ YES → Use Method 1 (Parsed Structs) ⭐
└─ NO  → Does your league have custom stat IDs?
         ├─ YES → Use Method 2 (Stat Helper)
         └─ NO  → Are you debugging or exploring?
                  ├─ YES → Use Method 3 (Raw Access)
                  └─ NO  → Use Method 1 (Parsed Structs)
```

## Performance Comparison

| Method | Setup Cost | Per-Stat Cost | Type Safety | LOC |
|--------|------------|---------------|-------------|-----|
| Parsed Structs | 1 call | Zero (struct field) | Strong | 3 |
| Stat Helper | 1 call | 1 map lookup | Medium | 6 |
| Raw Access | Zero | Full loop | Weak | 15+ |

## See Also

- **Complete Example**: `examples/shooting_stats_complete.go`
- **ADR**: `docs/adr/0001-expose-attempt-made-stats.md`
- **README**: Main documentation with full API reference
- **Quick Start**: `QUICK_START.md`

---

**All six stats (FGM, FGA, FTM, FTA, 3PM, 3PA) are fully supported with complete type safety and automatic calculations.**

# Architecture Gap Assessment: Python yahoofantasy vs Go Implementation

**Date:** 2025-10-28
**Project:** yahoo-fantasy-sports-api-go
**Purpose:** Identify gaps between Python yahoofantasy reference implementation and Go client

---

## Executive Summary

The Go implementation has a solid foundation with OAuth, caching, and repository patterns, but is missing approximately 60% of the core API client functionality present in the Python yahoofantasy package. The primary gaps are:

1. **Missing API endpoints**: Player stats, matchups, standings, draft results, transactions
2. **Missing domain entities**: Player, Week, Matchup, Standings, DraftResult, Transaction, Stat
3. **Limited query capabilities**: No player filtering, no weekly data retrieval, no historical tracking
4. **No CLI tooling**: Python has export tools for performances, matchups, drafts, transactions

**Maintainability Assessment**: The current Go architecture is well-structured with clean separation of concerns. Adding missing functionality will require careful attention to avoid over-abstraction while maintaining extensibility for multi-sport support.

---

## 1. Critical Missing API Client Methods

### Current State (`pkg/yahoo/client.go`)

The client currently exposes only 3 API methods:
- `GetUserLeagues(ctx, gameKey)` - Fetch user's leagues
- `GetLeagueTeams(ctx, leagueKey)` - Fetch league teams with basic standings
- `GetTeamRoster(ctx, teamKey)` - Fetch team roster without stats

### Required New Methods (Priority Order)

#### Phase 1: Core Read Operations (Weeks 1-2)

```go
// Player operations
GetLeaguePlayers(ctx context.Context, leagueKey string, status PlayerStatus) ([]Player, error)
GetPlayerStats(ctx context.Context, playerKey string, weekNum int) (*PlayerStats, error)
GetPlayerSeasonStats(ctx context.Context, playerKey string) (*PlayerStats, error)

// Weekly data
GetLeagueWeeks(ctx context.Context, leagueKey string) ([]Week, error)
GetWeekMatchups(ctx context.Context, leagueKey string, weekNum int) ([]Matchup, error)

// Standings
GetLeagueStandings(ctx context.Context, leagueKey string) (*Standings, error)
```

**Justification**: These methods provide the minimum viable data layer for fantasy analysis. Player stats and weekly matchups are the foundation of all fantasy operations.

#### Phase 2: Historical Data (Weeks 3-4)

```go
// Draft tracking
GetLeagueDraftResults(ctx context.Context, leagueKey string) ([]DraftResult, error)

// Transaction history
GetLeagueTransactions(ctx context.Context, leagueKey string, opts TransactionOptions) ([]Transaction, error)
GetTeamTransactions(ctx context.Context, teamKey string, opts TransactionOptions) ([]Transaction, error)

// League history
GetPastLeagueID(ctx context.Context, currentLeagueKey string, targetYear int) (string, error)
```

**Justification**: Draft and transaction data enable advanced analysis (keeper values, waiver strategies). Historical league linking enables multi-season tracking.

#### Phase 3: Advanced Queries (Week 5)

```go
// Roster with stats
GetTeamRosterWithStats(ctx context.Context, teamKey string, weekNum int) ([]RosterPlayer, error)

// League settings
GetLeagueSettings(ctx context.Context, leagueKey string) (*LeagueSettings, error)
GetLeagueStatCategories(ctx context.Context, leagueKey string) ([]StatCategory, error)

// Batch operations
GetMultiplePlayerStats(ctx context.Context, playerKeys []string, weekNum int) (map[string]*PlayerStats, error)
```

**Justification**: These optimize performance and reduce API calls for common operations. Settings metadata is required for accurate scoring calculations.

### API Response Handling Patterns

**Current Approach**: Direct struct mapping with nested JSON tags
```go
type yahooLeaguesResponse struct {
    Fantasy_Content struct {
        Users []struct {
            User []struct { ... } `json:"user"`
        } `json:"users"`
    } `json:"fantasy_content"`
}
```

**Maintainability Concern**: Yahoo's API response structure is deeply nested and inconsistent. Current approach requires complex traversal logic scattered across fetch methods.

**Recommendation**: Introduce intermediate response mappers
```go
// Internal response types (keep current approach)
type yahooLeaguesResponse struct { ... }

// Response mapper functions
func (r *yahooLeaguesResponse) ToLeagues() []League {
    // Centralized traversal logic
}

// Client method becomes simple
func (c *Client) fetchLeagues(ctx, gameKey) ([]League, error) {
    data := makeRequest(ctx, endpoint)
    var resp yahooLeaguesResponse
    json.Unmarshal(data, &resp)
    return resp.ToLeagues(), nil
}
```

This pattern:
- Isolates response structure complexity to mapper methods
- Makes client methods easier to test (mock mappers)
- Reduces duplication as more endpoints are added

---

## 2. New Entity Types Required

### Priority 1: Core Entities

#### Player Entity
```go
// pkg/yahoo/player.go
type Player struct {
    PlayerID        string
    PlayerKey       string
    FullName        string
    FirstName       string
    LastName        string
    Status          PlayerStatus  // A, FA, W, T, K (Available, Free Agent, Waivers, Taken, Keeper)
    EditorialTeamKey string
    DisplayPosition string
    EligiblePositions []string
    ImageURL        string
    IsUndroppable   bool
    Ownership       *PlayerOwnership
}

type PlayerOwnership struct {
    OwnershipType   string  // team, waivers, freeagents
    OwnerTeamKey    string
    OwnerTeamName   string
}

type PlayerStatus string
const (
    PlayerStatusAvailable  PlayerStatus = "A"
    PlayerStatusFreeAgent  PlayerStatus = "FA"
    PlayerStatusWaivers    PlayerStatus = "W"
    PlayerStatusTaken      PlayerStatus = "T"
    PlayerStatusKeeper     PlayerStatus = "K"
)
```

**Design Note**: Player is a value object, not an aggregate root. It represents Yahoo's player data, not your local player database. Separate concern from `players` table.

#### PlayerStats Entity
```go
// pkg/yahoo/player_stats.go
type PlayerStats struct {
    PlayerKey  string
    Week       int  // 0 for season stats
    Stats      map[string]StatValue
    Points     float64  // Fantasy points for week
}

type StatValue struct {
    StatID    int
    Value     float64
}
```

**Simplicity First**: Use `map[string]StatValue` instead of creating strongly-typed stat structs per sport. Avoids explosion of types while maintaining type safety for individual stats.

#### Week and Matchup Entities
```go
// pkg/yahoo/week.go
type Week struct {
    WeekNumber  int
    StartDate   string
    EndDate     string
    IsPlayoffs  bool
}

// pkg/yahoo/matchup.go
type Matchup struct {
    Week           int
    Team1Key       string
    Team1Points    float64
    Team1Projected float64
    Team2Key       string
    Team2Points    float64
    Team2Projected float64
    IsPlayoffs     bool
    IsConsolation  bool
    WinnerTeamKey  string
    IsTied         bool
}
```

**Design Note**: Keep matchup flat. Avoid nested Team objects to prevent circular dependencies and deep object graphs.

### Priority 2: Historical Entities

#### DraftResult
```go
// pkg/yahoo/draft.go
type DraftResult struct {
    Pick       int
    Round      int
    TeamKey    string
    PlayerKey  string
    PlayerName string
    Cost       int  // For auction drafts
}
```

#### Transaction
```go
// pkg/yahoo/transaction.go
type Transaction struct {
    TransactionKey string
    Type           TransactionType
    Status         string
    Timestamp      time.Time
    FAAPBid        int
    Players        []TransactionPlayer
}

type TransactionType string
const (
    TransactionTypeAdd      TransactionType = "add"
    TransactionTypeDrop     TransactionType = "drop"
    TransactionTypeTrade    TransactionType = "trade"
    TransactionTypeCommish  TransactionType = "commish"
)

type TransactionPlayer struct {
    PlayerKey     string
    PlayerName    string
    TransactionData map[string]string  // type-specific data
    SourceType    string              // team, waivers, freeagents
    SourceTeamKey string
    DestType      string
    DestTeamKey   string
}
```

**Extensibility**: `TransactionData` map handles type-specific fields (FAAB bid, trade notes, etc.) without requiring separate structs per transaction type.

### Priority 3: Enhanced Entities

#### Standings
```go
// pkg/yahoo/standings.go
type Standings struct {
    Teams []TeamStanding
}

type TeamStanding struct {
    TeamKey        string
    TeamName       string
    Rank           int
    OutcomeTotals  OutcomeTotals
    PointsFor      float64
    PointsAgainst  float64
    Streak         Streak
}

type OutcomeTotals struct {
    Wins       int
    Losses     int
    Ties       int
    Percentage float64
}

type Streak struct {
    Type   string  // "win", "loss", "tie"
    Length int
}
```

#### LeagueSettings
```go
// pkg/yahoo/league_settings.go
type LeagueSettings struct {
    LeagueKey       string
    NumTeams        int
    ScoringType     string
    MaxAdds         int
    TradeDeadline   string
    WaiverType      string
    WaiverRule      string
    CanTradeDraft   bool
    DraftType       string
    IsFinished      bool
    RosterPositions []RosterPosition
    StatCategories  []StatCategory
    StatModifiers   map[string]float64
}

type StatCategory struct {
    StatID       int
    Abbr         string
    Name         string
    DisplayName  string
    SortOrder    int
    PositionType string
}
```

---

## 3. Repository Pattern Recommendations

### Current Pattern Assessment

**Strengths**:
- Clean separation: repositories handle persistence, services orchestrate
- Consistent CRUD interface across entities
- Context propagation throughout

**Weaknesses**:
- No repository for Player (assumes players table exists but no interface)
- Tight coupling to SQLite in repository layer (SQL query strings)
- No abstraction for batch operations

### Recommended Repository Additions

#### PlayerRepository
```go
// pkg/repository/player_repository.go
type PlayerRepository struct {
    db *sql.DB
}

type Player struct {
    ID               int
    YahooPlayerKey   string
    YahooPlayerID    string
    FullName         string
    FirstName        string
    LastName         string
    Sport            string
    EditorialTeamKey string
    IsActive         bool
    CreatedAt        time.Time
    UpdatedAt        time.Time
}

func (r *PlayerRepository) GetOrCreateByYahooKey(ctx context.Context, yahooKey string, data map[string]interface{}) (*Player, error)
func (r *PlayerRepository) BulkUpsert(ctx context.Context, players []*Player) error
func (r *PlayerRepository) GetByKeys(ctx context.Context, yahooKeys []string) ([]*Player, error)
func (r *PlayerRepository) UpdateStatus(ctx context.Context, playerID int, isActive bool) error
```

**Key Design Decision**: `GetOrCreateByYahooKey` pattern reduces complexity. Yahoo API returns player data in many endpoints; this method ensures we don't duplicate players while avoiding excessive DB lookups.

#### PlayerStatsRepository
```go
// pkg/repository/player_stats_repository.go
type PlayerStatsRepository struct {
    db *sql.DB
}

type PlayerStatEntry struct {
    ID          int
    PlayerID    int
    LeagueID    int
    Week        int     // 0 for season
    StatType    string  // "actual", "projected"
    Stats       string  // JSON map of stat_id -> value
    Points      float64
    RecordedAt  time.Time
}

func (r *PlayerStatsRepository) SaveWeekStats(ctx context.Context, entry *PlayerStatEntry) error
func (r *PlayerStatsRepository) GetPlayerWeekStats(ctx context.Context, playerID int, leagueID int, week int) (*PlayerStatEntry, error)
func (r *PlayerStatsRepository) GetPlayerSeasonStats(ctx context.Context, playerID int, leagueID int) (*PlayerStatEntry, error)
func (r *PlayerStatsRepository) BulkSaveStats(ctx context.Context, entries []*PlayerStatEntry) error
```

**Schema Design**: Store stats as JSON instead of individual columns
- **Pro**: Schema doesn't break when sports add new stats (NBA added player efficiency rating mid-season in 2023)
- **Pro**: Simpler queries, no joins needed for multi-stat retrieval
- **Con**: Can't efficiently query by specific stat value (acceptable tradeoff - rare use case)

#### MatchupRepository
```go
// pkg/repository/matchup_repository.go
type MatchupRepository struct {
    db *sql.DB
}

type MatchupEntry struct {
    ID              int
    LeagueID        int
    Week            int
    Team1ID         int
    Team2ID         int
    Team1Points     float64
    Team2Points     float64
    Team1Projected  float64
    Team2Projected  float64
    WinnerTeamID    *int
    IsTied          bool
    IsPlayoffs      bool
    RecordedAt      time.Time
}

func (r *MatchupRepository) SaveWeekMatchups(ctx context.Context, leagueID int, week int, matchups []*MatchupEntry) error
func (r *MatchupRepository) GetLeagueWeekMatchups(ctx context.Context, leagueID int, week int) ([]*MatchupEntry, error)
func (r *MatchupRepository) GetTeamMatchupHistory(ctx context.Context, teamID int) ([]*MatchupEntry, error)
```

#### TransactionRepository
```go
// pkg/repository/transaction_repository.go
type TransactionRepository struct {
    db *sql.DB
}

type TransactionEntry struct {
    ID              int
    LeagueID        int
    YahooTxnKey     string
    TransactionType string
    Status          string
    Timestamp       time.Time
    FAAPBid         int
    Players         string  // JSON array of player movements
    CreatedAt       time.Time
}

func (r *TransactionRepository) SaveTransaction(ctx context.Context, txn *TransactionEntry) error
func (r *TransactionRepository) GetLeagueTransactions(ctx context.Context, leagueID int, limit int) ([]*TransactionEntry, error)
func (r *TransactionRepository) GetPlayerTransactionHistory(ctx context.Context, playerID int) ([]*TransactionEntry, error)
```

#### DraftRepository
```go
// pkg/repository/draft_repository.go
type DraftRepository struct {
    db *sql.DB
}

type DraftPick struct {
    ID          int
    LeagueID    int
    Pick        int
    Round       int
    TeamID      int
    PlayerID    int
    Cost        int
    PickedAt    time.Time
}

func (r *DraftRepository) SaveDraftResults(ctx context.Context, leagueID int, picks []*DraftPick) error
func (r *DraftRepository) GetLeagueDraft(ctx context.Context, leagueID int) ([]*DraftPick, error)
func (r *DraftRepository) GetTeamDraftPicks(ctx context.Context, teamID int) ([]*DraftPick, error)
```

### Repository Pattern: Simplicity Analysis

**Question**: Do we need repositories for every Yahoo API entity?

**Answer**: No. Repositories are for entities you persist. Ephemeral API response objects don't need repositories.

**Persist** (need repositories):
- Player (links to stats, rosters, drafts)
- PlayerStats (historical tracking, projections)
- Matchup (record tracking, predictions)
- Transaction (audit trail, pattern analysis)
- DraftPick (keeper values, draft strategy)

**Don't Persist** (no repositories needed):
- Week (derived from league settings)
- Standings (computed from matchup results)
- LeagueSettings (cached in API layer, rarely changes)

This approach avoids over-engineering while ensuring auditability of critical data.

---

## 4. Maintainability Impact Analysis

### Complexity Assessment

#### Current Codebase Metrics
- **Files**: 13 Go files
- **Packages**: 3 (yahoo, repository, service)
- **Public API Surface**: ~20 methods across client, repositories, services
- **External Dependencies**: 1 (database/sql)

#### Projected Metrics (After Full Implementation)
- **Files**: ~35 Go files (+169%)
- **Packages**: 3 (unchanged)
- **Public API Surface**: ~75 methods (+275%)
- **External Dependencies**: 1 (unchanged)

**Analysis**: The growth is substantial but manageable. The package structure constrains complexity growth - new entities live in existing packages with established patterns.

### Maintainability Risks

#### Risk 1: Sport-Specific Stat Handling

**Problem**: MLB/NFL/NBA/NHL have different stat categories. Naive implementation creates 4x the code.

**Bad Approach** (avoid):
```go
// DON'T DO THIS
type NBAPlayerStats struct { PTS, REB, AST float64 }
type NFLPlayerStats struct { PassYds, RushYds, TDs float64 }
type MLBPlayerStats struct { Hits, HR, RBI float64 }
// ... explosion of types
```

**Good Approach** (recommended):
```go
// Single stats structure with sport-agnostic storage
type PlayerStats struct {
    PlayerKey string
    Week      int
    Stats     map[string]StatValue  // Key is stat abbreviation
    Points    float64
}

// Sport-specific constants for type safety
const (
    NBA_PTS = "PTS"
    NBA_REB = "REB"
    NFL_PASS_YDS = "PassYds"
    // etc.
)

// Helper for stat access
func (ps *PlayerStats) GetStat(statID string) (float64, bool) {
    if v, ok := ps.Stats[statID]; ok {
        return v.Value, true
    }
    return 0, false
}
```

**Justification**:
- Eliminates code duplication across sports
- Yahoo API already uses stat IDs, we're just preserving their model
- Easy to add new sports without code changes
- Maintains type safety at access points

#### Risk 2: API Response Parsing Fragility

**Problem**: Yahoo API responses are inconsistent. Some endpoints return arrays, some return single objects, some wrap in `0` keys.

**Example of Yahoo's Inconsistency**:
```json
// Single league: wrapped in array with "0" key
{"leagues": [{"league": {...}}, {"0": {...}}]}

// Multiple leagues: just array
{"leagues": [{"league": {...}}, {"league": {...}}]}
```

**Mitigation Strategy**:
```go
// Defensive parsing with fallback logic
func (r *yahooLeaguesResponse) ToLeagues() []League {
    var leagues []League

    for _, item := range r.Fantasy_Content.Leagues {
        // Try normal league extraction
        if item.League.LeagueKey != "" {
            leagues = append(leagues, extractLeague(item.League))
            continue
        }

        // Fallback: sometimes Yahoo wraps in "0" key
        if item.ZeroKey.LeagueKey != "" {
            leagues = append(leagues, extractLeague(item.ZeroKey))
        }
    }

    return leagues
}
```

**Additional Safeguard**: Comprehensive test suite with real Yahoo API response fixtures
```go
// pkg/yahoo/client_test.go
func TestParseLeaguesResponse_SingleLeague(t *testing.T)
func TestParseLeaguesResponse_MultipleLeagues(t *testing.T)
func TestParseLeaguesResponse_EmptyResponse(t *testing.T)
func TestParseLeaguesResponse_MalformedData(t *testing.T)
```

#### Risk 3: Cache Invalidation Complexity

**Current State**: Simple TTL-based caching (24h leagues, 6h teams, 1h rosters)

**Future Problem**: With 10+ new API methods, determining correct TTL per endpoint becomes complex. Wrong TTLs cause:
- Too long: Stale data (missed roster changes, incorrect scores)
- Too short: Excessive API calls (rate limiting, slow performance)

**Recommended Approach**: Event-driven cache invalidation
```go
// pkg/yahoo/cache_strategy.go
type CacheStrategy struct {
    staticData    time.Duration  // 7 days (league settings, player info)
    dailyData     time.Duration  // 24 hours (season stats, draft results)
    hourlyData    time.Duration  // 1 hour (rosters, standings)
    liveData      time.Duration  // 5 minutes (matchup scores during games)
}

func (c *Client) determineTTL(endpoint string) time.Duration {
    switch {
    case strings.Contains(endpoint, "/settings"):
        return c.cacheStrategy.staticData
    case strings.Contains(endpoint, "/draft_results"):
        return c.cacheStrategy.dailyData
    case strings.Contains(endpoint, "/matchups"):
        if isGameDay() {
            return c.cacheStrategy.liveData
        }
        return c.cacheStrategy.hourlyData
    default:
        return c.cacheStrategy.hourlyData
    }
}
```

**Simplicity Note**: Start with coarse-grained TTLs. Add complexity only when users report staleness issues. Premature optimization here adds unnecessary code.

### Code Organization Recommendations

#### Current Structure
```
pkg/
  yahoo/client.go          (all API methods)
  repository/*.go          (persistence)
  service/*.go             (business logic)
```

#### Recommended Structure (Phase by Phase)

**Phase 1** (keep simple):
```
pkg/
  yahoo/
    client.go              (core client + auth)
    league.go              (league entities)
    team.go                (team entities)
    player.go              (player entities)
    stats.go               (stats entities)
    matchup.go             (matchup entities)
  repository/              (unchanged)
  service/                 (unchanged)
```

**Phase 2** (add only if yahoo/ exceeds 15 files):
```
pkg/
  yahoo/
    client.go
    entities/              (all entity types)
    responses/             (API response mappers)
  repository/              (unchanged)
  service/                 (unchanged)
```

**Justification**: Delay creating subdirectories until they're needed. Go's flat package structure encourages cohesion. Moving to subdirectories prematurely adds import overhead and navigation friction.

---

## 5. Phased Implementation Plan

### Phase 1: Core API Read Operations (Weeks 1-2)

**Goal**: Enable basic stat retrieval and weekly data access

**Tasks**:
1. Add Player entity and GetLeaguePlayers method
2. Add PlayerStats entity and GetPlayerStats method
3. Add Week/Matchup entities and GetWeekMatchups method
4. Add Standings entity and GetLeagueStandings method
5. Create PlayerRepository with GetOrCreateByYahooKey
6. Create PlayerStatsRepository with SaveWeekStats
7. Create MatchupRepository with SaveWeekMatchups

**Deliverable**: CLI tool to fetch and display current week matchups with player stats

**Testing**: Integration tests hitting Yahoo sandbox API

**Estimated LOC**: +800 lines

**Risk**: Low. These are straightforward GET operations with well-documented endpoints.

### Phase 2: Historical Data (Weeks 3-4)

**Goal**: Enable draft and transaction tracking

**Tasks**:
1. Add DraftResult entity and GetLeagueDraftResults method
2. Add Transaction entity and GetLeagueTransactions method
3. Add GetPastLeagueID for historical league linking
4. Create DraftRepository with SaveDraftResults
5. Create TransactionRepository with SaveTransaction
6. Extend cache strategy for historical data (longer TTLs)

**Deliverable**: CLI tools to export draft results and transaction history

**Testing**: Unit tests with mock responses (historical data harder to test live)

**Estimated LOC**: +600 lines

**Risk**: Medium. Transaction data structure is complex with many edge cases.

### Phase 3: Enhanced Queries (Week 5)

**Goal**: Optimize performance and add advanced features

**Tasks**:
1. Add GetLeagueSettings and GetLeagueStatCategories methods
2. Add GetTeamRosterWithStats (combines roster + stats in one call)
3. Add GetMultiplePlayerStats (batch operation)
4. Implement sport-specific stat category constants
5. Add response caching for settings/static data
6. Optimize repository batch operations

**Deliverable**: Service layer methods using enhanced queries for trade analysis

**Testing**: Performance benchmarks comparing batch vs individual calls

**Estimated LOC**: +400 lines

**Risk**: Low. Building on stable foundation from Phases 1-2.

### Phase 4: Service Layer Integration (Week 6)

**Goal**: Wire new data into existing services

**Tasks**:
1. Update ValuationService to use PlayerStatsRepository
2. Update TradeService to use weekly matchup data
3. Update AnalysisService to incorporate transaction history
4. Add PlayerService for player search/filtering
5. Add MatchupService for weekly predictions

**Deliverable**: Complete API parity with Python yahoofantasy functionality

**Testing**: End-to-end service tests with real league data

**Estimated LOC**: +500 lines

**Risk**: Low. Services already have patterns established.

### Phase 5: CLI Tools (Week 7)

**Goal**: Provide user-facing tools for data export

**Tasks**:
1. CLI command: export-stats (weekly player performances)
2. CLI command: export-matchups (weekly matchup history)
3. CLI command: export-draft (draft results)
4. CLI command: export-transactions (transaction log)
5. CLI command: sync-league (full data sync)

**Deliverable**: CLI matching Python yahoofantasy export capabilities

**Testing**: CLI integration tests with temporary database

**Estimated LOC**: +300 lines

**Risk**: Low. Pure IO operations, minimal logic.

### Total Estimated Effort

- **Duration**: 7 weeks
- **Lines of Code**: ~2,600 new lines
- **Files Added**: ~22 files
- **Breaking Changes**: None (all additive)

---

## 6. Sport-Specific Stat Definitions Strategy

### Problem Statement

Each sport has unique stats:
- **NBA**: PTS, REB, AST, STL, BLK, TO, FG%, FT%, 3PM
- **NFL**: PassYds, PassTD, RushYds, RushTD, Rec, RecYds, RecTD, INT, Fum
- **MLB**: AB, R, HR, RBI, SB, AVG, W, SV, K, ERA, WHIP
- **NHL**: G, A, +/-, PPP, SOG, W, GAA, SV%, SHO

Naive implementation creates 4 separate stat systems with no code reuse.

### Recommended Architecture: Registry Pattern

```go
// pkg/yahoo/stats/registry.go
package stats

type StatDefinition struct {
    ID          int
    Sport       string
    Abbr        string
    Name        string
    DisplayName string
    PositionType string  // "skater", "goalie", "batter", "pitcher", etc.
    SortOrder   int
    IsPercent   bool
    IsRatio     bool
}

var Registry = map[string]map[string]StatDefinition{
    "nba": {
        "PTS": {ID: 12, Sport: "nba", Abbr: "PTS", Name: "Points", DisplayName: "Points", PositionType: "player"},
        "REB": {ID: 15, Sport: "nba", Abbr: "REB", Name: "Rebounds", DisplayName: "Total Rebounds", PositionType: "player"},
        // ... all NBA stats
    },
    "nfl": {
        "PassYds": {ID: 4, Sport: "nfl", Abbr: "PassYds", Name: "Passing Yards", DisplayName: "Passing Yards", PositionType: "QB"},
        // ... all NFL stats
    },
    // ... MLB, NHL
}

func GetStatDefinition(sport, abbr string) (StatDefinition, bool) {
    if sportStats, ok := Registry[sport]; ok {
        if stat, ok := sportStats[abbr]; ok {
            return stat, true
        }
    }
    return StatDefinition{}, false
}

func GetSportStats(sport string) []StatDefinition {
    if sportStats, ok := Registry[sport]; ok {
        var stats []StatDefinition
        for _, stat := range sportStats {
            stats = append(stats, stat)
        }
        return stats
    }
    return nil
}
```

### Usage in PlayerStats

```go
// pkg/yahoo/player_stats.go
type PlayerStats struct {
    PlayerKey string
    Sport     string
    Week      int
    Stats     map[string]float64  // abbr -> value
    Points    float64
}

func (ps *PlayerStats) GetStat(abbr string) (float64, bool) {
    val, ok := ps.Stats[abbr]
    return val, ok
}

func (ps *PlayerStats) GetStatWithDefinition(abbr string) (float64, StatDefinition, bool) {
    val, ok := ps.Stats[abbr]
    if !ok {
        return 0, StatDefinition{}, false
    }

    def, defOK := stats.GetStatDefinition(ps.Sport, abbr)
    return val, def, defOK
}

func (ps *PlayerStats) FormatStat(abbr string) string {
    val, def, ok := ps.GetStatWithDefinition(abbr)
    if !ok {
        return "N/A"
    }

    if def.IsPercent {
        return fmt.Sprintf("%.1f%%", val * 100)
    }
    if def.IsRatio {
        return fmt.Sprintf("%.2f", val)
    }
    return fmt.Sprintf("%.0f", val)
}
```

### Populating the Registry

**Option A**: Hardcode all stats (Python approach)
```go
// Maintainable but requires updates when Yahoo adds stats
var nbaStats = map[string]StatDefinition{
    "PTS": {ID: 12, ...},
    // ... 50+ stats
}
```

**Option B**: Fetch from Yahoo API on startup
```go
// Client method
func (c *Client) GetLeagueStatCategories(ctx context.Context, leagueKey string) ([]StatCategory, error) {
    // Yahoo endpoint: /league/{key}/settings
    // Parse stat_categories section
}

// Initialize registry
func InitRegistry(ctx context.Context, client *Client, gameKeys []string) error {
    for _, gameKey := range gameKeys {
        leagueKey := fmt.Sprintf("%s.l.12345", gameKey)  // dummy league
        stats, err := client.GetLeagueStatCategories(ctx, leagueKey)
        if err != nil {
            continue
        }

        sport := strings.Split(gameKey, ".")[0]
        Registry[sport] = make(map[string]StatDefinition)
        for _, stat := range stats {
            Registry[sport][stat.Abbr] = StatDefinition{
                ID: stat.StatID,
                Sport: sport,
                Abbr: stat.Abbr,
                Name: stat.Name,
                // ...
            }
        }
    }
    return nil
}
```

**Recommendation**: Start with Option A (hardcoded), migrate to Option B in Phase 3.

**Justification**:
- Option A is simpler and has zero runtime dependencies
- Yahoo rarely changes stat definitions (happens ~once per season)
- Option B requires hitting Yahoo API for metadata, adds complexity
- Hybrid approach: hardcode common stats, fetch from API to fill gaps

### Schema Design for Sport-Specific Stats

**Don't create sport-specific tables**:
```sql
-- AVOID THIS
CREATE TABLE nba_player_stats (...);
CREATE TABLE nfl_player_stats (...);
CREATE TABLE mlb_player_stats (...);
```

**Use generic stat storage**:
```sql
CREATE TABLE player_stats (
    id INTEGER PRIMARY KEY,
    player_id INTEGER NOT NULL,
    league_id INTEGER NOT NULL,
    week INTEGER NOT NULL,  -- 0 for season
    stat_type TEXT NOT NULL,  -- 'actual', 'projected'
    stats_json TEXT NOT NULL,  -- JSON map: {"PTS": 25.0, "REB": 8.0}
    fantasy_points REAL,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(player_id, league_id, week, stat_type)
);

CREATE INDEX idx_player_stats_lookup ON player_stats(player_id, league_id, week);
```

**Benefits**:
- Single query to fetch stats regardless of sport
- No schema migrations when Yahoo adds new stats
- Easy to extend to new sports
- JSON queries supported in modern SQLite (json_extract)

**Example Query**:
```go
// Fetch NBA player's points for week 5
query := `
    SELECT json_extract(stats_json, '$.PTS') as points
    FROM player_stats
    WHERE player_id = ? AND league_id = ? AND week = 5
`
```

---

## 7. Critical Design Decisions

### Decision 1: Separate Yahoo API Entities from Persistence Entities

**Rationale**: Yahoo's `Player` object is different from your database `players` table.

```go
// pkg/yahoo/player.go - API entity (ephemeral)
type Player struct {
    PlayerKey       string
    FullName        string
    Status          PlayerStatus
    EligiblePositions []string
}

// pkg/repository/player_repository.go - Persistence entity
type Player struct {
    ID              int
    YahooPlayerKey  string
    FullName        string
    IsActive        bool
}
```

**Mapping happens in service layer**:
```go
// pkg/service/player_service.go
func (s *PlayerService) SyncPlayers(ctx context.Context, yahooPlayers []yahoo.Player) error {
    for _, yp := range yahooPlayers {
        dbPlayer := &repository.Player{
            YahooPlayerKey: yp.PlayerKey,
            FullName:       yp.FullName,
            IsActive:       yp.Status != yahoo.PlayerStatusUnavailable,
        }
        s.playerRepo.GetOrCreate(ctx, dbPlayer)
    }
}
```

**Why This Matters**:
- Yahoo API changes don't force database migrations
- Can store derived/computed fields in DB without polluting API layer
- Easier to test (mock Yahoo responses vs DB separately)

### Decision 2: Cache at Client Layer, Not Repository Layer

**Current implementation**: Cache in `Client.GetUserLeagues` before calling `fetchLeagues`

**Keep this pattern**. Don't move caching to repositories.

**Why**:
- Repositories represent source of truth (database)
- Caching is an optimization concern for expensive operations (API calls)
- Easier to disable caching for testing (mock client, not repositories)

### Decision 3: Avoid ORM, Stick with database/sql

**Status Quo**: Direct SQL queries with `database/sql`

**Recommendation**: Keep it. Don't introduce GORM, SQLX, or other ORMs.

**Justification**:
- Your queries are simple (no complex joins, mostly key lookups)
- ORMs add dependency weight and learning curve
- ORMs obscure query performance (N+1 problems harder to spot)
- database/sql is boring technology (battle-tested, well-documented)

**When to reconsider**: If you exceed 50+ repository methods with significant query duplication. Currently at ~15 methods, not a concern.

### Decision 4: Error Handling Strategy

**Current approach**: Wrap errors with context
```go
return fmt.Errorf("failed to fetch teams: %w", err)
```

**Recommendation**: Add error types for actionable errors

```go
// pkg/yahoo/errors.go
type APIError struct {
    StatusCode int
    Message    string
    Endpoint   string
}

func (e *APIError) Error() string {
    return fmt.Sprintf("Yahoo API error %d at %s: %s", e.StatusCode, e.Endpoint, e.Message)
}

type RateLimitError struct {
    RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
    return fmt.Sprintf("rate limited, retry after %v", e.RetryAfter)
}

// Client method
func (c *Client) makeRequest(ctx, endpoint) ([]byte, error) {
    // ...
    if resp.StatusCode == 429 {
        retryAfter := parseRetryAfter(resp.Header)
        return nil, &RateLimitError{RetryAfter: retryAfter}
    }
    if resp.StatusCode >= 400 {
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Message: string(body),
            Endpoint: endpoint,
        }
    }
    // ...
}
```

**Benefit**: Callers can handle rate limits intelligently (exponential backoff) vs generic errors.

### Decision 5: Testing Strategy

**Current state**: Service layer has unit tests, client has none

**Recommended approach**:

1. **Client layer**: Integration tests with real Yahoo sandbox API
   - Pro: Catches API changes immediately
   - Con: Requires valid test credentials
   - Frequency: Run in CI, not on every commit

2. **Service layer**: Unit tests with mock client
   - Pro: Fast, deterministic
   - Con: Can drift from real API behavior
   - Frequency: Run on every commit

3. **Repository layer**: Integration tests with in-memory SQLite
   - Pro: Tests real SQL without external dependencies
   - Con: None (SQLite is lightweight)
   - Frequency: Run on every commit

**Don't mock database**. SQLite is fast enough to use in tests directly.

```go
// Test helper
func setupTestDB(t *testing.T) *sql.DB {
    db, err := sql.Open("sqlite3", ":memory:")
    require.NoError(t, err)

    // Run migrations
    _, err = db.Exec(schema)
    require.NoError(t, err)

    return db
}
```

---

## 8. Migration Path for Existing Code

### Breaking Changes: None Required

All new functionality is additive. Existing code using `GetUserLeagues`, `GetLeagueTeams`, `GetTeamRoster` continues working unchanged.

### Deprecation Strategy

**Current**: `GetTeamRoster` returns minimal roster data without stats

**Phase 3**: Introduce `GetTeamRosterWithStats` with full player data

**Don't deprecate `GetTeamRoster`**. Keep both methods.

**Justification**:
- Different use cases (roster positions vs player performance)
- `GetTeamRoster` is simpler/faster for non-stat queries
- No maintenance burden (one endpoint, two mappers)

### Service Layer Evolution

**Current services** (trade, valuation, analysis) use hardcoded data:
```go
// service/league_service.go
scoringSettings := map[string]float64{
    "PTS": 1.0,
    // ... hardcoded
}
```

**Phase 1 migration**: Fetch from league settings
```go
settings, err := s.yahooClient.GetLeagueSettings(ctx, leagueKey)
scoringSettings := settings.StatModifiers
```

**Migration approach**: Add new code path, keep old as fallback
```go
scoringSettings, err := s.getLeagueScoring(ctx, leagueID)
if err != nil {
    // Fallback to defaults
    scoringSettings = defaultNBAScoring
}
```

Gradual migration reduces risk of breaking existing functionality.

---

## 9. Open Questions and Recommendations

### Question 1: Multi-Sport Support Priority

**Current implementation**: Hardcoded to NBA (`gameKey: "nba"`)

**Recommendation**: Delay generalization until Phase 3

**Rationale**:
- Get NBA working perfectly first
- Sport abstraction easier to design with concrete examples
- Most users use single sport, multi-sport is edge case
- Premature abstraction adds complexity without proven benefit

**When to generalize**: After Phase 2 complete, before Phase 4

### Question 2: Real-Time Data Updates

**Python yahoofantasy**: Polling-based (user calls sync methods)

**Your implementation**: Same approach (manual sync via service methods)

**Alternative**: Background sync worker

```go
// Background sync every 5 minutes during game days
type SyncWorker struct {
    client  *yahoo.Client
    service *service.LeagueService
}

func (w *SyncWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Minute)
    for {
        select {
        case <-ticker.C:
            w.syncAllLeagues(ctx)
        case <-ctx.Done():
            return
        }
    }
}
```

**Recommendation**: Don't implement this yet.

**Justification**:
- Adds complexity (worker lifecycle, error handling)
- Yahoo API has rate limits (background sync may trigger limits)
- User-initiated sync gives better control
- Can add later without breaking changes

**When to add**: If users request it and have rate limit budget

### Question 3: GraphQL vs REST for Internal API

**Current**: No internal API (direct service calls)

**Future**: May want API for web UI or mobile app

**Recommendation**: Start with simple REST endpoints when needed

```go
// cmd/api/main.go
http.HandleFunc("/api/leagues", handlers.GetLeagues)
http.HandleFunc("/api/teams/{id}/roster", handlers.GetRoster)
```

**Don't use GraphQL**. Your domain is simple CRUD, GraphQL overhead not justified.

**When to reconsider**: If client needs highly dynamic queries (unlikely for fantasy sports)

### Question 4: Deployment Strategy

**Current**: No deployment artifacts (library code)

**Future**: CLI tools and potential API server

**Recommendation**:

1. **Phase 5**: Build CLI as single binary
   ```bash
   go build -o yahoo-fantasy-cli cmd/cli/main.go
   ```

2. **Distribution**: GitHub releases with platform-specific binaries
   - macOS: `yahoo-fantasy-cli-darwin-amd64`
   - Linux: `yahoo-fantasy-cli-linux-amd64`
   - Windows: `yahoo-fantasy-cli-windows-amd64.exe`

3. **Installation**: Homebrew tap (macOS), apt repo (Linux)
   ```bash
   brew tap n-ae/yahoo-fantasy
   brew install yahoo-fantasy-cli
   ```

**Don't containerize yet**. CLI doesn't need Docker, adds complexity.

**When to containerize**: If you build API server that needs deployment to cloud

---

## 10. Success Metrics

### Functional Parity

- [ ] All Python yahoofantasy API methods have Go equivalents
- [ ] CLI tools match Python export capabilities
- [ ] Multi-sport support for NBA, NFL, MLB, NHL
- [ ] Transaction and draft history tracking

### Performance Targets

- **API calls**: <500ms per request (95th percentile)
- **Database queries**: <50ms per query (95th percentile)
- **Cache hit rate**: >80% for frequently accessed data
- **Rate limit adherence**: <1000 requests per hour per user

### Code Quality

- **Test coverage**: >80% for client and repository layers
- **Documentation**: GoDoc for all public APIs
- **Linting**: Zero warnings from golangci-lint
- **Cyclomatic complexity**: <15 per function

### Maintainability

- **No circular dependencies**: Enforce with `go mod graph | grep cycle`
- **Package cohesion**: <200 lines per file (except test files)
- **Dependency count**: <5 external dependencies
- **API stability**: No breaking changes for 1.x versions

---

## Conclusion

The Go implementation has a solid architectural foundation. The primary gap is breadth of API coverage, not design quality. The phased implementation plan prioritizes core functionality first, avoids premature optimization, and maintains the simplicity-first philosophy.

**Key Takeaways**:

1. **Add 15 new API methods** across 5 phases (7 weeks)
2. **Introduce 8 new entity types** (Player, PlayerStats, Week, Matchup, DraftResult, Transaction, Standings, LeagueSettings)
3. **Create 5 new repositories** (Player, PlayerStats, Matchup, Transaction, Draft)
4. **Use map-based stat storage** to avoid sport-specific code explosion
5. **Keep separation** between API entities (ephemeral) and persistence entities (database)
6. **Defer abstractions** until proven necessary (multi-sport, background sync, GraphQL)

**Biggest Risk**: Transaction parsing complexity. Mitigation: comprehensive test suite with real response fixtures.

**Biggest Win**: Generic stat storage eliminates 4x code duplication across sports.

This approach delivers Python yahoofantasy parity while maintaining Go idioms and the existing architecture's strengths.

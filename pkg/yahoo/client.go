package yahoo

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Client struct {
	apiKey       string
	apiSecret    string
	accessToken  string
	refreshToken string
	httpClient   HTTPDoer
	baseURL      string
	tokenURL     string
	cache        Cache
	logger       Logger
	tokenStore   TokenStore
	retry        RetryPolicy
	tokenMu      sync.RWMutex
	cacheEnabled bool
}

// currentAccessToken returns the access token under a read lock so reads do not
// race with a concurrent refresh.
func (c *Client) currentAccessToken() string {
	c.tokenMu.RLock()
	defer c.tokenMu.RUnlock()
	return c.accessToken
}

type apiCache struct {
	db *sql.DB
}

type League struct {
	YahooLeagueID string
	YahooGameKey  string
	LeagueName    string
	SeasonYear    int
	ScoringType   string
	NumTeams      int
	CurrentWeek   int
}

type Team struct {
	YahooTeamID  string
	YahooTeamKey string
	TeamName     string
	ManagerName  string
	Wins         int
	Losses       int
	Ties         int
	Rank         int
}

// SlotState classifies the lineup slot a player currently occupies.
type SlotState string

const (
	SlotStarting SlotState = "starting"
	SlotBench    SlotState = "bench"
	SlotInjured  SlotState = "injured"
)

type Roster struct {
	TeamID            string
	PlayerID          string
	PlayerKey         string
	EligiblePositions []string
	SelectedPos       string
	SlotState         SlotState
	// IsStarting reports whether the player occupies an active lineup slot
	// (i.e. not bench, injured, or inactive).
	IsStarting bool
}

// classifySlot maps a Yahoo selected_position value to a coarse lineup state.
// "BN" is the bench; injured/inactive slots include IL*, IR*, and NA; anything
// else is treated as an active starting slot.
func classifySlot(pos string) SlotState {
	switch {
	case pos == "BN":
		return SlotBench
	case pos == "NA" || strings.HasPrefix(pos, "IL") || strings.HasPrefix(pos, "IR"):
		return SlotInjured
	default:
		return SlotStarting
	}
}

type yahooLeaguesResponse struct {
	Fantasy_Content struct {
		Users []struct {
			User []struct {
				Games []struct {
					Game []struct {
						Leagues []struct {
							League struct {
								League_Key   string `json:"league_key"`
								League_ID    string `json:"league_id"`
								Name         string `json:"name"`
								Season       string `json:"season"`
								Scoring_Type string `json:"scoring_type"`
								Num_Teams    int    `json:"num_teams"`
								Current_Week int    `json:"current_week"`
							} `json:"league"`
						} `json:"leagues"`
					} `json:"game"`
				} `json:"games"`
			} `json:"user"`
		} `json:"users"`
	} `json:"fantasy_content"`
}

type yahooTeamsResponse struct {
	Fantasy_Content struct {
		League struct {
			Teams []struct {
				Team struct {
					Team_Key string `json:"team_key"`
					Team_ID  string `json:"team_id"`
					Name     string `json:"name"`
					Managers []struct {
						Manager struct {
							Nickname string `json:"nickname"`
						} `json:"manager"`
					} `json:"managers"`
					Team_Standings struct {
						Rank           int `json:"rank"`
						Outcome_Totals struct {
							Wins   int `json:"wins"`
							Losses int `json:"losses"`
							Ties   int `json:"ties"`
						} `json:"outcome_totals"`
					} `json:"team_standings"`
				} `json:"team"`
			} `json:"teams"`
		} `json:"league"`
	} `json:"fantasy_content"`
}

type yahooRosterResponse struct {
	Fantasy_Content struct {
		Team struct {
			Roster struct {
				Players []struct {
					Player struct {
						Player_Key         string `json:"player_key"`
						Player_ID          string `json:"player_id"`
						Eligible_Positions []struct {
							Position string `json:"position"`
						} `json:"eligible_positions"`
						Selected_Position struct {
							Position string `json:"position"`
						} `json:"selected_position"`
					} `json:"player"`
				} `json:"players"`
			} `json:"roster"`
		} `json:"team"`
	} `json:"fantasy_content"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

// maxResponseBody caps how many bytes are read from a Yahoo response into
// memory, preventing an unbounded or hostile body from exhausting memory.
const maxResponseBody = 10 << 20 // 10 MiB

// APIError is returned when Yahoo responds with a non-2xx status. It exposes
// the status code, endpoint, and (bounded) response body so callers can branch
// on failures with errors.As rather than string matching.
type APIError struct {
	StatusCode int
	Endpoint   string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Yahoo API error (status %d) for %q: %s", e.StatusCode, e.Endpoint, e.Body)
}

// readLimited reads r up to maxResponseBody bytes.
func readLimited(r io.Reader) ([]byte, error) {
	return io.ReadAll(io.LimitReader(r, maxResponseBody))
}

func (c *Client) GetUserLeagues(ctx context.Context, gameKey string) ([]League, error) {
	cacheKey := fmt.Sprintf("user:leagues:%s", gameKey)

	if v, ok := cacheGet[[]League](ctx, c, cacheKey); ok {
		return v, nil
	}

	leagues, err := c.fetchLeagues(ctx, gameKey)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, leagues, 24*time.Hour)
	return leagues, nil
}

func (c *Client) GetLeagueTeams(ctx context.Context, leagueKey string) ([]Team, error) {
	cacheKey := fmt.Sprintf("league:%s:teams", leagueKey)

	if v, ok := cacheGet[[]Team](ctx, c, cacheKey); ok {
		return v, nil
	}

	teams, err := c.fetchTeams(ctx, leagueKey)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, teams, 6*time.Hour)
	return teams, nil
}

func (c *Client) GetTeamRoster(ctx context.Context, teamKey string) ([]Roster, error) {
	cacheKey := fmt.Sprintf("team:%s:roster", teamKey)

	if v, ok := cacheGet[[]Roster](ctx, c, cacheKey); ok {
		return v, nil
	}

	roster, err := c.fetchRoster(ctx, teamKey)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, roster, 1*time.Hour)
	return roster, nil
}

// refreshIfStale exchanges the refresh token for a new access token, but only
// if the access token still matches usedToken — the value the caller sent on
// the request that got a 401. This single-flights concurrent refreshes: when N
// requests expire at once, the first refreshes and the rest observe a changed
// token and return without hitting the token endpoint again. The refresh HTTP
// call honours ctx.
func (c *Client) refreshIfStale(ctx context.Context, usedToken string) error {
	tok, refreshed, err := c.doRefresh(ctx, usedToken)
	if err != nil || !refreshed {
		return err
	}

	// Persist the rotated tokens outside the lock. Save is advisory: a failure
	// is logged but does not fail the request, since tok is valid in memory.
	if c.tokenStore != nil {
		if serr := c.tokenStore.Save(ctx, tok); serr != nil {
			c.logger.Printf("yahoo: token persistence failed: %v", serr)
		}
	}
	return nil
}

// doRefresh performs the locked, single-flight token exchange. It returns the
// rotated token and refreshed=true only when this call actually refreshed;
// refreshed=false means another goroutine already did (no persistence needed).
func (c *Client) doRefresh(ctx context.Context, usedToken string) (Token, bool, error) {
	c.tokenMu.Lock()
	defer c.tokenMu.Unlock()

	// Another goroutine already refreshed while we waited for the lock.
	if c.accessToken != usedToken {
		return Token{}, false, nil
	}

	if c.refreshToken == "" {
		return Token{}, false, fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", c.tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return Token{}, false, fmt.Errorf("failed to create token request: %w", err)
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(c.apiKey + ":" + c.apiSecret))
	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Token{}, false, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := readLimited(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return Token{}, false, fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return Token{}, false, fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.refreshToken = tokenResp.RefreshToken
	}

	tok := Token{
		AccessToken:  c.accessToken,
		RefreshToken: c.refreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}
	return tok, true, nil
}

func (c *Client) makeRequest(ctx context.Context, endpoint string) ([]byte, error) {
	token := c.currentAccessToken()
	if token == "" {
		return nil, fmt.Errorf("Yahoo access token not configured - set YAHOO_ACCESS_TOKEN environment variable")
	}

	reqURL := fmt.Sprintf("%s/%s?format=json", c.baseURL, endpoint)
	refreshed := false

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}

		// Expired access token: refresh once and retry immediately. This is not
		// counted as a transient retry and does not consume the retry budget.
		if resp.StatusCode == http.StatusUnauthorized && !refreshed {
			body, _ := readLimited(resp.Body)
			resp.Body.Close()
			if strings.Contains(string(body), "token_expired") {
				if err := c.refreshIfStale(ctx, token); err != nil {
					return nil, fmt.Errorf("failed to refresh expired token: %w", err)
				}
				token = c.currentAccessToken()
				refreshed = true
				continue
			}
			return nil, &APIError{StatusCode: resp.StatusCode, Endpoint: endpoint, Body: string(body)}
		}

		// Rate limiting and transient server errors: bounded retry with backoff,
		// honouring Retry-After on 429.
		if isRetryableStatus(resp.StatusCode) && attempt < c.retry.MaxRetries {
			delay := backoffDelay(c.retry, attempt)
			if resp.StatusCode == http.StatusTooManyRequests {
				if ra, ok := retryAfterDelay(resp.Header.Get("Retry-After")); ok {
					delay = ra
				}
			}
			status := resp.StatusCode
			resp.Body.Close()
			c.logger.Printf("yahoo: %s returned %d, retrying in %s (attempt %d/%d)", endpoint, status, delay, attempt+1, c.retry.MaxRetries)
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := readLimited(resp.Body)
			resp.Body.Close()
			return nil, &APIError{StatusCode: resp.StatusCode, Endpoint: endpoint, Body: string(body)}
		}

		body, err := readLimited(resp.Body)
		resp.Body.Close()
		return body, err
	}
}

func (c *Client) fetchLeagues(ctx context.Context, gameKey string) ([]League, error) {
	endpoint := fmt.Sprintf("users;use_login=1/games;game_keys=%s/leagues", gameKey)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooLeaguesResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse leagues response: %w", err)
	}

	var leagues []League
	for _, user := range resp.Fantasy_Content.Users {
		for _, userItem := range user.User {
			for _, game := range userItem.Games {
				for _, gameItem := range game.Game {
					for _, leagueItem := range gameItem.Leagues {
						l := leagueItem.League
						var season int
						fmt.Sscanf(l.Season, "%d", &season)
						leagues = append(leagues, League{
							YahooLeagueID: l.League_ID,
							YahooGameKey:  gameKey,
							LeagueName:    l.Name,
							SeasonYear:    season,
							ScoringType:   l.Scoring_Type,
							NumTeams:      l.Num_Teams,
							CurrentWeek:   l.Current_Week,
						})
					}
				}
			}
		}
	}

	return leagues, nil
}

func (c *Client) fetchTeams(ctx context.Context, leagueKey string) ([]Team, error) {
	endpoint := fmt.Sprintf("league/%s/teams", leagueKey)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooTeamsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse teams response: %w", err)
	}

	var teams []Team
	for _, teamItem := range resp.Fantasy_Content.League.Teams {
		t := teamItem.Team
		managerName := ""
		if len(t.Managers) > 0 {
			managerName = t.Managers[0].Manager.Nickname
		}
		teams = append(teams, Team{
			YahooTeamID:  t.Team_ID,
			YahooTeamKey: t.Team_Key,
			TeamName:     t.Name,
			ManagerName:  managerName,
			Wins:         t.Team_Standings.Outcome_Totals.Wins,
			Losses:       t.Team_Standings.Outcome_Totals.Losses,
			Ties:         t.Team_Standings.Outcome_Totals.Ties,
			Rank:         t.Team_Standings.Rank,
		})
	}

	return teams, nil
}

func (c *Client) fetchRoster(ctx context.Context, teamKey string) ([]Roster, error) {
	endpoint := fmt.Sprintf("team/%s/roster", teamKey)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooRosterResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse roster response: %w", err)
	}

	var roster []Roster
	for _, playerItem := range resp.Fantasy_Content.Team.Roster.Players {
		p := playerItem.Player
		eligible := make([]string, 0, len(p.Eligible_Positions))
		for _, ep := range p.Eligible_Positions {
			eligible = append(eligible, ep.Position)
		}
		slot := classifySlot(p.Selected_Position.Position)
		roster = append(roster, Roster{
			PlayerID:          p.Player_ID,
			PlayerKey:         p.Player_Key,
			EligiblePositions: eligible,
			SelectedPos:       p.Selected_Position.Position,
			SlotState:         slot,
			IsStarting:        slot == SlotStarting,
		})
	}

	return roster, nil
}

func (c *apiCache) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if c == nil || c.db == nil {
		return nil, false, fmt.Errorf("cache not configured: no database")
	}

	var value []byte
	var expiresAt time.Time

	query := `SELECT cache_value, expires_at FROM yahoo_api_cache WHERE cache_key = ?`
	err := c.db.QueryRowContext(ctx, query, key).Scan(&value, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	if time.Now().After(expiresAt) {
		_, _ = c.db.ExecContext(ctx, `DELETE FROM yahoo_api_cache WHERE cache_key = ?`, key)
		return nil, false, nil
	}

	return value, true, nil
}

func (c *apiCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if c == nil || c.db == nil {
		return fmt.Errorf("cache not configured: no database")
	}

	expiresAt := time.Now().Add(ttl)
	query := `INSERT OR REPLACE INTO yahoo_api_cache (cache_key, cache_value, expires_at) VALUES (?, ?, ?)`
	_, err := c.db.ExecContext(ctx, query, key, value, expiresAt)
	return err
}

// cacheGet returns a cached, JSON-decoded value of type T, or ok=false on a
// cache miss, disabled cache, or any decode/store error (caching is advisory).
func cacheGet[T any](ctx context.Context, c *Client, key string) (T, bool) {
	var zero T
	if !c.cacheEnabled {
		return zero, false
	}
	b, ok, err := c.cache.Get(ctx, key)
	if err != nil || !ok {
		return zero, false
	}
	var v T
	if json.Unmarshal(b, &v) != nil {
		return zero, false
	}
	return v, true
}

// cacheSet JSON-encodes v and stores it under key. Errors are ignored: caching
// is best-effort and never affects correctness.
func cacheSet(ctx context.Context, c *Client, key string, v any, ttl time.Duration) {
	if !c.cacheEnabled {
		return
	}
	if b, err := json.Marshal(v); err == nil {
		_ = c.cache.Set(ctx, key, b, ttl)
	}
}

func (c *Client) GetLeaguePlayers(ctx context.Context, leagueKey string, status PlayerStatus, start, count int) ([]Player, error) {
	cacheKey := fmt.Sprintf("league:%s:players:%s:%d:%d", leagueKey, status, start, count)

	if v, ok := cacheGet[[]Player](ctx, c, cacheKey); ok {
		return v, nil
	}

	players, err := c.fetchLeaguePlayers(ctx, leagueKey, status, start, count)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, players, 1*time.Hour)
	return players, nil
}

func (c *Client) GetPlayerStats(ctx context.Context, leagueKey, playerKey string, weekNum int) (*Player, error) {
	weekStr := "season"
	if weekNum > 0 {
		weekStr = fmt.Sprintf("week_%d", weekNum)
	}
	cacheKey := fmt.Sprintf("player:%s:stats:%s:%s", playerKey, leagueKey, weekStr)

	if v, ok := cacheGet[Player](ctx, c, cacheKey); ok {
		return &v, nil
	}

	player, err := c.fetchPlayerStats(ctx, leagueKey, playerKey, weekNum)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, player, 2*time.Hour)
	return player, nil
}

func (c *Client) GetLeagueStandings(ctx context.Context, leagueKey string) (*Standings, error) {
	cacheKey := fmt.Sprintf("league:%s:standings", leagueKey)

	if v, ok := cacheGet[Standings](ctx, c, cacheKey); ok {
		return &v, nil
	}

	standings, err := c.fetchStandings(ctx, leagueKey)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, standings, 6*time.Hour)
	return standings, nil
}

func (c *Client) GetLeagueMatchups(ctx context.Context, leagueKey string, weekNum int) ([]Matchup, error) {
	cacheKey := fmt.Sprintf("league:%s:matchups:week_%d", leagueKey, weekNum)

	if v, ok := cacheGet[[]Matchup](ctx, c, cacheKey); ok {
		return v, nil
	}

	matchups, err := c.fetchMatchups(ctx, leagueKey, weekNum)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, matchups, 1*time.Hour)
	return matchups, nil
}

// GetLeagueDraftResults returns draft results using Yahoo's default paging.
// For explicit pagination use GetLeagueDraftResultsPage.
func (c *Client) GetLeagueDraftResults(ctx context.Context, leagueKey string) ([]DraftResult, error) {
	return c.GetLeagueDraftResultsPage(ctx, leagueKey, PageOptions{})
}

// GetLeagueDraftResultsPage returns one page of draft results per the given
// PageOptions.
func (c *Client) GetLeagueDraftResultsPage(ctx context.Context, leagueKey string, page PageOptions) ([]DraftResult, error) {
	cacheKey := fmt.Sprintf("league:%s:draft_results:%d:%d", leagueKey, page.Start, page.Count)

	if v, ok := cacheGet[[]DraftResult](ctx, c, cacheKey); ok {
		return v, nil
	}

	results, err := c.fetchDraftResults(ctx, leagueKey, page)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, results, 24*time.Hour)
	return results, nil
}

// GetLeagueTransactions returns transactions using Yahoo's default paging. For
// explicit pagination use GetLeagueTransactionsPage.
func (c *Client) GetLeagueTransactions(ctx context.Context, leagueKey string) ([]Transaction, error) {
	return c.GetLeagueTransactionsPage(ctx, leagueKey, PageOptions{})
}

// GetLeagueTransactionsPage returns one page of transactions per the given
// PageOptions.
func (c *Client) GetLeagueTransactionsPage(ctx context.Context, leagueKey string, page PageOptions) ([]Transaction, error) {
	cacheKey := fmt.Sprintf("league:%s:transactions:%d:%d", leagueKey, page.Start, page.Count)

	if v, ok := cacheGet[[]Transaction](ctx, c, cacheKey); ok {
		return v, nil
	}

	transactions, err := c.fetchTransactions(ctx, leagueKey, page)
	if err != nil {
		return nil, err
	}

	cacheSet(ctx, c, cacheKey, transactions, 30*time.Minute)
	return transactions, nil
}

func (c *Client) fetchLeaguePlayers(ctx context.Context, leagueKey string, status PlayerStatus, start, count int) ([]Player, error) {
	statusParam := ""
	if status != "" {
		statusParam = fmt.Sprintf(";status=%s", status)
	}
	endpoint := fmt.Sprintf("league/%s/players%s;start=%d;count=%d", leagueKey, statusParam, start, count)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooPlayerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse players response: %w", err)
	}

	var players []Player
	for _, item := range resp.FantasyContent.League.Players {
		players = append(players, convertYahooPlayerToPlayer(item.Player))
	}

	return players, nil
}

func (c *Client) fetchPlayerStats(ctx context.Context, leagueKey, playerKey string, weekNum int) (*Player, error) {
	statsParam := ""
	if weekNum > 0 {
		statsParam = fmt.Sprintf(";type=week;week=%d", weekNum)
	}
	endpoint := fmt.Sprintf("league/%s/players;player_keys=%s/stats%s", leagueKey, playerKey, statsParam)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooSinglePlayerResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse player stats response: %w", err)
	}

	player := convertYahooPlayerToPlayer(resp.FantasyContent.League.Players.Player)
	return &player, nil
}

func (c *Client) fetchStandings(ctx context.Context, leagueKey string) (*Standings, error) {
	endpoint := fmt.Sprintf("league/%s/standings", leagueKey)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooStandingsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse standings response: %w", err)
	}

	var teams []StandingsTeam
	for _, item := range resp.FantasyContent.League.Standings.Teams {
		teams = append(teams, convertYahooStandingsTeam(item.Team))
	}

	return &Standings{Teams: teams}, nil
}

func (c *Client) fetchMatchups(ctx context.Context, leagueKey string, weekNum int) ([]Matchup, error) {
	endpoint := fmt.Sprintf("league/%s/scoreboard;week=%d", leagueKey, weekNum)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooScoreboardResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse scoreboard response: %w", err)
	}

	var matchups []Matchup
	for _, item := range resp.FantasyContent.League.Scoreboard.Matchups {
		matchups = append(matchups, convertYahooMatchup(item.Matchup))
	}

	return matchups, nil
}

func (c *Client) fetchDraftResults(ctx context.Context, leagueKey string, page PageOptions) ([]DraftResult, error) {
	endpoint := fmt.Sprintf("league/%s/draftresults%s", leagueKey, page.suffix())
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooDraftResultsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse draft results response: %w", err)
	}

	var results []DraftResult
	for _, item := range resp.FantasyContent.League.DraftResults {
		results = append(results, convertYahooDraftResult(item.DraftResult))
	}

	return results, nil
}

func (c *Client) fetchTransactions(ctx context.Context, leagueKey string, page PageOptions) ([]Transaction, error) {
	endpoint := fmt.Sprintf("league/%s/transactions%s", leagueKey, page.suffix())
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	var resp yahooTransactionsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse transactions response: %w", err)
	}

	var transactions []Transaction
	for _, item := range resp.FantasyContent.League.Transactions {
		transactions = append(transactions, convertYahooTransaction(item.Transaction))
	}

	return transactions, nil
}

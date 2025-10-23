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
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	apiKey       string
	apiSecret    string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
	baseURL      string
	tokenURL     string
	cache        *APICache
	tokenMutex   sync.Mutex
	cacheEnabled bool
}

type APICache struct {
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
	YahooTeamID   string
	YahooTeamKey  string
	TeamName      string
	ManagerName   string
	Wins          int
	Losses        int
	Ties          int
	Rank          int
}

type Roster struct {
	TeamID       string
	PlayerID     string
	PlayerKey    string
	Position     string
	SelectedPos  string
	IsStarting   bool
}

type yahooLeaguesResponse struct {
	Fantasy_Content struct {
		Users []struct {
			User []struct {
				Games []struct {
					Game []struct {
						Leagues []struct {
							League struct {
								League_Key  string `json:"league_key"`
								League_ID   string `json:"league_id"`
								Name        string `json:"name"`
								Season      string `json:"season"`
								Scoring_Type string `json:"scoring_type"`
								Num_Teams   int    `json:"num_teams"`
								Current_Week int   `json:"current_week"`
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
					Team_Key    string `json:"team_key"`
					Team_ID     string `json:"team_id"`
					Name        string `json:"name"`
					Managers    []struct {
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
						Player_Key        string `json:"player_key"`
						Player_ID         string `json:"player_id"`
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

func NewClient(apiKey, apiSecret string, db *sql.DB) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("YAHOO_CONSUMER_KEY")
	}
	if apiSecret == "" {
		apiSecret = os.Getenv("YAHOO_CONSUMER_SECRET")
	}

	accessToken := os.Getenv("YAHOO_ACCESS_TOKEN")
	refreshToken := os.Getenv("YAHOO_REFRESH_TOKEN")
	baseURL := os.Getenv("YAHOO_BASE_URL")
	if baseURL == "" {
		baseURL = "https://fantasysports.yahooapis.com/fantasy/v2"
	}

	cacheEnabled := os.Getenv("YAHOO_ENABLE_CACHE") == "true"

	tokenURL := "https://api.login.yahoo.com/oauth2/get_token"

	return &Client{
		apiKey:       apiKey,
		apiSecret:    apiSecret,
		accessToken:  accessToken,
		refreshToken: refreshToken,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		baseURL:      baseURL,
		tokenURL:     tokenURL,
		cache:        &APICache{db: db},
		cacheEnabled: cacheEnabled,
	}
}

func (c *Client) GetUserLeagues(ctx context.Context, gameKey string) ([]League, error) {
	cacheKey := fmt.Sprintf("user:leagues:%s", gameKey)

	if c.cacheEnabled {
		if cached, err := c.cache.Get(cacheKey); err == nil {
			var leagues []League
			if json.Unmarshal([]byte(cached), &leagues) == nil {
				return leagues, nil
			}
		}
	}

	leagues, err := c.fetchLeagues(ctx, gameKey)
	if err != nil {
		return nil, err
	}

	if c.cacheEnabled {
		c.cache.Set(cacheKey, leagues, 24*time.Hour)
	}
	return leagues, nil
}

func (c *Client) GetLeagueTeams(ctx context.Context, leagueKey string) ([]Team, error) {
	cacheKey := fmt.Sprintf("league:%s:teams", leagueKey)

	if c.cacheEnabled {
		if cached, err := c.cache.Get(cacheKey); err == nil {
			var teams []Team
			if json.Unmarshal([]byte(cached), &teams) == nil {
				return teams, nil
			}
		}
	}

	teams, err := c.fetchTeams(ctx, leagueKey)
	if err != nil {
		return nil, err
	}

	if c.cacheEnabled {
		c.cache.Set(cacheKey, teams, 6*time.Hour)
	}
	return teams, nil
}

func (c *Client) GetTeamRoster(ctx context.Context, teamKey string) ([]Roster, error) {
	cacheKey := fmt.Sprintf("team:%s:roster", teamKey)

	if c.cacheEnabled {
		if cached, err := c.cache.Get(cacheKey); err == nil {
			var roster []Roster
			if json.Unmarshal([]byte(cached), &roster) == nil {
				return roster, nil
			}
		}
	}

	roster, err := c.fetchRoster(ctx, teamKey)
	if err != nil {
		return nil, err
	}

	if c.cacheEnabled {
		c.cache.Set(cacheKey, roster, 1*time.Hour)
	}
	return roster, nil
}

func (c *Client) refreshAccessToken() error {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	if c.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequest("POST", c.tokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	authHeader := base64.StdEncoding.EncodeToString([]byte(c.apiKey + ":" + c.apiSecret))
	req.Header.Set("Authorization", "Basic "+authHeader)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed (status %d): %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	if tokenResp.RefreshToken != "" {
		c.refreshToken = tokenResp.RefreshToken
	}

	fmt.Printf("✅ Refreshed Yahoo access token (expires in %d seconds)\n", tokenResp.ExpiresIn)
	return nil
}

func (c *Client) makeRequest(ctx context.Context, endpoint string) ([]byte, error) {
	if c.accessToken == "" {
		return nil, fmt.Errorf("Yahoo access token not configured - set YAHOO_ACCESS_TOKEN environment variable")
	}

	url := fmt.Sprintf("%s/%s?format=json", c.baseURL, endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		if strings.Contains(string(body), "token_expired") {
			if err := c.refreshAccessToken(); err != nil {
				return nil, fmt.Errorf("failed to refresh expired token: %w", err)
			}

			req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create retry request: %w", err)
			}
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.accessToken))
			req.Header.Set("Accept", "application/json")

			resp, err = c.httpClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("failed to retry request: %w", err)
			}
			defer resp.Body.Close()
		}
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Yahoo API error (status %d): %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
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
		eligiblePos := ""
		if len(p.Eligible_Positions) > 0 {
			eligiblePos = p.Eligible_Positions[0].Position
		}
		roster = append(roster, Roster{
			PlayerID:    p.Player_ID,
			PlayerKey:   p.Player_Key,
			Position:    eligiblePos,
			SelectedPos: p.Selected_Position.Position,
			IsStarting:  p.Selected_Position.Position != "BN",
		})
	}

	return roster, nil
}

func (c *APICache) Get(key string) (string, error) {
	var value string
	var expiresAt time.Time

	query := `SELECT cache_value, expires_at FROM yahoo_api_cache WHERE cache_key = ?`
	err := c.db.QueryRow(query, key).Scan(&value, &expiresAt)
	if err != nil {
		return "", err
	}

	if time.Now().After(expiresAt) {
		c.Delete(key)
		return "", fmt.Errorf("cache expired")
	}

	return value, nil
}

func (c *APICache) Set(key string, value interface{}, ttl time.Duration) error {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(ttl)

	query := `INSERT OR REPLACE INTO yahoo_api_cache (cache_key, cache_value, expires_at) VALUES (?, ?, ?)`
	_, err = c.db.Exec(query, key, string(jsonValue), expiresAt)
	return err
}

func (c *APICache) Delete(key string) error {
	query := `DELETE FROM yahoo_api_cache WHERE cache_key = ?`
	_, err := c.db.Exec(query, key)
	return err
}

func (c *APICache) CleanExpired() error {
	query := `DELETE FROM yahoo_api_cache WHERE expires_at < datetime('now')`
	_, err := c.db.Exec(query)
	return err
}

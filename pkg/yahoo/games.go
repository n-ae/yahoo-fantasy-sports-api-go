package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

var gameIDMap = map[string]map[string]int{
	"mlb": {
		"2001": 12, "2002": 39, "2003": 74, "2004": 98, "2005": 113,
		"2006": 147, "2007": 171, "2008": 195, "2009": 215, "2010": 238,
		"2011": 253, "2012": 268, "2013": 308, "2014": 328, "2015": 346,
		"2016": 357, "2017": 370, "2018": 378, "2019": 388, "2020": 398,
		"2021": 404, "2022": 412, "2023": 422, "2024": 431, "2025": 458,
	},
	"nfl": {
		"2001": 57, "2002": 49, "2003": 79, "2004": 101, "2005": 124,
		"2006": 153, "2007": 175, "2008": 199, "2009": 222, "2010": 242,
		"2011": 257, "2012": 273, "2013": 314, "2014": 331, "2015": 348,
		"2016": 359, "2017": 371, "2018": 380, "2019": 390, "2020": 399,
		"2021": 406, "2022": 414, "2023": 423, "2024": 449, "2025": 461,
	},
	"nba": {
		"2001": 16, "2002": 67, "2003": 95, "2004": 112, "2005": 131,
		"2006": 165, "2007": 187, "2008": 211, "2009": 234, "2010": 249,
		"2011": 265, "2012": 304, "2013": 322, "2014": 342, "2015": 353,
		"2016": 364, "2017": 375, "2018": 385, "2019": 395, "2020": 402,
		"2021": 410, "2022": 418, "2023": 428, "2024": 454, "2025": 466,
	},
	"nhl": {
		"2001": 15, "2002": 64, "2003": 94, "2004": 111, "2005": 130,
		"2006": 164, "2007": 186, "2008": 210, "2009": 233, "2010": 248,
		"2011": 263, "2012": 303, "2013": 321, "2014": 341, "2015": 352,
		"2016": 363, "2017": 376, "2018": 386, "2019": 396, "2020": 403,
		"2021": 411, "2022": 419, "2023": 427, "2024": 453, "2025": 465,
	},
}

func GetGameID(gameCode string, season int) (int, error) {
	seasonStr := strconv.Itoa(season)

	seasons, ok := gameIDMap[gameCode]
	if !ok {
		return 0, fmt.Errorf("invalid game code '%s', must be 'mlb', 'nba', 'nhl', or 'nfl'", gameCode)
	}

	gameID, ok := seasons[seasonStr]
	if !ok {
		return 0, fmt.Errorf("invalid season %d for %s", season, gameCode)
	}

	return gameID, nil
}

func GetGameKey(gameCode string, season int) (string, error) {
	gameID, err := GetGameID(gameCode, season)
	if err != nil {
		return "", err
	}
	return strconv.Itoa(gameID), nil
}

// GameKey resolves the Yahoo game key for a sport code and season.
//
// It first consults the built-in static map (offline, no request), which covers
// the seasons GetGameKey knows. On a miss — e.g. a season newer than the shipped
// map — it queries Yahoo's games resource and caches the result. Unlike the
// package-level GetGameKey, this requires a configured client and may perform a
// network request. See docs/adr/0005-dynamic-game-key-discovery.md.
func (c *Client) GameKey(ctx context.Context, gameCode string, season int) (string, error) {
	if key, err := GetGameKey(gameCode, season); err == nil {
		return key, nil
	}

	cacheKey := fmt.Sprintf("gamekey:%s:%d", gameCode, season)
	if v, ok := cacheGet[string](ctx, c, cacheKey); ok {
		return v, nil
	}

	endpoint := fmt.Sprintf("games;game_codes=%s;seasons=%d", gameCode, season)
	data, err := c.makeRequest(ctx, endpoint)
	if err != nil {
		return "", err
	}

	key, err := parseGameKey(data)
	if err != nil {
		return "", fmt.Errorf("discover game key for %s %d: %w", gameCode, season, err)
	}

	// Game keys are stable for a season; cache generously.
	cacheSet(ctx, c, cacheKey, key, 30*24*time.Hour)
	return key, nil
}

// parseGameKey extracts a game_key from a Yahoo games-resource response. Yahoo's
// games collection is an object with numbered string keys plus "count", and each
// "game" may be an array or a single object — handle both defensively.
func parseGameKey(data []byte) (string, error) {
	var resp struct {
		FantasyContent struct {
			Games json.RawMessage `json:"games"`
		} `json:"fantasy_content"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse games response: %w", err)
	}

	var games map[string]json.RawMessage
	if err := json.Unmarshal(resp.FantasyContent.Games, &games); err != nil {
		return "", fmt.Errorf("parse games collection: %w", err)
	}

	for k, v := range games {
		if k == "count" {
			continue
		}
		var entry struct {
			Game json.RawMessage `json:"game"`
		}
		if json.Unmarshal(v, &entry) != nil {
			continue
		}
		if key := extractGameKey(entry.Game); key != "" {
			return key, nil
		}
	}
	return "", fmt.Errorf("no game found in response")
}

// extractGameKey reads game_key from a "game" value that may be either an array
// whose first element carries the metadata, or a single metadata object.
func extractGameKey(raw json.RawMessage) string {
	type gameMeta struct {
		GameKey string `json:"game_key"`
	}
	var arr []gameMeta
	if json.Unmarshal(raw, &arr) == nil {
		for _, g := range arr {
			if g.GameKey != "" {
				return g.GameKey
			}
		}
	}
	var single gameMeta
	if json.Unmarshal(raw, &single) == nil {
		return single.GameKey
	}
	return ""
}

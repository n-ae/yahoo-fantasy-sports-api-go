package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/n-ae/yahoo-fantasy-sports-api-go/v2/cmd/nba-tool/internal/repository"
	"github.com/n-ae/yahoo-fantasy-sports-api-go/v2/pkg/yahoo"
)

// DefaultScoringSettings is the fallback fantasy scoring model used when a
// LeagueService has no ScoringSettings configured. It is one opinionated
// points model; override it per-service by setting LeagueService.ScoringSettings
// (ideally from the league's actual Yahoo settings).
var DefaultScoringSettings = map[string]float64{
	"PTS": 1.0,
	"REB": 1.2,
	"AST": 1.5,
	"STL": 3.0,
	"BLK": 3.0,
	"TO":  -1.0,
	"3PM": 1.0,
}

type LeagueService struct {
	yahooClient *yahoo.Client
	leagueRepo  *repository.LeagueRepository
	teamRepo    *repository.TeamRepository
	rosterRepo  *repository.RosterRepository
	db          *sql.DB

	// ScoringSettings, when non-nil, overrides DefaultScoringSettings for
	// imported leagues.
	ScoringSettings map[string]float64
}

func NewLeagueService(
	yahooClient *yahoo.Client,
	leagueRepo *repository.LeagueRepository,
	teamRepo *repository.TeamRepository,
	rosterRepo *repository.RosterRepository,
	db *sql.DB,
) *LeagueService {
	return &LeagueService{
		yahooClient: yahooClient,
		leagueRepo:  leagueRepo,
		teamRepo:    teamRepo,
		rosterRepo:  rosterRepo,
		db:          db,
	}
}

func (s *LeagueService) ImportLeague(ctx context.Context, yahooLeagueID string, isUserTeamID string) error {
	existing, err := s.leagueRepo.GetByYahooID(ctx, yahooLeagueID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing league: %w", err)
	}

	if existing != nil {
		return fmt.Errorf("league already imported")
	}

	leagues, err := s.yahooClient.GetUserLeagues(ctx, "nba")
	if err != nil {
		return fmt.Errorf("failed to fetch leagues from Yahoo: %w", err)
	}

	var targetLeague *yahoo.League
	for _, league := range leagues {
		if league.YahooLeagueID == yahooLeagueID {
			targetLeague = &league
			break
		}
	}

	if targetLeague == nil {
		return fmt.Errorf("league not found in user's leagues")
	}

	scoringSettings := s.ScoringSettings
	if scoringSettings == nil {
		scoringSettings = DefaultScoringSettings
	}
	scoringJSON, _ := json.Marshal(scoringSettings)

	league := &repository.League{
		YahooLeagueID:   targetLeague.YahooLeagueID,
		YahooGameKey:    targetLeague.YahooGameKey,
		LeagueName:      targetLeague.LeagueName,
		SeasonYear:      targetLeague.SeasonYear,
		ScoringType:     targetLeague.ScoringType,
		ScoringSettings: string(scoringJSON),
		NumTeams:        targetLeague.NumTeams,
		CurrentWeek:     targetLeague.CurrentWeek,
	}

	if err := s.leagueRepo.Create(ctx, league); err != nil {
		return fmt.Errorf("failed to save league: %w", err)
	}

	if err := s.SyncTeamsAndRosters(ctx, league.ID, targetLeague.YahooLeagueID, isUserTeamID); err != nil {
		return fmt.Errorf("failed to sync teams and rosters: %w", err)
	}

	return nil
}

func (s *LeagueService) SyncTeamsAndRosters(ctx context.Context, leagueID int, yahooLeagueID string, userTeamID string) error {
	// Build the Yahoo league key from the league's actual game key (e.g.
	// "454.l.<id>") rather than assuming the current-season "nba" code.
	gameKey := "nba"
	if lg, err := s.leagueRepo.GetByYahooID(ctx, yahooLeagueID); err == nil && lg != nil && lg.YahooGameKey != "" {
		gameKey = lg.YahooGameKey
	}
	leagueKey := fmt.Sprintf("%s.l.%s", gameKey, yahooLeagueID)

	teams, err := s.yahooClient.GetLeagueTeams(ctx, leagueKey)
	if err != nil {
		return fmt.Errorf("failed to fetch teams: %w", err)
	}

	for _, yahooTeam := range teams {
		isUserTeam := yahooTeam.YahooTeamID == userTeamID

		team := &repository.FantasyTeam{
			LeagueID:     leagueID,
			YahooTeamID:  yahooTeam.YahooTeamID,
			YahooTeamKey: yahooTeam.YahooTeamKey,
			TeamName:     yahooTeam.TeamName,
			ManagerName:  yahooTeam.ManagerName,
			IsUserTeam:   isUserTeam,
			Wins:         yahooTeam.Wins,
			Losses:       yahooTeam.Losses,
			Ties:         yahooTeam.Ties,
			Rank:         yahooTeam.Rank,
		}

		if err := s.teamRepo.Create(ctx, team); err != nil {
			return fmt.Errorf("failed to save team %s: %w", yahooTeam.TeamName, err)
		}

		roster, err := s.yahooClient.GetTeamRoster(ctx, yahooTeam.YahooTeamKey)
		if err != nil {
			return fmt.Errorf("failed to fetch roster for team %s: %w", yahooTeam.TeamName, err)
		}

		for _, rosterEntry := range roster {
			playerID, err := s.rosterRepo.GetPlayerIDByYahooKey(ctx, rosterEntry.PlayerKey)
			if err != nil {
				// Surface the partial import instead of silently dropping the
				// player (assessment M5 10.5).
				log.Printf("warning: skipping roster player %s on team %s: %v", rosterEntry.PlayerKey, yahooTeam.TeamName, err)
				continue
			}

			rosterPos := ""
			if len(rosterEntry.EligiblePositions) > 0 {
				rosterPos = rosterEntry.EligiblePositions[0]
			}
			entry := &repository.RosterEntry{
				TeamID:           team.ID,
				PlayerID:         playerID,
				RosterPosition:   rosterPos,
				SelectedPosition: rosterEntry.SelectedPos,
				IsStarting:       rosterEntry.IsStarting,
			}

			if err := s.rosterRepo.Create(ctx, entry); err != nil {
				return fmt.Errorf("failed to save roster entry: %w", err)
			}
		}
	}

	now := time.Now()
	if err := s.leagueRepo.UpdateSyncTime(ctx, leagueID); err != nil {
		return fmt.Errorf("failed to update sync time: %w", err)
	}

	syncQuery := `
		INSERT INTO sync_history (league_id, sync_type, sync_status, items_synced, completed_at)
		VALUES (?, 'full', 'success', ?, ?)
	`
	if _, err := s.db.ExecContext(ctx, syncQuery, leagueID, len(teams), now); err != nil {
		return fmt.Errorf("failed to record sync history: %w", err)
	}

	return nil
}

func (s *LeagueService) GetUserLeagues(ctx context.Context) ([]*repository.League, error) {
	return s.leagueRepo.GetAll(ctx)
}

func (s *LeagueService) GetLeagueTeams(ctx context.Context, leagueID int) ([]*repository.FantasyTeam, error) {
	return s.teamRepo.GetByLeague(ctx, leagueID)
}

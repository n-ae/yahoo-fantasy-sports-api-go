package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"yahoo-fantasy-sdk/internal/repository"
	"yahoo-fantasy-sdk/internal/yahoo"
)

type LeagueService struct {
	yahooClient *yahoo.Client
	leagueRepo  *repository.LeagueRepository
	teamRepo    *repository.TeamRepository
	rosterRepo  *repository.RosterRepository
	db          *sql.DB
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

	scoringSettings := map[string]float64{
		"PTS": 1.0,
		"REB": 1.2,
		"AST": 1.5,
		"STL": 3.0,
		"BLK": 3.0,
		"TO":  -1.0,
		"3PM": 1.0,
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
	leagueKey := fmt.Sprintf("nba.l.%s", yahooLeagueID)

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
				continue
			}

			entry := &repository.RosterEntry{
				TeamID:           team.ID,
				PlayerID:         playerID,
				RosterPosition:   rosterEntry.Position,
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
	s.db.ExecContext(ctx, syncQuery, leagueID, len(teams), now)

	return nil
}

func (s *LeagueService) GetUserLeagues(ctx context.Context) ([]*repository.League, error) {
	return s.leagueRepo.GetAll(ctx)
}

func (s *LeagueService) GetLeagueTeams(ctx context.Context, leagueID int) ([]*repository.FantasyTeam, error) {
	return s.teamRepo.GetByLeague(ctx, leagueID)
}

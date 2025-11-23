package services

import (
	"CodeRewievService/internal/models"
	"CodeRewievService/internal/repository"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

const (
	massDeactivationWarningThreshold = 100 * time.Millisecond
)

type TeamService struct {
	teamRepository *repository.TeamRepository
	prRepository   *repository.PullRequestRepository
	logger         *slog.Logger
}

func NewTeamService(
	teamRepository *repository.TeamRepository,
	prRepository *repository.PullRequestRepository,
	logger *slog.Logger,
) *TeamService {
	return &TeamService{
		teamRepository: teamRepository,
		prRepository:   prRepository,
		logger:         logger,
	}
}

func (s *TeamService) Add(team *models.Team) error {
	if err := s.validateTeamInput(team); err != nil {
		return err
	}

	if err := s.checkTeamNotExists(team.TeamName); err != nil {
		return err
	}

	return s.teamRepository.CreateWithUsers(team, team.Members)
}

func (s *TeamService) Get(teamName string) (*models.Team, error) {
	if teamName == "" {
		return nil, errors.New("team name cannot be empty")
	}

	team, err := s.teamRepository.FindByName(teamName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("team not found")
	}
	if err != nil {
		return nil, err
	}

	users, err := s.teamRepository.GetUsersByTeam(team.TeamName)
	if err != nil {
		return nil, err
	}

	team.Members = users
	return team, nil
}

func (s *TeamService) MassDeactivateTeamUsers(teamName string) error {
	startTime := time.Now()

	return s.teamRepository.Transaction(func(tx *gorm.DB) error {
		if err := s.validateTeamExists(teamName); err != nil {
			return err
		}

		rowsAffected, err := s.teamRepository.DeactivateUsers(teamName)
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			return nil
		}

		if err := s.removeReviewersFromOpenPRs(tx, teamName); err != nil {
			return err
		}

		s.logPerformanceWarning(teamName, startTime)
		return nil
	})
}

func (s *TeamService) validateTeamInput(team *models.Team) error {
	if team == nil {
		return errors.New("team cannot be nil")
	}
	if team.TeamName == "" {
		return errors.New("team name cannot be empty")
	}
	return nil
}

func (s *TeamService) checkTeamNotExists(teamName string) error {
	_, err := s.teamRepository.FindByName(teamName)
	if err == nil {
		return errors.New("team already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	return nil
}

func (s *TeamService) validateTeamExists(teamName string) error {
	_, err := s.teamRepository.FindByName(teamName)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("team not found")
	}
	return err
}

func (s *TeamService) removeReviewersFromOpenPRs(tx *gorm.DB, teamName string) error {
	openPRs, err := s.teamRepository.GetOpenPRsByTeam(tx, teamName)
	if err != nil {
		return err
	}

	if len(openPRs) == 0 {
		return nil
	}

	prIDs := s.extractPRIDs(openPRs)
	return s.prRepository.DeleteReviewersByPRIDsInTx(tx, prIDs)
}

func (s *TeamService) extractPRIDs(prs []models.PullRequest) []string {
	prIDs := make([]string, len(prs))
	for i, pr := range prs {
		prIDs[i] = pr.PullRequestID
	}
	return prIDs
}

func (s *TeamService) logPerformanceWarning(teamName string, startTime time.Time) {
	duration := time.Since(startTime)
	if duration > massDeactivationWarningThreshold {
		s.logger.Warn("MassDeactivateTeamUsers execution time exceeded threshold",
			"team", teamName,
			"duration", duration,
			"threshold", massDeactivationWarningThreshold,
		)
	}
}

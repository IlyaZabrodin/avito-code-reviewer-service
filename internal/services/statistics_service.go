package services

import (
	"CodeRewievService/internal/models"
	"CodeRewievService/internal/repository"
)

type StatisticsService struct {
	statsRepository *repository.StatisticsRepository
}

func NewStatisticsService(statsRepository *repository.StatisticsRepository) *StatisticsService {
	return &StatisticsService{
		statsRepository: statsRepository,
	}
}

func (s *StatisticsService) GetAssignmentsStats(teamName string) ([]models.AssignmentStats, error) {
	users, err := s.statsRepository.GetTeamUsers(teamName)
	if err != nil {
		return nil, err
	}

	assignmentStats := make([]models.AssignmentStats, 0, len(users))
	for _, user := range users {
		assignedReviews, err := s.statsRepository.CountUserAssignedReviews(user.UserID)
		if err != nil {
			continue
		}

		stats := models.AssignmentStats{
			UserID:          user.UserID,
			Username:        user.Username,
			AssignedReviews: assignedReviews,
		}
		assignmentStats = append(assignmentStats, stats)
	}

	return assignmentStats, nil
}

func (s *StatisticsService) TeamExists(teamName string) (bool, error) {
	return s.statsRepository.TeamExists(teamName)
}

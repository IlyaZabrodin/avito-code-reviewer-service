package repository

import (
	"CodeRewievService/internal/models"

	"gorm.io/gorm"
)

type StatisticsRepository struct {
	database *gorm.DB
}

func NewStatisticsRepository(database *gorm.DB) *StatisticsRepository {
	return &StatisticsRepository{
		database: database,
	}
}
func (r *StatisticsRepository) GetTeamUsers(teamName string) ([]models.User, error) {
	var users []models.User
	err := r.database.Where("team_name = ?", teamName).Find(&users).Error
	return users, err
}

func (r *StatisticsRepository) CountUserAssignedReviews(userID string) (int64, error) {
	var count int64
	err := r.database.Model(&models.PullRequestReviewer{}).
		Where("user_id = ?", userID).
		Count(&count).Error

	return count, err
}

func (r *StatisticsRepository) TeamExists(teamName string) (bool, error) {
	var exists bool
	err := r.database.Model(&models.Team{}).
		Select("count(*) > 0").
		Where("team_name = ?", teamName).
		Find(&exists).Error

	return exists, err
}

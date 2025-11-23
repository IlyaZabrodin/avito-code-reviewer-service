package repository

import (
	"CodeRewievService/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserRepository struct {
	database *gorm.DB
}

func NewUserRepository(database *gorm.DB) *UserRepository {
	return &UserRepository{
		database: database,
	}
}

func (r *UserRepository) FindByID(userID string) (*models.User, error) {
	var user models.User
	result := r.database.Where("user_id = ?", userID).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

func (r *UserRepository) FindActiveByID(userID string) (*models.User, error) {
	var user models.User
	result := r.database.Where("user_id = ? AND is_active = ?", userID, true).First(&user)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &user, nil
}

func (r *UserRepository) Update(user *models.User) error {
	return r.database.Save(user).Error
}

func (r *UserRepository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]models.User, error) {
	var users []models.User
	err := r.database.
		Where("team_name = ? AND user_id != ? AND is_active = ?", teamName, excludeUserID, true).
		Find(&users).Error

	return users, err
}

func (r *UserRepository) GetAvailableReviewers(
	teamName string,
	excludeUserIDs []string,
	excludeAuthorID string,
) ([]models.User, error) {
	var users []models.User
	query := r.database.
		Where("team_name = ? AND user_id != ? AND is_active = ?", teamName, excludeAuthorID, true)

	for _, userID := range excludeUserIDs {
		query = query.Where("user_id != ?", userID)
	}

	err := query.Find(&users).Error
	return users, err
}

func (r *UserRepository) GetPullRequestsByReviewer(userID string) ([]models.PullRequest, error) {
	var pullRequests []models.PullRequest
	err := r.database.
		Joins("JOIN pull_request_reviewers ON pull_requests.pull_request_id = pull_request_reviewers.pull_request_id").
		Where("pull_request_reviewers.user_id = ?", userID).
		Find(&pullRequests).Error

	return pullRequests, err
}

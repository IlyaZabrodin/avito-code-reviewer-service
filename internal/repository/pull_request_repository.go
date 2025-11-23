package repository

import (
	"CodeRewievService/internal/models"
	"errors"

	"gorm.io/gorm"
)

type PullRequestRepository struct {
	database *gorm.DB
}

func NewPullRequestRepository(database *gorm.DB) *PullRequestRepository {
	return &PullRequestRepository{
		database: database,
	}
}

func (r *PullRequestRepository) FindByID(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	result := r.database.Where("pull_request_id = ?", prID).First(&pr)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &pr, nil
}

func (r *PullRequestRepository) FindByIDWithRelations(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	result := r.database.
		Preload("AssignedReviewers.User").
		Preload("Author").
		Where("pull_request_id = ?", prID).
		First(&pr)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &pr, nil
}

func (r *PullRequestRepository) FindByIDWithReviewers(prID string) (*models.PullRequest, error) {
	var pr models.PullRequest
	result := r.database.
		Preload("AssignedReviewers").
		Where("pull_request_id = ?", prID).
		First(&pr)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &pr, nil
}

func (r *PullRequestRepository) Create(pr *models.PullRequest) error {
	return r.database.Create(pr).Error
}

func (r *PullRequestRepository) Update(pr *models.PullRequest) error {
	return r.database.Save(pr).Error
}

func (r *PullRequestRepository) DeleteReviewer(prID string, userID string) error {
	return r.database.
		Where("pull_request_id = ? AND user_id = ?", prID, userID).
		Delete(&models.PullRequestReviewer{}).Error
}

func (r *PullRequestRepository) CreateReviewer(reviewer *models.PullRequestReviewer) error {
	return r.database.Create(reviewer).Error
}

func (r *PullRequestRepository) DeleteReviewersByPRIDs(prIDs []string) error {
	return r.database.
		Where("pull_request_id IN (?)", prIDs).
		Delete(&models.PullRequestReviewer{}).Error
}

func (r *PullRequestRepository) DeleteReviewersByPRIDsInTx(tx *gorm.DB, prIDs []string) error {
	return tx.
		Where("pull_request_id IN (?)", prIDs).
		Delete(&models.PullRequestReviewer{}).Error
}

func (r *PullRequestRepository) Transaction(fn func(*gorm.DB) error) error {
	return r.database.Transaction(fn)
}

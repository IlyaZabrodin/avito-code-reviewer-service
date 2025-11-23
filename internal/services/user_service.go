package services

import (
	"CodeRewievService/internal/models"
	"CodeRewievService/internal/repository"
	"errors"

	"gorm.io/gorm"
)

type UserService struct {
	userRepository *repository.UserRepository
}

func NewUserService(userRepository *repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

func (s *UserService) SetIsActive(user *models.User) (*models.User, error) {
	if err := s.validateUserInput(user); err != nil {
		return nil, err
	}

	existingUser, err := s.userRepository.FindByID(user.UserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	existingUser.IsActive = user.IsActive

	if err := s.userRepository.Update(existingUser); err != nil {
		return nil, err
	}

	return existingUser, nil
}

func (s *UserService) GetReview(userID string) (*models.UserReview, error) {
	if userID == "" {
		return nil, errors.New("user_id cannot be empty")
	}

	if err := s.validateUserExists(userID); err != nil {
		return nil, err
	}

	pullRequests, err := s.userRepository.GetPullRequestsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	return s.buildUserReview(userID, pullRequests), nil
}

func (s *UserService) validateUserInput(user *models.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}
	if user.UserID == "" {
		return errors.New("user_id cannot be empty")
	}
	return nil
}

func (s *UserService) validateUserExists(userID string) error {
	_, err := s.userRepository.FindByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New("user not found")
	}
	return err
}

func (s *UserService) buildUserReview(userID string, pullRequests []models.PullRequest) *models.UserReview {
	pullRequestDTOs := make([]models.PullRequestDTO, len(pullRequests))
	for i, pr := range pullRequests {
		pullRequestDTOs[i] = pr.ToResponse()
	}

	return &models.UserReview{
		UserID:       userID,
		PullRequests: pullRequestDTOs,
	}
}

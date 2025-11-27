package services

import (
	"CodeRewievService/internal/models"
	"CodeRewievService/internal/repository"
	"errors"
	"math/rand"
	"time"

	"gorm.io/gorm"
)

const defaultReviewersCount = 2

type PullRequestService struct {
	prRepository   *repository.PullRequestRepository
	userRepository *repository.UserRepository
	randomizer     *rand.Rand
}

func NewPullRequestService(
	prRepository *repository.PullRequestRepository,
	userRepository *repository.UserRepository,
) *PullRequestService {
	return &PullRequestService{
		prRepository:   prRepository,
		userRepository: userRepository,
		//nolint:gosec // math/rand достаточно для балансировки нагрузки
		randomizer: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *PullRequestService) Create(pr *models.PullRequest) (*models.PullRequest, error) {
	if err := s.validatePullRequestInput(pr); err != nil {
		return nil, err
	}

	if err := s.checkPRNotExists(pr.PullRequestID); err != nil {
		return nil, err
	}

	author, err := s.validateAuthor(pr.AuthorID)
	if err != nil {
		return nil, err
	}

	teamMembers, err := s.userRepository.GetActiveTeamMembers(author.TeamName, pr.AuthorID)
	if err != nil {
		return nil, err
	}

	assignedReviewers := s.selectRandomReviewers(teamMembers, defaultReviewersCount, pr.PullRequestID)

	newPR := models.PullRequest{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            "OPEN",
		AssignedReviewers: assignedReviewers,
	}

	if err := s.createPRInTransaction(&newPR); err != nil {
		return nil, err
	}

	return s.prRepository.FindByIDWithRelations(pr.PullRequestID)
}

func (s *PullRequestService) Merge(prID string) (*models.PullRequest, error) {
	if prID == "" {
		return nil, errors.New("pull_request_id cannot be empty")
	}

	pr, err := s.prRepository.FindByID(prID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrPullRequestNotFound
	}

	if err != nil {
		return nil, err
	}

	if pr.Status == "MERGED" {
		return pr, nil
	}

	now := time.Now()
	pr.Status = "MERGED"
	pr.MergedAt = &now

	if err := s.prRepository.Update(pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (s *PullRequestService) Reassign(prID string, oldUserID string) (*models.PullRequest, string, error) {
	if err := s.validateReassignInput(prID, oldUserID); err != nil {
		return nil, "", err
	}

	pr, err := s.prRepository.FindByIDWithReviewers(prID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", models.ErrPullRequestNotFound
	}

	if err != nil {
		return nil, "", err
	}

	if pr.Status == "MERGED" {
		return nil, "", models.ErrPullRequestAlreadyMerged
	}

	if err := s.validateReviewerAssigned(pr, oldUserID); err != nil {
		return nil, "", err
	}

	oldReviewer, err := s.userRepository.FindByID(oldUserID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", models.ErrPullRequestNotFound
	}

	if err != nil {
		return nil, "", err
	}

	newReviewerID, err := s.performReassignment(pr, oldReviewer, oldUserID)
	if err != nil {
		return nil, "", err
	}

	updatedPR, err := s.prRepository.FindByIDWithRelations(prID)
	if err != nil {
		return nil, "", err
	}

	return updatedPR, newReviewerID, nil
}

func (s *PullRequestService) validatePullRequestInput(pr *models.PullRequest) error {
	if pr == nil {
		return errors.New("pull request cannot be nil")
	}

	if pr.PullRequestID == "" {
		return errors.New("pull_request_id cannot be empty")
	}

	return nil
}

func (s *PullRequestService) checkPRNotExists(prID string) error {
	_, err := s.prRepository.FindByID(prID)
	if err == nil {
		return models.ErrPullRequestAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func (s *PullRequestService) validateAuthor(authorID string) (*models.User, error) {
	author, err := s.userRepository.FindActiveByID(authorID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, models.ErrAuthorNotFoundOrInactive
	}

	return author, err
}

func (s *PullRequestService) createPRInTransaction(pr *models.PullRequest) error {
	return s.prRepository.Transaction(func(tx *gorm.DB) error {
		return s.prRepository.Create(pr)
	})
}

func (s *PullRequestService) validateReassignInput(prID, oldUserID string) error {
	if prID == "" {
		return errors.New("pull_request_id cannot be empty")
	}

	if oldUserID == "" {
		return errors.New("old_user_id cannot be empty")
	}

	return nil
}

func (s *PullRequestService) validateReviewerAssigned(pr *models.PullRequest, oldUserID string) error {
	for _, reviewer := range pr.AssignedReviewers {
		if reviewer.UserID == oldUserID {
			return nil
		}
	}

	return models.ErrReviewerNotAssigned
}

func (s *PullRequestService) performReassignment(
	pr *models.PullRequest,
	oldReviewer *models.User,
	oldUserID string,
) (string, error) {
	excludeUserIDs := s.extractReviewerIDs(pr.AssignedReviewers)

	availableReviewers, err := s.userRepository.GetAvailableReviewers(
		oldReviewer.TeamName,
		excludeUserIDs,
		pr.AuthorID,
	)

	if err != nil {
		return "", err
	}

	if len(availableReviewers) == 0 {
		return "", models.ErrNoReplacementFound
	}

	newReviewer := availableReviewers[s.randomizer.Intn(len(availableReviewers))]

	err = s.prRepository.Transaction(func(tx *gorm.DB) error {
		if err := s.prRepository.DeleteReviewer(pr.PullRequestID, oldUserID); err != nil {
			return err
		}

		newPRReviewer := models.PullRequestReviewer{
			PullRequestID: pr.PullRequestID,
			UserID:        newReviewer.UserID,
			AssignedAt:    time.Now(),
		}

		return s.prRepository.CreateReviewer(&newPRReviewer)
	})

	return newReviewer.UserID, err
}

func (s *PullRequestService) extractReviewerIDs(reviewers []models.PullRequestReviewer) []string {
	ids := make([]string, len(reviewers))
	for i, reviewer := range reviewers {
		ids[i] = reviewer.UserID
	}

	return ids
}

func (s *PullRequestService) selectRandomReviewers(
	users []models.User,
	maxCount int,
	pullRequestID string,
) []models.PullRequestReviewer {
	if len(users) == 0 {
		return []models.PullRequestReviewer{}
	}

	if len(users) < maxCount {
		maxCount = len(users)
	}

	shuffled := s.shuffleUsers(users)
	reviewers := make([]models.PullRequestReviewer, maxCount)

	for i := 0; i < maxCount; i++ {
		reviewers[i] = models.PullRequestReviewer{
			PullRequestID: pullRequestID,
			UserID:        shuffled[i].UserID,
			AssignedAt:    time.Now(),
		}
	}

	return reviewers
}

func (s *PullRequestService) shuffleUsers(users []models.User) []models.User {
	shuffled := make([]models.User, len(users))
	copy(shuffled, users)

	s.randomizer.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	return shuffled
}

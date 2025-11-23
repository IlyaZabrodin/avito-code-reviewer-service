package controllers

import (
	"CodeRewievService/internal/models"
)

type StatisticsService interface {
	GetAssignmentsStats(teamName string) ([]models.AssignmentStats, error)
	TeamExists(teamName string) (bool, error)
}

type PullRequestService interface {
	Create(PullRequest *models.PullRequest) (*models.PullRequest, error)
	Reassign(prID string, userID string) (*models.PullRequest, string, error)
	Merge(prID string) (*models.PullRequest, error)
}

type TeamService interface {
	Add(team *models.Team) error
	Get(teamName string) (*models.Team, error)
	MassDeactivateTeamUsers(teamName string) error
}

type UserService interface {
	SetIsActive(user *models.User) (*models.User, error)
	GetReview(userID string) (*models.UserReview, error)
}

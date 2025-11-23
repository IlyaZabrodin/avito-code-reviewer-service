package models

import "time"

type Team struct {
	TeamName string `gorm:"primaryKey;column:team_name" json:"team_name"`
	Members  []User  `gorm:"-" json:"members"`
}

type User struct {
	UserID   string `gorm:"primaryKey" json:"user_id"`
	Username string `gorm:"not null" json:"username"`
	TeamName string `gorm:"not null" json:"team_name"`
	IsActive bool   `gorm:"default:true" json:"is_active"`
}

type PullRequest struct {
	PullRequestID   string     `gorm:"primaryKey" json:"pull_request_id"`
	PullRequestName string     `gorm:"not null" json:"pull_request_name"`
	AuthorID        string     `gorm:"not null" json:"author_id"`
	Status          string     `gorm:"type:pull_request_status;default:'OPEN'" json:"status"`
	CreatedAt       time.Time  `gorm:"column:created_at"`
	MergedAt        *time.Time `gorm:"column:merged_at"`
	UpdatedAt       time.Time  `gorm:"column:updated_at"`

	Author            User                  `gorm:"foreignKey:AuthorID" json:"-"`
	AssignedReviewers []PullRequestReviewer `gorm:"foreignKey:PullRequestID" json:"assigned_reviewers"`
}

type PullRequestReviewer struct {
	PullRequestID string    `gorm:"primaryKey" json:"pull_request_id"`
	UserID        string    `gorm:"primaryKey" json:"user_id"`
	AssignedAt    time.Time `gorm:"default:CURRENT_TIMESTAMP" json:"assigned_at"`

	User User `gorm:"foreignKey:UserID;references:UserID" json:"user"`
}

type UserReview struct {
	UserID       string           `json:"user_id"`
	PullRequests []PullRequestDTO `json:"pull_requests"`
}

func (Team) TableName() string {
	return "teams"
}

func (User) TableName() string {
	return "users"
}

func (PullRequest) TableName() string {
	return "pull_requests"
}

func (PullRequestReviewer) TableName() string {
	return "pull_request_reviewers"
}

func (p PullRequestReviewer) PrimaryKey() []string {
	return []string{"pull_request_id", "user_id"}
}

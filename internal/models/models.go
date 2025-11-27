package models

import "time"

type Team struct {
	TeamName string `gorm:"primaryKey;column:team_name" json:"teamName"`
	Members  []User `gorm:"-" json:"teamMembers"`
}

type User struct {
	UserID   string `gorm:"primaryKey;column:user_id" json:"userId"`
	Username string `gorm:"not null;column:username" json:"userName"`
	TeamName string `gorm:"not null;column:team_name;index" json:"teamName"`
	IsActive bool   `gorm:"default:true;column:is_active" json:"isActive"`
}

type PullRequest struct {
	PullRequestID     string                `gorm:"primaryKey;column:pull_request_id" json:"pullRequestId"`
	PullRequestName   string                `gorm:"not null;column:pull_request_name" json:"pullRequestName"`
	AuthorID          string                `gorm:"not null;column:author_id;index" json:"authorId"`
	Status            string                `gorm:"type:pull_request_status;default:'OPEN';column:status" json:"status"`
	CreatedAt         time.Time             `gorm:"autoCreateTime;column:created_at" json:"createdAt"`
	MergedAt          *time.Time            `gorm:"column:merged_at" json:"mergedAt,omitempty"`
	UpdatedAt         time.Time             `gorm:"autoUpdateTime;column:updated_at" json:"updatedAt"`
	Author            User                  `gorm:"foreignKey:AuthorID;references:UserID" json:"-"`
	AssignedReviewers []PullRequestReviewer `gorm:"foreignKey:PullRequestID;references:PullRequestID" json:"assignedReviewers"`
}

type PullRequestReviewer struct {
	PullRequestID string    `gorm:"primaryKey;column:pull_request_id;index:idx_pr_reviewer" json:"pullRequestId"`
	UserID        string    `gorm:"primaryKey;column:user_id;index:idx_pr_reviewer" json:"userId"`
	AssignedAt    time.Time `gorm:"autoCreateTime;default:CURRENT_TIMESTAMP;column:assigned_at" json:"assignedAt"`
	User          User      `gorm:"foreignKey:UserID;references:UserID" json:"user"`
}

type UserReview struct {
	UserID       string           `json:"userId"`
	PullRequests []PullRequestDTO `json:"pullRequests"`
}

type AssignmentStats struct {
	UserID          string `json:"userId"`
	Username        string `json:"userName"`
	AssignedReviews int    `json:"assignedReviews"`
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

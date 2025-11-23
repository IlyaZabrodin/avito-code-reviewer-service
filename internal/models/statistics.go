package models

type AssignmentStats struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	AssignedReviews int64  `json:"assigned_reviews"`
}

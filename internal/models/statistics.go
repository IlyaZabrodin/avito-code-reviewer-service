package models

type UserStatistics struct {
	UserID           string `json:"userId"`
	Username        string `json:"userName"`
	AssignedReviews  int    `json:"assignedReviews"`
}

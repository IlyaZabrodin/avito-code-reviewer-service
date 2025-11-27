package models

import "time"

type PullRequestDTO struct {
	PullRequestID     string   `json:"pullRequestId"`
	PullRequestName   string   `json:"pullRequestName"`
	AuthorID          string   `json:"authorId"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assignedReviewers,omitempty"`
}

func (pr *PullRequest) ToResponse() PullRequestDTO {
	reviewerIDs := make([]string, len(pr.AssignedReviewers))
	for i, reviewer := range pr.AssignedReviewers {
		reviewerIDs[i] = reviewer.UserID
	}

	return PullRequestDTO{
		PullRequestID:     pr.PullRequestID,
		PullRequestName:   pr.PullRequestName,
		AuthorID:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: reviewerIDs,
	}
}

type ResponseCreatePR struct {
	PullRequest PullRequestDTO `json:"pullRequest"`
}

type ResponseMerge struct {
	PullRequest PullRequestDTO `json:"pullRequest"`
	MergedAT    time.Time      `json:"mergedAt"`
}

type ResponseReassign struct {
	PullRequest PullRequestDTO `json:"pullRequest"`
	ReplacedBy  string         `json:"replacedBy"`
}

type ResponseAddTeam struct {
	Team Team `json:"team"`
}

type ResponseSetIsActive struct {
	User User `json:"user"`
}

package models

type RequestCreateTeam struct {
	TeamName string `json:"teamName"`
	Members  []User `json:"members"`
}

func (r *RequestCreateTeam) ToTeam() Team {
	return Team{
		TeamName: r.TeamName,
		Members:  r.Members,
	}
}

type RequestSetIsActive struct {
	UserID   string `json:"userId"`
	IsActive bool   `json:"isActive"`
}

type RequestCreatePR struct {
	PullRequestID   string `json:"pullRequestId"`
	PullRequestName string `json:"pullRequestName"`
	AuthorID        string `json:"authorId"`
}

type RequestMergePR struct {
	PullRequestID string `json:"pullRequestId"`
}

type RequestReassignPR struct {
	PullRequestID  string `json:"pullRequestId"`
	OldReviewerID  string `json:"oldReviewerId"`
}

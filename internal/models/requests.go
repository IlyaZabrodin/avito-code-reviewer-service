package models

type RequestCreateTeam struct {
	TeamName string `json:"team_name"`
	Members  []User  `json:"members"`
}

func (r RequestCreateTeam) ToTeam() Team {
	return Team{
		TeamName: r.TeamName,
		Members:  r.Members,
	}
}

type RequestSetIsActive struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type RequestCreatePR struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type RequestMergePR struct {
	PullRequestID string `json:"pull_request_id"`
}

type RequestReassignPR struct {
	PullRequestID string `json:"pull_request_id"`
	OldReviewerID string `json:"old_reviewer_id"`
}

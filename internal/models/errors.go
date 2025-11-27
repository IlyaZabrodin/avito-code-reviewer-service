package models

import "errors"

var (
	ErrPullRequestAlreadyExists = errors.New("PULL REQUEST ALREADY EXISTS")
	ErrAuthorNotFoundOrInactive = errors.New("AUTHOR NOT FOUND OR INACTIVE")
	ErrPullRequestNotFound      = errors.New("PULL REQUEST NOT FOUND")
	ErrPullRequestAlreadyMerged = errors.New("PULL REQUEST ALREADY MERGED")
	ErrReviewerNotAssigned      = errors.New("REVIEWER IS NOT ASSIGNED TO PULL REQUEST")
	ErrNoReplacementFound       = errors.New("NO REPLACEMENT REVIEWER FOUND")
	ErrTeamAlreadyExists        = errors.New("TEAM ALREADY EXISTS")
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorStatsResponse struct {
	Error string `json:"error"`
}

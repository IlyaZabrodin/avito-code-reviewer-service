package controllers

import (
	"CodeRewievService/internal/models"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

type PullRequestController struct {
	service PullRequestService
	logger  *slog.Logger
}

func NewPullRequestController(service PullRequestService, logger *slog.Logger) *PullRequestController {
	return &PullRequestController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *PullRequestController) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req models.RequestCreatePR
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctrl.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		ctrl.logger.Error("Failed to decode request body", "error", err)
		return
	}

	pr, err := ctrl.service.Create(&models.PullRequest{
		PullRequestID:   req.PullRequestID,
		PullRequestName: req.PullRequestName,
		AuthorID:        req.AuthorID,
		Status:          "OPEN",
	})

	if errors.Is(err, models.ErrPRAlreadyExists) {
		ctrl.logger.Error("PR already exists", "prID", req.PullRequestID)
		ctrl.sendConflictResponse(w, "PR_EXISTS", "PR id already exists")
		return
	}

	if errors.Is(err, models.ErrAuthorNotFound) {
		ctrl.logger.Error("Author or team not found", "authorID", req.AuthorID)
		ctrl.sendNotFoundResponse(w)
		return
	}

	if err != nil {
		ctrl.logger.Error("Failed to create PR", "error", err, "prID", req.PullRequestID)
		ctrl.sendErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, models.ResponseCreatePR{
		PullRequest: pr.ToResponse(),
	}, http.StatusCreated)
}

func (ctrl *PullRequestController) MergePR(w http.ResponseWriter, r *http.Request) {
	var req models.RequestMergePR
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctrl.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		ctrl.logger.Error("Failed to decode request body", "error", err)
		return
	}

	pr, err := ctrl.service.Merge(req.PullRequestID)
	if errors.Is(err, models.ErrNotFound) {
		ctrl.logger.Error("PR not found for merge", "prID", req.PullRequestID)
		ctrl.sendNotFoundResponse(w)
		return
	}

	if err != nil {
		ctrl.logger.Error("Failed to merge PR", "error", err, "prID", req.PullRequestID)
		ctrl.sendErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, models.ResponseMerge{
		PullRequest: pr.ToResponse(),
		MergedAT:    time.Now(),
	}, http.StatusOK)
}

func (ctrl *PullRequestController) ReassignPR(w http.ResponseWriter, r *http.Request) {
	var req models.RequestReassignPR
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctrl.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		ctrl.logger.Error("Failed to decode request body", "error", err)
		return
	}

	pr, newUserID, err := ctrl.service.Reassign(req.PullRequestID, req.OldReviewerID)

	if errors.Is(err, models.ErrPRAlreadyMerged) {
		ctrl.logger.Error("Cannot reassign merged PR", "prID", req.PullRequestID)
		ctrl.sendConflictResponse(w, "PR_MERGED", "cannot reassign on merged PR")
		return
	}

	if errors.Is(err, models.ErrUserIsNotAssignedToPR) {
		ctrl.logger.Error("Reviewer not assigned to PR", "prID", req.PullRequestID, "reviewerID", req.OldReviewerID)
		ctrl.sendConflictResponse(w, "NOT_ASSIGNED", "reviewer not assigned to this PR")
		return
	}

	if errors.Is(err, models.ErrNotFound) {
		ctrl.logger.Error("PR or author not found", "prID", req.PullRequestID)
		ctrl.sendNotFoundResponse(w)
		return
	}

	if errors.Is(err, models.ErrNoReplacement) {
		ctrl.logger.Error("No replacement found for reviewer")
		ctrl.sendNotFoundResponse(w)
		return
	}

	if err != nil {
		ctrl.logger.Error("Failed to reassign PR", "error", err, "prID", req.PullRequestID)
		ctrl.sendErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, models.ResponseReassign{
		PullRequest: pr.ToResponse(),
		ReplacedBy:  newUserID,
	}, http.StatusOK)
}

func (ctrl *PullRequestController) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctrl.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (ctrl *PullRequestController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "ERROR",
		Message: message,
	}, statusCode)
}

func (ctrl *PullRequestController) sendNotFoundResponse(w http.ResponseWriter) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}, http.StatusNotFound)
}

func (ctrl *PullRequestController) sendConflictResponse(w http.ResponseWriter, code, message string) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    code,
		Message: message,
	}, http.StatusConflict)
}


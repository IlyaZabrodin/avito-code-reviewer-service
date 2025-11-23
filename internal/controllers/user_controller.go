package controllers

import (
	"CodeRewievService/internal/models"
	"encoding/json"
	"log/slog"
	"net/http"
)

type UserController struct {
	service UserService
	logger  *slog.Logger
}

func NewUserController(service UserService, logger *slog.Logger) *UserController {
	return &UserController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *UserController) SetUserIsActive(w http.ResponseWriter, r *http.Request) {
	var req models.RequestSetIsActive
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctrl.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		ctrl.logger.Error("Failed to decode request body", "error", err)
		return
	}

	user, err := ctrl.service.SetIsActive(&models.User{
		UserID:   req.UserID,
		IsActive: req.IsActive,
	})

	if err != nil {
		ctrl.logger.Error("Failed to set user active status", "error", err, "userID", req.UserID)
		ctrl.sendErrorResponse(w, "resource not found", http.StatusBadRequest)
		return
	}

	ctrl.sendJSONResponse(w, models.ResponseSetIsActive{User: *user}, http.StatusOK)
}

func (ctrl *UserController) GetUserReview(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		ctrl.sendErrorResponse(w, "user_id parameter is required", http.StatusBadRequest)
		return
	}

	review, err := ctrl.service.GetReview(userID)
	if err != nil {
		ctrl.logger.Error("Failed to get user review", "error", err, "userID", userID)
		ctrl.sendErrorResponse(w, "internal server error", http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, review, http.StatusOK)
}

func (ctrl *UserController) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctrl.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (ctrl *UserController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "NOT_FOUND",
		Message: message,
	}, statusCode)
}


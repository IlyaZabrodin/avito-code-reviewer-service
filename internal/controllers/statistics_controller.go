package controllers

import (
	"CodeRewievService/internal/models"
	"encoding/json"
	"log/slog"
	"net/http"
)

type StatisticsController struct {
	service StatisticsService
	logger  *slog.Logger
}

func NewStatisticsController(service StatisticsService, logger *slog.Logger) *StatisticsController {
	return &StatisticsController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *StatisticsController) GetAssignmentsStats(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		ctrl.sendErrorResponse(w, "team_name is required", http.StatusBadRequest)
		return
	}

	if err := ctrl.validateTeamExists(teamName); err != nil {
		ctrl.logger.Error("Team validation failed", "error", err, "team", teamName)
		if err == errTeamNotFound {
			ctrl.sendErrorResponse(w, "team not found", http.StatusNotFound)
		} else {
			ctrl.sendErrorResponse(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}

	stats, err := ctrl.service.GetAssignmentsStats(teamName)
	if err != nil {
		ctrl.logger.Error("Failed to get assignments stats", "error", err, "team", teamName)
		ctrl.sendErrorResponse(w, "failed to get assignments statistics", http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, stats, http.StatusOK)
}

var errTeamNotFound = &teamNotFoundError{}

type teamNotFoundError struct{}

func (e *teamNotFoundError) Error() string {
	return "team not found"
}

func (ctrl *StatisticsController) validateTeamExists(teamName string) error {
	exists, err := ctrl.service.TeamExists(teamName)
	if err != nil {
		return err
	}
	if !exists {
		return errTeamNotFound
	}
	return nil
}

func (ctrl *StatisticsController) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctrl.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (ctrl *StatisticsController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	ctrl.sendJSONResponse(w, models.ErrorStatsResponse{Error: message}, statusCode)
}

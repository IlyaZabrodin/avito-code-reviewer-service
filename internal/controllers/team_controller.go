package controllers

import (
	"CodeRewievService/internal/models"
	"encoding/json"
	"log/slog"
	"net/http"
)

type TeamController struct {
	service TeamService
	logger  *slog.Logger
}

func NewTeamController(service TeamService, logger *slog.Logger) *TeamController {
	return &TeamController{
		service: service,
		logger:  logger,
	}
}

func (ctrl *TeamController) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req models.RequestCreateTeam
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ctrl.sendErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		ctrl.logger.Error("Failed to decode request body", "error", err)
		return
	}

	existingTeam, err := ctrl.service.Get(req.TeamName)
	if err == nil && existingTeam != nil {
		ctrl.logger.Warn("Attempt to create existing team", "teamName", req.TeamName)
		ctrl.sendConflictResponse(w, "team_name already exists")
		return
	}

	team := req.ToTeam()

	if err := ctrl.service.Add(&team); err != nil {
		ctrl.logger.Error("Failed to create team", "error", err, "teamName", req.TeamName)
		ctrl.sendErrorResponse(w, "Failed to create team", http.StatusBadRequest)
		return
	}

	ctrl.sendJSONResponse(w, models.ResponseAddTeam{
		Team: team,
	}, http.StatusCreated)
}

func (ctrl *TeamController) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		ctrl.sendErrorResponse(w, "team_name parameter is required", http.StatusBadRequest)
		return
	}

	ctrl.logger.Info("Fetching team information", "teamName", teamName)

	team, err := ctrl.service.Get(teamName)
	if err != nil {
		ctrl.logger.Error("Team not found", "error", err, "teamName", teamName)
		ctrl.sendNotFoundResponse(w)
		return
	}

	ctrl.sendJSONResponse(w, team, http.StatusOK)
}

func (ctrl *TeamController) MassDeactivateTeamUsers(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		ctrl.sendErrorResponse(w, "Team name is required", http.StatusBadRequest)
		return
	}

	if err := ctrl.service.MassDeactivateTeamUsers(teamName); err != nil {
		ctrl.logger.Error("Failed to deactivate team users", "error", err, "teamName", teamName)
		ctrl.sendErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctrl.sendJSONResponse(w, map[string]string{
		"message": "Team users deactivated successfully",
		"team":    teamName,
	}, http.StatusOK)
}

func (ctrl *TeamController) sendJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		ctrl.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (ctrl *TeamController) sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "ERROR",
		Message: message,
	}, statusCode)
}

func (ctrl *TeamController) sendNotFoundResponse(w http.ResponseWriter) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "NOT_FOUND",
		Message: "resource not found",
	}, http.StatusBadRequest)
}

func (ctrl *TeamController) sendConflictResponse(w http.ResponseWriter, message string) {
	ctrl.sendJSONResponse(w, models.Error{
		Code:    "CONFLICT",
		Message: message,
	}, http.StatusConflict)
}

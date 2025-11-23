package repository

import (
	"CodeRewievService/internal/models"
	"errors"

	"gorm.io/gorm"
)

type TeamRepository struct {
	database *gorm.DB
}

func NewTeamRepository(database *gorm.DB) *TeamRepository {
	return &TeamRepository{
		database: database,
	}
}

func (r *TeamRepository) FindByName(teamName string) (*models.Team, error) {
	var team models.Team
	result := r.database.Where("team_name = ?", teamName).First(&team)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, gorm.ErrRecordNotFound
	}
	if result.Error != nil {
		return nil, result.Error
	}

	return &team, nil
}

func (r *TeamRepository) Create(team *models.Team) error {
	return r.database.Create(team).Error
}

func (r *TeamRepository) GetUsersByTeam(teamName string) ([]models.User, error) {
	var users []models.User
	err := r.database.Where("team_name = ?", teamName).Find(&users).Error
	return users, err
}

func (r *TeamRepository) CreateWithUsers(team *models.Team, users []models.User) error {
	return r.database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}

		return r.createTeamMembers(tx, team.TeamName, users)
	})
}

func (r *TeamRepository) DeactivateUsers(teamName string) (int64, error) {
	result := r.database.Model(&models.User{}).
		Where("team_name = ? AND is_active = ?", teamName, true).
		Update("is_active", false)

	return result.RowsAffected, result.Error
}

func (r *TeamRepository) GetOpenPRsByTeam(tx *gorm.DB, teamName string) ([]models.PullRequest, error) {
	var openPRs []models.PullRequest
	err := tx.
		Joins("JOIN users ON pull_requests.author_id = users.user_id").
		Where("users.team_name = ? AND pull_requests.status = ?", teamName, "OPEN").
		Find(&openPRs).Error

	return openPRs, err
}

func (r *TeamRepository) Transaction(fn func(*gorm.DB) error) error {
	return r.database.Transaction(fn)
}

func (r *TeamRepository) createTeamMembers(tx *gorm.DB, teamName string, users []models.User) error {
	for _, member := range users {
		user := &models.User{
			UserID:   member.UserID,
			Username: member.Username,
			TeamName: teamName,
			IsActive: member.IsActive,
		}

		if err := tx.Save(user).Error; err != nil {
			return err
		}
	}

	return nil
}

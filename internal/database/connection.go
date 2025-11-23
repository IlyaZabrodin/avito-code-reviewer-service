package database

import (
	"CodeRewievService/internal/models"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	defaultMigrationsPath = "file://migrations"
)

func InitializeConnection() *gorm.DB {
	loadEnvironmentVariables()

	dsn := buildConnectionString()
	database, err := establishConnection(dsn)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	if err := applyMigrations(database); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	syncGORMSchema(database)

	log.Println("Database connected and migrated successfully")
	return database
}

func loadEnvironmentVariables() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Failed to load .env file: %v", err)
	}
}

func buildConnectionString() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)
}

func establishConnection(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
}

func applyMigrations(database *gorm.DB) error {
	return RunMigrations(database)
}

func syncGORMSchema(database *gorm.DB) {
	if err := database.AutoMigrate(&models.Team{}, &models.User{}); err != nil {
		log.Printf("Warning: AutoMigrate failed (this is ok if tables already exist): %v", err)
	}
}

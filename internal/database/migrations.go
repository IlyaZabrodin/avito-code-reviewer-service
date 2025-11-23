package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/gorm"
)

func RunMigrations(database *gorm.DB) error {
	migrator, err := createMigrator(database)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	if err := migrator.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Println("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Migrations applied successfully")
	return nil
}

func GetMigrationVersion(database *gorm.DB) (uint, bool, error) {
	migrator, err := createMigrator(database)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create migrator: %w", err)
	}

	version, dirty, err := migrator.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			return 0, false, nil
		}
		return 0, false, err
	}

	return version, dirty, nil
}

func createMigrator(database *gorm.DB) (*migrate.Migrate, error) {
	sqlDB, err := database.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	migrationsPath := resolveMigrationsPath()

	return migrate.NewWithDatabaseInstance(
		migrationsPath,
		"postgres",
		driver,
	)
}

func resolveMigrationsPath() string {
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		return defaultMigrationsPath
	}

	return normalizeMigrationsPath(migrationsPath)
}

func normalizeMigrationsPath(path string) string {
	if filepath.IsAbs(path) {
		return "file://" + path
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Printf("Warning: Failed to get absolute path for migrations, using as is: %v", err)
		return "file://" + path
	}

	return "file://" + absPath
}

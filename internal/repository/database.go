// Package repository provides data access layer using GORM for database operations.
package repository

import (
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/aimd54/gitlab-reviewer-roulette/internal/config"
	"github.com/aimd54/gitlab-reviewer-roulette/internal/models"
	"github.com/aimd54/gitlab-reviewer-roulette/pkg/logger"
)

// DB holds the database connection.
type DB struct {
	*gorm.DB
}

// NewDB creates a new database connection.
func NewDB(cfg *config.PostgresConfig, log *logger.Logger) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	// Configure GORM logger
	var gormLogLevel gormlogger.LogLevel
	switch log.GetLogger().GetLevel() {
	case 0: // debug
		gormLogLevel = gormlogger.Info
	default:
		gormLogLevel = gormlogger.Warn
	}

	gormConfig := &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormLogLevel),
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool settings
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetime) * time.Second)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("Connected to PostgreSQL")

	return &DB{db}, nil
}

// AutoMigrate runs database migrations for all models.
func (db *DB) AutoMigrate() error {
	return db.DB.AutoMigrate(
		&models.User{},
		&models.OOOStatus{},
		&models.MRReview{},
		&models.ReviewerAssignment{},
		&models.ReviewMetrics{},
		&models.Badge{},
		&models.UserBadge{},
		&models.Configuration{},
	)
}

// Close closes the database connection.
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Health checks if the database is healthy.
func (db *DB) Health() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

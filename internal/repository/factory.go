package repository

import (
	"fmt"
	"strings"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

// New creates a new repository instance based on the database configuration
func New(ctx any, cfg config.DatabaseConfig, logger *logger.Logger) (domain.Repository, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	dbType := strings.ToLower(cfg.Type)
	logger.Info("Initializing repository", "type", dbType, "connection", cfg.ConnectionString)

	switch dbType {
	case "csv":
		return NewCSVRepository(cfg.ConnectionString, logger)
	case "googlesheet":
		return NewGoogleSheetRepository(ctx, cfg.ConnectionString, logger)
	case "postgresql":
		return NewPostgresRepository(ctx, cfg.ConnectionString, logger)
	case "mysql":
		return NewMySQLRepository(ctx, cfg.ConnectionString, logger)
	case "mongodb":
		return NewMongoDBRepository(ctx, cfg.ConnectionString, logger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
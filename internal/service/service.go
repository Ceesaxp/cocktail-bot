package service

import (
	"context"
	"errors"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/ratelimit"
	"github.com/ceesaxp/cocktail-bot/internal/repository"
	"github.com/ceesaxp/cocktail-bot/internal/utils"
)

// Service handles business logic for the bot
type Service struct {
	repo    domain.Repository
	limiter *ratelimit.Limiter
	logger  *logger.Logger
}

// New creates a new service instance
func New(ctx context.Context, cfg *config.Config, logger *logger.Logger) (*Service, error) {
	// Initialize repository based on config
	repo, err := repository.New(ctx, cfg.Database, logger)
	if err != nil {
		return nil, err
	}

	// Initialize rate limiter
	limiter := ratelimit.New(cfg.RateLimiting.RequestsPerMinute, cfg.RateLimiting.RequestsPerHour)

	return &Service{
		repo:    repo,
		limiter: limiter,
		logger:  logger,
	}, nil
}

// NewForTest creates a new service instance for testing
func NewForTest(repo domain.Repository, limiter *ratelimit.Limiter, logger *logger.Logger) *Service {
	return &Service{
		repo:    repo,
		limiter: limiter,
		logger:  logger,
	}
}

// CheckEmailStatus checks if an email exists in the database and if it has been redeemed
func (s *Service) CheckEmailStatus(ctx any, userID int64, email string) (status string, user *domain.User, err error) {
	// Apply rate limiting
	if !s.limiter.Allow(userID) {
		return "rate_limited", nil, nil
	}

	// Normalize email
	email = utils.NormalizeEmail(email)

	// Log the lookup
	s.logger.Info("Checking email status", "email", email, "user_id", userID)

	// Find user by email
	user, err = s.repo.FindByEmail(ctx, email)
	if err != nil {
		if err == domain.ErrUserNotFound {
			s.logger.Info("Email not found in database", "email", email)
			return "not_found", nil, nil
		}
		if err == domain.ErrDatabaseUnavailable {
			s.logger.Error("Database unavailable", "error", err)
			return "unavailable", nil, err
		}
		s.logger.Error("Error finding user", "email", email, "error", err)
		return "error", nil, err
	}

	// Check if already redeemed
	if user.IsRedeemed() {
		s.logger.Info("Email already redeemed", "email", email, "redeemed_at", user.AlreadyConsumed)
		return "redeemed", user, nil
	}

	s.logger.Info("Email eligible for redemption", "email", email)
	return "eligible", user, nil
}

// RedeemCocktail marks a user as having redeemed their cocktail
func (s *Service) RedeemCocktail(ctx any, userID int64, email string) (time.Time, error) {
	// Apply rate limiting (just to be extra safe, though the button should be gone)
	if !s.limiter.Allow(userID) {
		return time.Time{}, nil // No error because this is a rare edge case
	}

	// Normalize email
	email = utils.NormalizeEmail(email)

	// Find user by email
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Error finding user for redemption", "email", email, "error", err)
		return time.Time{}, err
	}

	// Check if already redeemed (double-check, should not happen)
	if user.IsRedeemed() {
		s.logger.Warn("Attempted to redeem already redeemed email", "email", email, "user_id", userID)
		return *user.AlreadyConsumed, nil
	}

	// Mark as redeemed
	user.Redeem()

	// Update user in repository
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("Error updating user for redemption", "email", email, "error", err)
		return time.Time{}, err
	}

	// Log the redemption
	s.logger.Info("Cocktail redeemed", "email", email, "user_id", userID, "time", *user.AlreadyConsumed)

	return *user.AlreadyConsumed, nil
}

// UpdateUser adds or updates a user in the database
func (s *Service) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	// Normalize email (in case it wasn't already)
	user.Email = utils.NormalizeEmail(user.Email)

	// Log the operation
	s.logger.Info("Updating user", "email", user.Email, "id", user.ID)

	// Update user in repository
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("Error updating user", "email", user.Email, "error", err)
		return err
	}

	return nil
}

// Close closes the service and its dependencies
func (s *Service) Close() error {
	return s.repo.Close()
}

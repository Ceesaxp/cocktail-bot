package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresRepository implements a PostgreSQL-backed repository
type PostgresRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository
func NewPostgresRepository(ctx context.Context, connStr string, logger *logger.Logger) (*PostgresRepository, error) {
	// Open database
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL database: %w", err)
	}

	// Check connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL database: %w", err)
	}

	repo := &PostgresRepository{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := repo.initSchema(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return repo, nil
}

// FindByEmail finds a user by email (case-insensitive)
func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Normalize email
	email = strings.ToLower(email)

	// Query user
	query := `SELECT id, email, date_added, already_consumed FROM users WHERE LOWER(email) = $1`

	var row *sql.Row
	var err error

	// Use context with timeout to handle database unavailability
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row = r.db.QueryRowContext(ctxWithTimeout, query, email)

	// Parse user
	var id, retrievedEmail string
	var dateAdded time.Time
	var alreadyConsumedTime sql.NullTime

	if err = row.Scan(&id, &retrievedEmail, &dateAdded, &alreadyConsumedTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}

		// Check for connection issues
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "connection") {
			r.logger.Error("Database connection issue", "error", err)
			return nil, domain.ErrDatabaseUnavailable
		}

		r.logger.Error("Error querying user", "email", email, "error", err)
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Parse already consumed date
	var alreadyConsumed *time.Time
	if alreadyConsumedTime.Valid {
		alreadyConsumed = &alreadyConsumedTime.Time
	}

	// Create user
	user := &domain.User{
		ID:              id,
		Email:           retrievedEmail,
		DateAdded:       dateAdded,
		AlreadyConsumed: alreadyConsumed,
	}

	return user, nil
}

// UpdateUser updates a user in the repository
func (r *PostgresRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	// Normalize email
	email := strings.ToLower(user.Email)

	// Update user
	query := `UPDATE users SET already_consumed = $1 WHERE LOWER(email) = $2`

	// Use context with timeout to handle database unavailability
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := r.db.ExecContext(ctxWithTimeout, query, user.AlreadyConsumed, email)
	if err != nil {
		// Check for connection issues
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "connection") {
			r.logger.Error("Database connection issue", "error", err)
			return domain.ErrDatabaseUnavailable
		}

		r.logger.Error("Error updating user", "email", email, "error", err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Check if user was found
	rows, err := result.RowsAffected()
	if err != nil {
		r.logger.Error("Error checking rows affected", "email", email, "error", err)
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// Close closes the repository
func (r *PostgresRepository) Close() error {
	return r.db.Close()
}

// initSchema initializes the database schema
func (r *PostgresRepository) initSchema(ctx context.Context) error {
	// Create users table
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT NOT NULL UNIQUE,
		date_added TIMESTAMP NOT NULL,
		already_consumed TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_email ON users (email);
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		r.logger.Error("Error initializing schema", "error", err)
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

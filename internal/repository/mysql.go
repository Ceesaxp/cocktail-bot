package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ceesaxp/cocktail-bot/internal/domain"
	"github.com/Ceesaxp/cocktail-bot/internal/logger"
	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

// MySQLRepository implements a MySQL-backed repository
type MySQLRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

// NewMySQLRepository creates a new MySQL repository
func NewMySQLRepository(ctx context.Context, connStr string, logger *logger.Logger) (*MySQLRepository, error) {
	// Open database
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open MySQL database: %w", err)
	}

	// Check connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping MySQL database: %w", err)
	}

	repo := &MySQLRepository{
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
func (r *MySQLRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Normalize email
	email = strings.ToLower(email)

	// Query user
	query := `SELECT id, email, date_added, already_consumed FROM users WHERE LOWER(email) = ?`

	// Use context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := r.db.QueryRowContext(ctxWithTimeout, query, email)

	// Parse user
	var id, retrievedEmail string
	var dateAdded time.Time
	var alreadyConsumedTime sql.NullTime

	if err := row.Scan(&id, &retrievedEmail, &dateAdded, &alreadyConsumedTime); err != nil {
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
func (r *MySQLRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	// Normalize email
	email := strings.ToLower(user.Email)

	// Update user
	query := `UPDATE users SET already_consumed = ? WHERE LOWER(email) = ?`

	// Use context with timeout
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
func (r *MySQLRepository) Close() error {
	return r.db.Close()
}

// initSchema initializes the database schema
func (r *MySQLRepository) initSchema(ctx context.Context) error {
	// Create users table
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(255) PRIMARY KEY,
		email VARCHAR(255) NOT NULL UNIQUE,
		date_added DATETIME NOT NULL,
		already_consumed DATETIME NULL
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

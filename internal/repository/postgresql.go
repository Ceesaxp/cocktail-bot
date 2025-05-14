package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	_ "github.com/lib/pq" // PostgreSQL driver
)

type PostgresRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

func NewPostgresRepository(ctx any, connectionString string, logger *logger.Logger) (*PostgresRepository, error) {
	if connectionString == "" {
		return nil, errors.New("connection string cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		logger.Error("Failed to connect to PostgreSQL", "error", err)
		return nil, err
	}

	// Check connection
	err = db.PingContext(context.Background())
	if err != nil {
		db.Close()
		logger.Error("Failed to ping PostgreSQL", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}

	// Create table if it doesn't exist
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			date_added TIMESTAMP NOT NULL,
			already_consumed TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`)
	if err != nil {
		db.Close()
		logger.Error("Failed to create table", "error", err)
		return nil, err
	}

	logger.Info("PostgreSQL Repository initialized")
	return &PostgresRepository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *PostgresRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.logger.Debug("Looking for email in PostgreSQL", "email", email)

	// Query for user
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := r.db.QueryRowContext(ctxWithTimeout, `
		SELECT id, email, date_added, already_consumed
		FROM users
		WHERE email = $1
	`, email)

	// Parse result
	var (
		id                string
		userEmail         string
		dateAdded         time.Time
		alreadyConsumedSQL sql.NullTime
	)

	err := row.Scan(&id, &userEmail, &dateAdded, &alreadyConsumedSQL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug("User not found in PostgreSQL", "email", email)
			return nil, domain.ErrUserNotFound
		}
		r.logger.Error("Error querying PostgreSQL", "error", err)
		return nil, err
	}

	// Create user
	user := &domain.User{
		ID:        id,
		Email:     userEmail,
		DateAdded: dateAdded,
	}

	// Handle already consumed
	if alreadyConsumedSQL.Valid {
		consumed := alreadyConsumedSQL.Time
		user.AlreadyConsumed = &consumed
	}

	r.logger.Debug("Found user in PostgreSQL", "email", email, "redeemed", user.IsRedeemed())
	return user, nil
}

func (r *PostgresRepository) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Updating user in PostgreSQL", "email", user.Email)

	// Prepare transaction
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctxWithTimeout, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", "error", err)
		return err
	}
	defer tx.Rollback()

	// Use upsert (INSERT ON CONFLICT UPDATE) for atomic operation
	query := `
		INSERT INTO users (id, email, date_added, already_consumed)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email)
		DO UPDATE SET
			id = EXCLUDED.id,
			date_added = EXCLUDED.date_added,
			already_consumed = EXCLUDED.already_consumed
	`

	var args []interface{}
	if user.AlreadyConsumed != nil {
		args = []interface{}{user.ID, user.Email, user.DateAdded, user.AlreadyConsumed}
	} else {
		args = []interface{}{user.ID, user.Email, user.DateAdded, nil}
	}

	_, err = tx.ExecContext(ctxWithTimeout, query, args...)
	if err != nil {
		r.logger.Error("Error upserting user", "error", err)
		return fmt.Errorf("failed to upsert user: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", "error", err)
		return err
	}

	r.logger.Debug("User updated in PostgreSQL", "email", user.Email)
	return nil
}

func (r *PostgresRepository) Close() error {
	r.logger.Debug("Closing PostgreSQL repository")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
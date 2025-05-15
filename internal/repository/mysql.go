package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	_ "github.com/go-sql-driver/mysql"
)

type MySQLRepository struct {
	db     *sql.DB
	logger *logger.Logger
}

func NewMySQLRepository(ctx any, connectionString string, logger *logger.Logger) (*MySQLRepository, error) {
	if connectionString == "" {
		return nil, errors.New("connection string cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Connect to MySQL
	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		logger.Error("Failed to connect to MySQL", "error", err)
		return nil, err
	}

	// Check connection
	err = db.PingContext(context.Background())
	if err != nil {
		db.Close()
		logger.Error("Failed to ping MySQL", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}

	// Create table if it doesn't exist
	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			date_added DATETIME NOT NULL,
			redeemed DATETIME
		);
		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`)
	if err != nil {
		db.Close()
		logger.Error("Failed to create table", "error", err)
		return nil, err
	}

	logger.Info("MySQL Repository initialized")
	return &MySQLRepository{
		db:     db,
		logger: logger,
	}, nil
}

func (r *MySQLRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.logger.Debug("Looking for email in MySQL", "email", email)

	// Query for user
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := r.db.QueryRowContext(ctxWithTimeout, `
		SELECT id, email, date_added, redeemed
		FROM users
		WHERE email = ?
	`, email)

	// Parse result
	var (
		id          string
		userEmail   string
		dateAdded   time.Time
		redeemedSQL sql.NullTime
	)

	err := row.Scan(&id, &userEmail, &dateAdded, &redeemedSQL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Debug("User not found in MySQL", "email", email)
			return nil, domain.ErrUserNotFound
		}
		r.logger.Error("Error querying MySQL", "error", err)
		return nil, err
	}

	// Create user
	user := &domain.User{
		ID:        id,
		Email:     userEmail,
		DateAdded: dateAdded,
	}

	// Handle redeemed
	if redeemedSQL.Valid {
		redeemed := redeemedSQL.Time
		user.Redeemed = &redeemed
	}

	r.logger.Debug("Found user in MySQL", "email", email, "redeemed", user.IsRedeemed())
	return user, nil
}

func (r *MySQLRepository) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Updating user in MySQL", "email", user.Email)

	// Prepare transaction
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := r.db.BeginTx(ctxWithTimeout, nil)
	if err != nil {
		r.logger.Error("Failed to begin transaction", "error", err)
		return err
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Error("Failed to rollback transaction", "error", err)
		}
	}()

	// Check if user exists
	var exists bool
	err = tx.QueryRowContext(ctxWithTimeout, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ?)", user.Email).Scan(&exists)
	if err != nil {
		r.logger.Error("Error checking if user exists", "error", err)
		return err
	}

	if exists {
		// Update existing user
		var query string
		var args []interface{}

		if user.Redeemed != nil {
			query = "UPDATE users SET id = ?, date_added = ?, redeemed = ? WHERE email = ?"
			args = []interface{}{user.ID, user.DateAdded, user.Redeemed, user.Email}
		} else {
			query = "UPDATE users SET id = ?, date_added = ?, redeemed = NULL WHERE email = ?"
			args = []interface{}{user.ID, user.DateAdded, user.Email}
		}

		_, err = tx.ExecContext(ctxWithTimeout, query, args...)
	} else {
		// Insert new user
		var query string
		var args []interface{}

		if user.Redeemed != nil {
			query = "INSERT INTO users(id, email, date_added, redeemed) VALUES(?, ?, ?, ?)"
			args = []interface{}{user.ID, user.Email, user.DateAdded, user.Redeemed}
		} else {
			query = "INSERT INTO users(id, email, date_added, redeemed) VALUES(?, ?, ?, NULL)"
			args = []interface{}{user.ID, user.Email, user.DateAdded}
		}

		_, err = tx.ExecContext(ctxWithTimeout, query, args...)
	}

	if err != nil {
		r.logger.Error("Error updating user", "error", err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		r.logger.Error("Failed to commit transaction", "error", err)
		return err
	}

	r.logger.Debug("User updated in MySQL", "email", user.Email)
	return nil
}

func (r *MySQLRepository) Close() error {
	r.logger.Debug("Closing MySQL repository")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
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
			redeemed TIMESTAMP
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
		SELECT id, email, date_added, redeemed
		FROM users
		WHERE email = $1
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

	// Handle redeemed
	if redeemedSQL.Valid {
		redeemed := redeemedSQL.Time
		user.Redeemed = &redeemed
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
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			r.logger.Error("Failed to rollback transaction", "error", err)
		}
	}()

	// Use upsert (INSERT ON CONFLICT UPDATE) for atomic operation
	query := `
		INSERT INTO users (id, email, date_added, redeemed)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (email)
		DO UPDATE SET
			id = EXCLUDED.id,
			date_added = EXCLUDED.date_added,
			redeemed = EXCLUDED.redeemed
	`

	var args []interface{}
	if user.Redeemed != nil {
		args = []interface{}{user.ID, user.Email, user.DateAdded, user.Redeemed}
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

// AddUser adds a new user to the database
func (r *PostgresRepository) AddUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Adding user to PostgreSQL", "email", user.Email)

	// Check if user already exists
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var exists bool
	err := r.db.QueryRowContext(ctxWithTimeout, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", user.Email).Scan(&exists)
	if err != nil {
		r.logger.Error("Error checking if user exists", "error", err)
		return err
	}
	
	if exists {
		r.logger.Debug("User already exists in PostgreSQL", "email", user.Email)
		return errors.New("user already exists")
	}

	// Insert new user
	query := `INSERT INTO users (id, email, date_added, redeemed) VALUES ($1, $2, $3, $4)`
	
	var args []interface{}
	if user.Redeemed != nil {
		args = []interface{}{user.ID, user.Email, user.DateAdded, user.Redeemed}
	} else {
		args = []interface{}{user.ID, user.Email, user.DateAdded, nil}
	}
	
	_, err = r.db.ExecContext(ctxWithTimeout, query, args...)
	if err != nil {
		r.logger.Error("Error adding user", "error", err)
		return fmt.Errorf("failed to add user: %w", err)
	}
	
	r.logger.Debug("User added to PostgreSQL", "email", user.Email)
	return nil
}

// GetReport retrieves users based on the report parameters
func (r *PostgresRepository) GetReport(ctx any, params domain.ReportParams) ([]*domain.User, error) {
	r.logger.Debug("Generating report from PostgreSQL", "type", params.Type, "from", params.From, "to", params.To)

	var query string
	var args []interface{}

	// Build different queries based on report type
	switch params.Type {
	case domain.ReportTypeRedeemed:
		// Only get users who have redeemed within the date range
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users 
			WHERE date_added >= $1 AND date_added <= $2 
			AND redeemed IS NOT NULL
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	case domain.ReportTypeAdded:
		// Get users added within the date range
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users 
			WHERE date_added >= $1 AND date_added <= $2
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	case domain.ReportTypeAll:
		// Get all users
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users
			WHERE date_added >= $1 AND date_added <= $2
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	default:
		return nil, fmt.Errorf("invalid report type: %s", params.Type)
	}

	// Execute query with timeout
	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctxWithTimeout, query, args...)
	if err != nil {
		r.logger.Error("Failed to execute report query", "error", err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	// Process results
	var users []*domain.User
	for rows.Next() {
		var (
			id            string
			email         string
			dateAdded     time.Time
			redeemedTime  sql.NullTime
		)

		if err := rows.Scan(&id, &email, &dateAdded, &redeemedTime); err != nil {
			r.logger.Error("Error scanning row", "error", err)
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		// Create user object
		user := &domain.User{
			ID:        id,
			Email:     email,
			DateAdded: dateAdded,
		}

		// Handle redeemed time
		if redeemedTime.Valid {
			t := redeemedTime.Time
			user.Redeemed = &t
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating rows", "error", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	r.logger.Info("Report generated from PostgreSQL", "type", params.Type, "count", len(users))
	return users, nil
}

func (r *PostgresRepository) Close() error {
	r.logger.Debug("Closing PostgreSQL repository")
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
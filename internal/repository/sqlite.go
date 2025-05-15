package repository

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	_ "github.com/mattn/go-sqlite3" // SQLite driver
)

// SQLiteRepository implements the domain.Repository interface for SQLite
type SQLiteRepository struct {
	db     *sql.DB
	logger *logger.Logger
	mu     sync.Mutex // For thread safety
}

// OpenSQLiteForTesting opens the SQLite database for testing purposes
// This is only for testing and shouldn't be used in production code
func OpenSQLiteForTesting(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	
	// Create table if not exists
	if err := createTableIfNotExists(db); err != nil {
		db.Close()
		return nil, err
	}
	
	return db, nil
}

// NewSQLiteRepository creates a new SQLite repository
func NewSQLiteRepository(dbPath string, logger *logger.Logger) (domain.Repository, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	// Open SQLite database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to SQLite database: %w", err)
	}

	// Create table if not exists
	if err := createTableIfNotExists(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	logger.Info("SQLite repository initialized", "path", dbPath)

	return &SQLiteRepository{
		db:     db,
		logger: logger,
	}, nil
}

// createTableIfNotExists creates the users table if it doesn't exist
func createTableIfNotExists(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		date_added TIMESTAMP NOT NULL,
		redeemed TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
	`
	_, err := db.Exec(query)
	return err
}

// FindByEmail looks up a user by email
func (r *SQLiteRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	query := `SELECT id, email, date_added, redeemed FROM users WHERE LOWER(email) = LOWER(?)`
	row := r.db.QueryRow(query, email)

	var (
		id              string
		dbEmail         string
		dateAdded       time.Time
		alreadyConsumed sql.NullTime
	)

	err := row.Scan(&id, &dbEmail, &dateAdded, &alreadyConsumed)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		if r.logger != nil {
			r.logger.Error("Error querying user", "email", email, "error", err)
		}
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Convert nullable time to pointer
	var consumedTime *time.Time
	if alreadyConsumed.Valid {
		consumedTime = &alreadyConsumed.Time
	}

	return &domain.User{
		ID:              id,
		Email:           dbEmail,
		DateAdded:       dateAdded,
		Redeemed: consumedTime,
	}, nil
}

// UpdateUser updates an existing user information (primarily for marking cocktail as redeemed)
func (r *SQLiteRepository) UpdateUser(ctx any, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var consumedTime sql.NullTime
	if user.Redeemed != nil {
		consumedTime = sql.NullTime{
			Time:  *user.Redeemed,
			Valid: true,
		}
	}

	query := `UPDATE users SET redeemed = ? WHERE id = ?`
	result, err := r.db.Exec(query, consumedTime, user.ID)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Error updating user", "id", user.ID, "error", err)
		}
		return fmt.Errorf("database error: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		if r.logger != nil {
			r.logger.Error("Error getting rows affected", "error", err)
		}
		return nil // Ignore this error
	}

	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

// AddUser adds a new user to the database
func (r *SQLiteRepository) AddUser(ctx any, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Prepare consumed time for SQL
	var consumedTime sql.NullTime
	if user.Redeemed != nil {
		consumedTime = sql.NullTime{
			Time:  *user.Redeemed,
			Valid: true,
		}
	}

	// Insert new user
	query := `INSERT INTO users (id, email, date_added, redeemed) VALUES (?, ?, ?, ?)`
	_, err := r.db.Exec(query, user.ID, user.Email, user.DateAdded, consumedTime)
	if err != nil {
		r.logger.Error("Error adding user", "email", user.Email, "error", err)
		return fmt.Errorf("database error: %w", err)
	}

	r.logger.Debug("User added to SQLite", "email", user.Email, "id", user.ID)
	return nil
}

// GetReport retrieves users based on the report parameters
func (r *SQLiteRepository) GetReport(ctx any, params domain.ReportParams) ([]*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.logger.Debug("Generating report from SQLite", "type", params.Type, "from", params.From, "to", params.To)

	var query string
	var args []interface{}

	// Build different queries based on report type
	switch params.Type {
	case domain.ReportTypeRedeemed:
		// Only get users who have redeemed within the date range
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users
			WHERE date_added >= ? AND date_added <= ?
			AND redeemed IS NOT NULL
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	case domain.ReportTypeAdded:
		// Get users added within the date range
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users
			WHERE date_added >= ? AND date_added <= ?
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	case domain.ReportTypeAll:
		// Get all users
		query = `
			SELECT id, email, date_added, redeemed 
			FROM users
			WHERE date_added >= ? AND date_added <= ?
			ORDER BY date_added DESC
		`
		args = []interface{}{params.From, params.To}
	default:
		return nil, fmt.Errorf("invalid report type: %s", params.Type)
	}

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.logger.Error("Error querying for report", "error", err)
		return nil, fmt.Errorf("database error: %w", err)
	}
	defer rows.Close()

	// Parse results
	var users []*domain.User
	for rows.Next() {
		var (
			id              string
			email           string
			dateAdded       time.Time
			alreadyConsumed sql.NullTime
		)

		if err := rows.Scan(&id, &email, &dateAdded, &alreadyConsumed); err != nil {
			r.logger.Error("Error scanning row", "error", err)
			return nil, fmt.Errorf("error scanning row: %w", err)
		}

		// Convert nullable time to pointer
		var consumedTime *time.Time
		if alreadyConsumed.Valid {
			consumedTime = &alreadyConsumed.Time
		}

		users = append(users, &domain.User{
			ID:        id,
			Email:     email,
			DateAdded: dateAdded,
			Redeemed:  consumedTime,
		})
	}

	if err := rows.Err(); err != nil {
		r.logger.Error("Error iterating rows", "error", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	r.logger.Info("Report generated from SQLite", "type", params.Type, "count", len(users))
	return users, nil
}

// Close closes the database connection
func (r *SQLiteRepository) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
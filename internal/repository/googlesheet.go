package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleSheetRepository implements a Google Sheets-backed repository
type GoogleSheetRepository struct {
	sheetID     string
	sheetRange  string
	service     *sheets.Service
	cache       map[string]*domain.User
	cacheMutex  sync.RWMutex
	lastRefresh time.Time
	logger      *logger.Logger
}

// NewGoogleSheetRepository creates a new Google Sheets repository
func NewGoogleSheetRepository(ctx context.Context, connStr string, logger *logger.Logger) (*GoogleSheetRepository, error) {
	// Parse connection string (format: "credentials_file:sheet_id:sheet_range")
	parts := strings.Split(connStr, ":")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid Google Sheets connection string format")
	}

	credentialsFile := parts[0]
	sheetID := parts[1]
	sheetRange := parts[2]

	// Load credentials
	credentials, err := os.ReadFile(credentialsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	// Create OAuth config
	config, err := google.JWTConfigFromJSON(credentials, sheets.SpreadsheetsReadonlyScope, sheets.SpreadsheetsScope)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Create service
	service, err := sheets.NewService(ctx, option.WithHTTPClient(config.Client(ctx)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Sheets service: %w", err)
	}

	repo := &GoogleSheetRepository{
		sheetID:    sheetID,
		sheetRange: sheetRange,
		service:    service,
		cache:      make(map[string]*domain.User),
		logger:     logger,
	}

	// Load initial data
	if err := repo.refreshCache(ctx); err != nil {
		return nil, err
	}

	return repo, nil
}

// FindByEmail finds a user by email (case-insensitive)
func (r *GoogleSheetRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Normalize email
	email = strings.ToLower(email)

	// Check if cache needs refreshing (refresh every 5 minutes)
	if time.Since(r.lastRefresh) > 5*time.Minute {
		if err := r.refreshCache(ctx); err != nil {
			// If we can't refresh, but have a cached user, return the cached user
			r.cacheMutex.RLock()
			cachedUser, exists := r.cache[email]
			r.cacheMutex.RUnlock()

			if exists {
				r.logger.Warn("Using cached user due to refresh failure", "email", email)
				return cachedUser, nil
			}

			// Otherwise, return error
			return nil, err
		}
	}

	// Check cache
	r.cacheMutex.RLock()
	user, exists := r.cache[email]
	r.cacheMutex.RUnlock()

	if !exists {
		return nil, domain.ErrUserNotFound
	}

	// Return a copy to prevent mutation
	userCopy := *user
	if user.AlreadyConsumed != nil {
		timeCopy := *user.AlreadyConsumed
		userCopy.AlreadyConsumed = &timeCopy
	}

	return &userCopy, nil
}

// UpdateUser updates a user in the repository
func (r *GoogleSheetRepository) UpdateUser(ctx context.Context, user *domain.User) error {
	// Normalize email
	email := strings.ToLower(user.Email)

	// Find row index
	rowIndex := -1
	r.cacheMutex.RLock()

	// Try to find the email in the cache first
	_, exists := r.cache[email]
	if !exists {
		r.cacheMutex.RUnlock()
		return domain.ErrUserNotFound
	}
	r.cacheMutex.RUnlock()

	// Refresh cache to get the latest sheet data
	if err := r.refreshCache(ctx); err != nil {
		return err
	}

	// Now find the row index in the sheet
	resp, err := r.service.Spreadsheets.Values.Get(r.sheetID, r.sheetRange).Context(ctx).Do()
	if err != nil {
		r.logger.Error("Error fetching sheet", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	if len(resp.Values) == 0 {
		return fmt.Errorf("empty sheet")
	}

	// Find column indices
	var idIdx, emailIdx, dateAddedIdx, alreadyConsumedIdx int
	idIdx, emailIdx, dateAddedIdx, alreadyConsumedIdx = -1, -1, -1, -1

	header := resp.Values[0]
	for i, col := range header {
		colName, ok := col.(string)
		if !ok {
			continue
		}

		switch strings.ToLower(colName) {
		case "id":
			idIdx = i
		case "email":
			emailIdx = i
		case "date added", "dateadded":
			dateAddedIdx = i
		case "already consumed", "alreadyconsumed":
			alreadyConsumedIdx = i
		}
	}

	if idIdx == -1 || emailIdx == -1 || dateAddedIdx == -1 || alreadyConsumedIdx == -1 {
		return fmt.Errorf("missing required columns in sheet header")
	}

	// Find row with matching email
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) <= emailIdx {
			continue // Skip rows that don't have enough columns
		}

		rowEmail, ok := row[emailIdx].(string)
		if !ok {
			continue
		}

		if strings.EqualFold(rowEmail, email) {
			rowIndex = i
			break
		}
	}

	if rowIndex == -1 {
		return domain.ErrUserNotFound
	}

	// Update value in sheet
	rowRange := fmt.Sprintf("%s!%s%d", strings.Split(r.sheetRange, "!")[0], columnLetter(alreadyConsumedIdx), rowIndex+1)

	var value string
	if user.AlreadyConsumed != nil {
		value = user.AlreadyConsumed.Format(time.RFC3339)
	} else {
		value = ""
	}

	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{{
			value,
		}},
	}

	_, err = r.service.Spreadsheets.Values.Update(r.sheetID, rowRange, valueRange).ValueInputOption("RAW").Context(ctx).Do()
	if err != nil {
		r.logger.Error("Error updating sheet", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	// Update cache
	r.cacheMutex.Lock()
	r.cache[email] = user
	r.cacheMutex.Unlock()

	return nil
}

// Close closes the repository
func (r *GoogleSheetRepository) Close() error {
	// No need to close anything for Google Sheets
	return nil
}

// refreshCache refreshes the cache from the Google Sheet
func (r *GoogleSheetRepository) refreshCache(ctx context.Context) error {
	// Fetch data from sheet
	resp, err := r.service.Spreadsheets.Values.Get(r.sheetID, r.sheetRange).Context(ctx).Do()
	if err != nil {
		// Check for network/connection issues
		if errors.Is(err, context.DeadlineExceeded) || isHTTPError(err) {
			r.logger.Error("Google Sheets API connection issue", "error", err)
			return domain.ErrDatabaseUnavailable
		}

		r.logger.Error("Error fetching sheet", "error", err)
		return fmt.Errorf("failed to fetch sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		return fmt.Errorf("empty sheet")
	}

	// Find column indices
	var idIdx, emailIdx, dateAddedIdx, alreadyConsumedIdx int
	idIdx, emailIdx, dateAddedIdx, alreadyConsumedIdx = -1, -1, -1, -1

	header := resp.Values[0]
	for i, col := range header {
		colName, ok := col.(string)
		if !ok {
			continue
		}

		switch strings.ToLower(colName) {
		case "id":
			idIdx = i
		case "email":
			emailIdx = i
		case "date added", "dateadded":
			dateAddedIdx = i
		case "already consumed", "alreadyconsumed":
			alreadyConsumedIdx = i
		}
	}

	if idIdx == -1 || emailIdx == -1 || dateAddedIdx == -1 || alreadyConsumedIdx == -1 {
		return fmt.Errorf("missing required columns in sheet header")
	}

	// Parse rows
	newCache := make(map[string]*domain.User)

	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) <= max(idIdx, emailIdx, dateAddedIdx, alreadyConsumedIdx) {
			continue // Skip rows that don't have enough columns
		}

		// Parse values
		id, ok := row[idIdx].(string)
		if !ok || id == "" {
			continue
		}

		email, ok := row[emailIdx].(string)
		if !ok || email == "" {
			continue
		}
		email = strings.ToLower(email)

		dateAddedStr, ok := row[dateAddedIdx].(string)
		if !ok || dateAddedStr == "" {
			continue
		}

		dateAdded, err := parseDateTime(dateAddedStr)
		if err != nil {
			r.logger.Warn("Failed to parse date added", "email", email, "value", dateAddedStr, "error", err)
			dateAdded = time.Now() // Fallback to current time
		}

		var alreadyConsumed *time.Time
		if len(row) > alreadyConsumedIdx {
			alreadyConsumedStr, ok := row[alreadyConsumedIdx].(string)
			if ok && alreadyConsumedStr != "" {
				consumed, err := parseDateTime(alreadyConsumedStr)
				if err != nil {
					r.logger.Warn("Failed to parse already consumed", "email", email, "value", alreadyConsumedStr, "error", err)
				} else {
					alreadyConsumed = &consumed
				}
			}
		}

		// Create user
		user := &domain.User{
			ID:              id,
			Email:           email,
			DateAdded:       dateAdded,
			AlreadyConsumed: alreadyConsumed,
		}

		// Add to new cache
		newCache[email] = user
	}

	// Update cache
	r.cacheMutex.Lock()
	r.cache = newCache
	r.lastRefresh = time.Now()
	r.cacheMutex.Unlock()

	r.logger.Info("Refreshed cache from Google Sheet", "count", len(newCache))
	return nil
}

// parseDateTime parses a date-time string using various formats
func parseDateTime(str string) (time.Time, error) {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
		"01/02/2006",
		"2006-01-02",
	}

	for _, format := range formats {
		t, err := time.Parse(format, str)
		if err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unknown date-time format: %s", str)
}

// columnLetter converts a column index to a column letter (A, B, C, ...)
func columnLetter(index int) string {
	result := ""
	for {
		remainder := index % 26
		result = string(rune('A'+remainder)) + result
		index = index / 26
		if index == 0 {
			break
		}
		index-- // Adjust for 1-indexed
	}
	return result
}

// max returns the maximum of multiple integers
func max(values ...int) int {
	result := values[0]
	for _, v := range values[1:] {
		if v > result {
			result = v
		}
	}
	return result
}

// isHTTPError checks if an error is an HTTP error
func isHTTPError(err error) bool {
	// Check if it's a transport error
	var transportErr *googleapi.Error
	if errors.As(err, &transportErr) {
		return true
	}

	// Otherwise check if it contains common network error strings
	errStr := err.Error()
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "reset")
}

// googleapi is a stub for the google API error type
type googleapi struct{}

// Error is the error interface implementation for googleapi.Error
type googleapi.Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Errors  []struct {
		Domain  string `json:"domain"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"errors"`
}

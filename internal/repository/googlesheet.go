package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

type GoogleSheetRepository struct {
	service       *sheets.Service
	spreadsheetID string
	sheetName     string
	logger        *logger.Logger
}

func NewGoogleSheetRepository(ctx any, connectionString string, logger *logger.Logger) (*GoogleSheetRepository, error) {
	if connectionString == "" {
		return nil, errors.New("connection string cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}

	// Parse connection string (format: credentialsPath|spreadsheetID|sheetName)
	parts := parseConnectionString(connectionString)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid connection string format: %s", connectionString)
	}

	credentialsPath := parts[0]
	spreadsheetID := parts[1]
	sheetName := parts[2]

	// Initialize Google Sheets API
	service, err := sheets.NewService(context.Background(), option.WithCredentialsFile(credentialsPath))
	if err != nil {
		logger.Error("Failed to create Google Sheets service", "error", err)
		return nil, err
	}

	logger.Info("Google Sheets Repository initialized", "spreadsheetID", spreadsheetID, "sheet", sheetName)
	return &GoogleSheetRepository{
		service:       service,
		spreadsheetID: spreadsheetID,
		sheetName:     sheetName,
		logger:        logger,
	}, nil
}

func (r *GoogleSheetRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.logger.Debug("Looking for email in Google Sheets", "email", email)

	// Define the range to read
	readRange := fmt.Sprintf("%s!A:D", r.sheetName)

	// Read data from sheet
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(context.Background()).Do()
	if err != nil {
		r.logger.Error("Failed to read Google Sheet", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}

	if len(resp.Values) == 0 {
		r.logger.Debug("Sheet is empty", "sheet", r.sheetName)
		return nil, domain.ErrUserNotFound
	}

	// Skip header and look for email
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) >= 2 {
			// Convert interface{} to string safely
			rowEmail, ok := row[1].(string)
			if ok && rowEmail == email {
				// Found the user
				user := &domain.User{
					Email: email,
				}

				// Get ID if available
				if len(row) >= 1 {
					if id, ok := row[0].(string); ok {
						user.ID = id
					}
				}

				// Parse DateAdded
				if len(row) >= 3 {
					if dateStr, ok := row[2].(string); ok && dateStr != "" {
						dateAdded, err := time.Parse(time.RFC3339, dateStr)
						if err == nil {
							user.DateAdded = dateAdded
						}
					}
				}

				// Parse Redeemed
				if len(row) >= 4 {
					if redeemedStr, ok := row[3].(string); ok && redeemedStr != "" {
						redeemed, err := time.Parse(time.RFC3339, redeemedStr)
						if err == nil {
							user.Redeemed = &redeemed
						}
					}
				}

				r.logger.Debug("Found user in Google Sheets", "email", email, "redeemed", user.IsRedeemed())
				return user, nil
			}
		}
	}

	r.logger.Debug("User not found in Google Sheets", "email", email)
	return nil, domain.ErrUserNotFound
}

func (r *GoogleSheetRepository) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Updating user in Google Sheets", "email", user.Email)

	// Define the range to read
	readRange := fmt.Sprintf("%s!A:D", r.sheetName)

	// Read data from sheet to find the row
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(context.Background()).Do()
	if err != nil {
		r.logger.Error("Failed to read Google Sheet for update", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	// Find the row with the user's email
	rowIndex := -1
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) >= 2 {
			if rowEmail, ok := row[1].(string); ok && rowEmail == user.Email {
				rowIndex = i + 1 // 1-based index for API
				break
			}
		}
	}

	// Prepare the updated/new row data
	var values []interface{}
	values = append(values, user.ID)
	values = append(values, user.Email)
	values = append(values, user.DateAdded.Format(time.RFC3339))

	if user.Redeemed != nil {
		values = append(values, user.Redeemed.Format(time.RFC3339))
	} else {
		values = append(values, "")
	}

	var updateRange string
	var valueRange sheets.ValueRange

	if rowIndex > 0 {
		// Update existing row
		updateRange = fmt.Sprintf("%s!A%d:D%d", r.sheetName, rowIndex, rowIndex)
		valueRange = sheets.ValueRange{
			Values: [][]interface{}{values},
		}

		_, err = r.service.Spreadsheets.Values.Update(r.spreadsheetID, updateRange, &valueRange).
			ValueInputOption("RAW").Context(context.Background()).Do()
	} else {
		// Append new row
		updateRange = fmt.Sprintf("%s!A:D", r.sheetName)
		valueRange = sheets.ValueRange{
			Values: [][]interface{}{values},
		}

		_, err = r.service.Spreadsheets.Values.Append(r.spreadsheetID, updateRange, &valueRange).
			ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(context.Background()).Do()
	}

	if err != nil {
		r.logger.Error("Failed to update Google Sheet", "error", err)
		return err
	}

	r.logger.Debug("User updated in Google Sheets", "email", user.Email)
	return nil
}

// AddUser adds a new user to the Google Sheet
func (r *GoogleSheetRepository) AddUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Adding user to Google Sheets", "email", user.Email)

	// Define the range to read
	readRange := fmt.Sprintf("%s!A:D", r.sheetName)

	// Read data from sheet to check for duplicates
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(context.Background()).Do()
	if err != nil {
		r.logger.Error("Failed to read Google Sheet for add", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	// Check if user already exists
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) >= 2 {
			if rowEmail, ok := row[1].(string); ok && rowEmail == user.Email {
				r.logger.Debug("User already exists in Google Sheets", "email", user.Email)
				return errors.New("user already exists")
			}
		}
	}

	// Prepare the new row data
	var values []interface{}
	values = append(values, user.ID)
	values = append(values, user.Email)
	values = append(values, user.DateAdded.Format(time.RFC3339))

	if user.Redeemed != nil {
		values = append(values, user.Redeemed.Format(time.RFC3339))
	} else {
		values = append(values, "")
	}

	// Append new row
	updateRange := fmt.Sprintf("%s!A:D", r.sheetName)
	valueRange := sheets.ValueRange{
		Values: [][]interface{}{values},
	}

	_, err = r.service.Spreadsheets.Values.Append(r.spreadsheetID, updateRange, &valueRange).
		ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(context.Background()).Do()

	if err != nil {
		r.logger.Error("Failed to add user to Google Sheet", "error", err)
		return err
	}

	r.logger.Debug("User added to Google Sheets", "email", user.Email)
	return nil
}

// GetReport retrieves users based on the report parameters
func (r *GoogleSheetRepository) GetReport(ctx any, params domain.ReportParams) ([]*domain.User, error) {
	r.logger.Debug("Generating report from Google Sheets", "type", params.Type, "from", params.From, "to", params.To)

	// Define the range to read
	readRange := fmt.Sprintf("%s!A:D", r.sheetName)

	// Read data from sheet
	resp, err := r.service.Spreadsheets.Values.Get(r.spreadsheetID, readRange).Context(context.Background()).Do()
	if err != nil {
		r.logger.Error("Failed to read Google Sheet for report", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}

	if len(resp.Values) == 0 {
		r.logger.Debug("Sheet is empty", "sheet", r.sheetName)
		return []*domain.User{}, nil
	}

	var users []*domain.User

	// Skip header and process rows
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		if len(row) < 2 {
			continue // Skip invalid rows
		}

		// Extract user data
		var user domain.User

		// Get ID
		if id, ok := row[0].(string); ok {
			user.ID = id
		} else {
			continue // Skip row without ID
		}

		// Get Email
		if email, ok := row[1].(string); ok {
			user.Email = email
		} else {
			continue // Skip row without email
		}

		// Parse DateAdded
		var dateAdded time.Time
		if len(row) >= 3 {
			if dateStr, ok := row[2].(string); ok && dateStr != "" {
				parsedDate, err := time.Parse(time.RFC3339, dateStr)
				if err == nil {
					dateAdded = parsedDate
					user.DateAdded = parsedDate
				} else {
					continue // Skip row with invalid date
				}
			} else {
				continue // Skip row without date
			}
		} else {
			continue // Skip row without date
		}

		// Parse Redeemed
		if len(row) >= 4 {
			if redeemedStr, ok := row[3].(string); ok && redeemedStr != "" {
				redeemed, err := time.Parse(time.RFC3339, redeemedStr)
				if err == nil {
					user.Redeemed = &redeemed
				}
			}
		}

		// Apply date range filter
		if !dateAdded.Before(params.From) && !dateAdded.After(params.To) {
			// Apply report type filter
			switch params.Type {
			case domain.ReportTypeRedeemed:
				// Include only redeemed records
				if user.Redeemed != nil {
					users = append(users, &user)
				}
			case domain.ReportTypeAdded:
				// Include all records within the date range
				users = append(users, &user)
			case domain.ReportTypeAll:
				// Include all records
				users = append(users, &user)
			}
		}
	}

	r.logger.Info("Report generated from Google Sheets", "type", params.Type, "count", len(users))
	return users, nil
}

func (r *GoogleSheetRepository) Close() error {
	r.logger.Debug("Closing Google Sheets repository")
	// No explicit close method for Google Sheets API client
	return nil
}

// Helper function to parse connection string
func parseConnectionString(connStr string) []string {
	// Simple parsing logic - in a real implementation, this would be more robust
	// Format: credentialsPath|spreadsheetID|sheetName
	parts := make([]string, 3)

	// Split by pipe
	split := make([]string, 0)
	current := ""
	for i := 0; i < len(connStr); i++ {
		if connStr[i] == '|' {
			split = append(split, current)
			current = ""
		} else {
			current += string(connStr[i])
		}
	}
	if current != "" {
		split = append(split, current)
	}

	// Copy to parts array
	for i := 0; i < len(split) && i < 3; i++ {
		parts[i] = split[i]
	}

	return parts
}

package repository

import (
	"encoding/csv"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

type CSVRepository struct {
	filePath string
	logger   *logger.Logger
}

func NewCSVRepository(filePath string, logger *logger.Logger) (*CSVRepository, error) {
	if filePath == "" {
		return nil, errors.New("file path cannot be empty")
	}
	if logger == nil {
		return nil, errors.New("logger cannot be nil")
	}
	
	// Check if file exists, create it if it doesn't
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// Create file with header
		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		
		writer := csv.NewWriter(file)
		defer writer.Flush()
		
		// Write header
		err = writer.Write([]string{"ID", "Email", "DateAdded", "Redeemed"})
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	
	logger.Info("CSV Repository initialized", "path", filePath)
	return &CSVRepository{
		filePath: filePath, 
		logger: logger,
	}, nil
}

func (r *CSVRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	if email == "" {
		return nil, errors.New("email cannot be empty")
	}

	r.logger.Debug("Looking for email in CSV", "email", email)

	// Open file for reading
	file, err := os.Open(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	_, err = reader.Read()
	if err != nil {
		r.logger.Error("Failed to read CSV header", "error", err)
		return nil, err
	}

	// Read rows
	for {
		record, err := reader.Read()
		if err != nil {
			break // End of file or error
		}

		// Check if this is the email we're looking for (case-insensitive)
			if len(record) >= 2 && strings.EqualFold(record[1], email) {
			user := &domain.User{
				ID:    record[0],
				Email: record[1],
			}

			// Parse DateAdded
			if len(record) >= 3 && record[2] != "" {
				dateAdded, err := time.Parse(time.RFC3339, record[2])
				if err == nil {
					user.DateAdded = dateAdded
				}
			}

			// Parse Redeemed
			if len(record) >= 4 && record[3] != "" {
				consumed, err := time.Parse(time.RFC3339, record[3])
				if err == nil {
					user.Redeemed = &consumed
				}
			}

			r.logger.Debug("Found user in CSV", "email", email, "redeemed", user.IsRedeemed())
			return user, nil
		}
	}

	r.logger.Debug("User not found in CSV", "email", email)
	return nil, domain.ErrUserNotFound
}

func (r *CSVRepository) UpdateUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Updating user in CSV", "email", user.Email)

	// Read all records
	file, err := os.Open(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file for update", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		file.Close()
		r.logger.Error("Failed to read CSV records", "error", err)
		return err
	}
	file.Close()

	// Update the record
	found := false
	for i, record := range records {
		if i == 0 { // Skip header
			continue
		}

		if len(record) >= 2 && strings.EqualFold(record[1], user.Email) {
			// Update record
			record[0] = user.ID
			record[2] = user.DateAdded.Format(time.RFC3339)

			if user.Redeemed != nil {
				record[3] = user.Redeemed.Format(time.RFC3339)
			} else {
				record[3] = ""
			}

			records[i] = record
			found = true
			break
		}
	}

	if !found {
		// User doesn't exist
		r.logger.Debug("User not found for update", "email", user.Email)
		return domain.ErrUserNotFound
	}

	// Write all records back
	outFile, err := os.Create(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file for writing", "error", err)
		return err
	}
	defer outFile.Close()
	
	writer := csv.NewWriter(outFile)
	defer writer.Flush()
	
	err = writer.WriteAll(records)
	if err != nil {
		r.logger.Error("Failed to write CSV records", "error", err)
		return err
	}
	
	r.logger.Debug("User updated in CSV", "email", user.Email)
	return nil
}

func (r *CSVRepository) AddUser(ctx any, user *domain.User) error {
	if user == nil {
		return errors.New("user cannot be nil")
	}

	r.logger.Debug("Adding user to CSV", "email", user.Email)

	// Read all records
	file, err := os.Open(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file for addition", "error", err)
		return domain.ErrDatabaseUnavailable
	}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		file.Close()
		r.logger.Error("Failed to read CSV records", "error", err)
		return err
	}
	file.Close()

	// Check if user already exists
	for i, record := range records {
		if i == 0 { // Skip header
			continue
		}

		if len(record) >= 2 && strings.EqualFold(record[1], user.Email) {
			// User already exists, should use UpdateUser instead
			r.logger.Debug("User already exists", "email", user.Email)
			return errors.New("user already exists")
		}
	}

	// Add new record
	newRecord := []string{
		user.ID,
		user.Email,
		user.DateAdded.Format(time.RFC3339),
		"",
	}

	if user.Redeemed != nil {
		newRecord[3] = user.Redeemed.Format(time.RFC3339)
	}

	records = append(records, newRecord)

	// Write all records back
	outFile, err := os.Create(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file for writing", "error", err)
		return err
	}
	defer outFile.Close()
	
	writer := csv.NewWriter(outFile)
	defer writer.Flush()
	
	err = writer.WriteAll(records)
	if err != nil {
		r.logger.Error("Failed to write CSV records", "error", err)
		return err
	}
	
	r.logger.Debug("User added to CSV", "email", user.Email)
	return nil
}

// GetReport retrieves users based on the report parameters
func (r *CSVRepository) GetReport(ctx any, params domain.ReportParams) ([]*domain.User, error) {
	r.logger.Debug("Generating report from CSV", "type", params.Type, "from", params.From, "to", params.To)

	// Open file for reading
	file, err := os.Open(r.filePath)
	if err != nil {
		r.logger.Error("Failed to open CSV file for report", "error", err)
		return nil, domain.ErrDatabaseUnavailable
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read header
	_, err = reader.Read()
	if err != nil {
		r.logger.Error("Failed to read CSV header", "error", err)
		return nil, err
	}

	var users []*domain.User

	// Read all rows
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			r.logger.Error("Error reading CSV record", "error", err)
			return nil, err
		}

		// Parse user data
		if len(record) < 4 {
			continue // Skip invalid records
		}

		// Parse DateAdded
		var dateAdded time.Time
		if record[2] != "" {
			parsedDate, err := time.Parse(time.RFC3339, record[2])
			if err != nil {
				r.logger.Debug("Invalid date format for DateAdded", "value", record[2], "error", err)
				continue // Skip invalid date
			}
			dateAdded = parsedDate
		} else {
			continue // Skip if date added is empty
		}

		// Parse Redeemed
		var redeemed *time.Time
		if record[3] != "" {
			parsedRedeemed, err := time.Parse(time.RFC3339, record[3])
			if err == nil {
				redeemed = &parsedRedeemed
			}
		}

		// Apply date filters
		if !dateAdded.Before(params.From) && !dateAdded.After(params.To) {
			// Apply report type filter
			switch params.Type {
			case domain.ReportTypeRedeemed:
				// Include only redeemed records
				if redeemed != nil {
					users = append(users, &domain.User{
						ID:        record[0],
						Email:     record[1],
						DateAdded: dateAdded,
						Redeemed:  redeemed,
					})
				}
			case domain.ReportTypeAdded:
				// Include all records within the date range
				users = append(users, &domain.User{
					ID:        record[0],
					Email:     record[1],
					DateAdded: dateAdded,
					Redeemed:  redeemed,
				})
			case domain.ReportTypeAll:
				// Include all records
				users = append(users, &domain.User{
					ID:        record[0],
					Email:     record[1],
					DateAdded: dateAdded,
					Redeemed:  redeemed,
				})
			}
		}
	}

	r.logger.Info("Report generated from CSV", "type", params.Type, "count", len(users))
	return users, nil
}

func (r *CSVRepository) Close() error {
	r.logger.Debug("Closing CSV repository")
	// No resources to close for CSV
	return nil
}

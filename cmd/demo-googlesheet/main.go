package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/repository"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

func main() {
	// Parse command-line flags
	credentials := flag.String("creds", "credentials.json", "Path to Google API credentials JSON file")
	spreadsheetID := flag.String("sheet-id", "", "Google Spreadsheet ID")
	sheetName := flag.String("sheet-name", "Sheet1", "Google Sheet name (tab name)")
	action := flag.String("action", "show", "Action to perform: setup, add, show, check, redeem")
	email := flag.String("email", "", "Email address (for add, check, or redeem)")
	flag.Parse()

	// Check required parameters
	if *credentials == "" || *spreadsheetID == "" {
		fmt.Println("Error: credentials and spreadsheet ID are required")
		fmt.Println("Usage example: go run cmd/demo-googlesheet/main.go -creds credentials.json -sheet-id 1abc123xyz -action show")
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New("info")
	log.Info("Google Sheets Demo", "action", *action, "sheet-id", *spreadsheetID)

	// Build connection string
	connectionString := fmt.Sprintf("%s|%s|%s", *credentials, *spreadsheetID, *sheetName)

	// Create repository (except for setup action which creates the sheet)
	var repo domain.Repository
	var err error
	if *action != "setup" {
		repo, err = repository.NewGoogleSheetRepository(context.Background(), connectionString, log)
		if err != nil {
			log.Fatal("Failed to initialize repository", "error", err)
		}
		defer repo.Close()
	}

	// Perform action
	switch *action {
	case "setup":
		err = setupSheet(*credentials, *spreadsheetID, *sheetName)
		if err != nil {
			log.Fatal("Failed to setup sheet", "error", err)
		}
		log.Info("Sheet setup completed successfully")

	case "add":
		if *email == "" {
			log.Fatal("Email is required for add action")
		}
		err = addUser(repo, *email)
		if err != nil {
			log.Fatal("Failed to add user", "error", err)
		}
		log.Info("Added user", "email", *email)

	case "show":
		err = showAllUsers(*credentials, *spreadsheetID, *sheetName)
		if err != nil {
			log.Fatal("Failed to show users", "error", err)
		}

	case "check":
		if *email == "" {
			log.Fatal("Email is required for check action")
		}
		err = checkUser(repo, *email)
		if err != nil {
			log.Fatal("Failed to check user", "error", err)
		}

	case "redeem":
		if *email == "" {
			log.Fatal("Email is required for redeem action")
		}
		err = redeemUser(repo, *email)
		if err != nil {
			log.Fatal("Failed to redeem user", "error", err)
		}

	default:
		log.Fatal("Invalid action", "action", *action)
	}
}

// setupSheet initializes a Google Sheet with the correct headers
func setupSheet(credentials string, spreadsheetID string, sheetName string) error {
	ctx := context.Background()

	// Initialize Google Sheets API
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(credentials))
	if err != nil {
		return fmt.Errorf("failed to create Google Sheets service: %w", err)
	}

	// First, check if the sheet exists and create it if not
	sheet, err := service.Spreadsheets.Get(spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to access spreadsheet: %w", err)
	}

	// Check if the sheet (tab) already exists
	sheetExists := false
	for _, s := range sheet.Sheets {
		if s.Properties.Title == sheetName {
			sheetExists = true
			break
		}
	}

	// If the sheet doesn't exist, create it
	if !sheetExists {
		addSheetRequest := &sheets.AddSheetRequest{
			Properties: &sheets.SheetProperties{
				Title: sheetName,
			},
		}

		batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{
				{
					AddSheet: addSheetRequest,
				},
			},
		}

		_, err = service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to create sheet: %w", err)
		}
		fmt.Printf("Created new sheet: %s\n", sheetName)
	}

	// Set up headers
	headers := []interface{}{"ID", "Email", "Date Added", "Redeemed"}
	valueRange := &sheets.ValueRange{
		Values: [][]interface{}{headers},
	}

	// Check if the sheet already has headers
	existingData, err := service.Spreadsheets.Values.Get(spreadsheetID, fmt.Sprintf("%s!A1:D1", sheetName)).Do()
	if err != nil {
		return fmt.Errorf("failed to read sheet headers: %w", err)
	}

	if len(existingData.Values) > 0 && len(existingData.Values[0]) >= 4 {
		fmt.Println("Headers already exist, skipping header setup")
	} else {
		// Write headers to sheet
		_, err = service.Spreadsheets.Values.Update(
			spreadsheetID,
			fmt.Sprintf("%s!A1:D1", sheetName),
			valueRange,
		).ValueInputOption("RAW").Context(ctx).Do()
		if err != nil {
			return fmt.Errorf("failed to update headers: %w", err)
		}
		fmt.Println("Headers added to sheet")
	}

	// Format header row (bold, freeze)
	requests := []*sheets.Request{
		{
			RepeatCell: &sheets.RepeatCellRequest{
				Range: &sheets.GridRange{
					SheetId:       getSheetID(sheet, sheetName),
					StartRowIndex: 0,
					EndRowIndex:   1,
				},
				Cell: &sheets.CellData{
					UserEnteredFormat: &sheets.CellFormat{
						TextFormat: &sheets.TextFormat{
							Bold: true,
						},
						BackgroundColor: &sheets.Color{
							Red:   0.9,
							Green: 0.9,
							Blue:  0.9,
						},
					},
				},
				Fields: "userEnteredFormat(textFormat,backgroundColor)",
			},
		},
		{
			UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
				Properties: &sheets.SheetProperties{
					SheetId:            getSheetID(sheet, sheetName),
					GridProperties:     &sheets.GridProperties{FrozenRowCount: 1},
				},
				Fields: "gridProperties.frozenRowCount",
			},
		},
	}

	batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
		Requests: requests,
	}

	_, err = service.Spreadsheets.BatchUpdate(spreadsheetID, batchUpdateRequest).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to format headers: %w", err)
	}

	fmt.Println("Sheet setup completed successfully")
	return nil
}

// getSheetID returns the internal sheet ID for a given sheet name
func getSheetID(spreadsheet *sheets.Spreadsheet, sheetName string) int64 {
	for _, sheet := range spreadsheet.Sheets {
		if sheet.Properties.Title == sheetName {
			return sheet.Properties.SheetId
		}
	}
	return 0
}

// addUser adds a new user to the sheet
func addUser(repo domain.Repository, email string) error {
	// Generate a simple ID
	now := time.Now()
	id := fmt.Sprintf("user_%d", now.UnixNano())

	// Create user
	user := &domain.User{
		ID:              id,
		Email:           email,
		DateAdded:       now,
		Redeemed: nil,
	}

	// Add to repository
	err := repo.UpdateUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("failed to add user: %w", err)
	}

	fmt.Printf("Added user: %s (ID: %s)\n", email, id)
	return nil
}

// showAllUsers displays all users in the sheet
func showAllUsers(credentials string, spreadsheetID string, sheetName string) error {
	ctx := context.Background()

	// Initialize Google Sheets API directly (without repository)
	service, err := sheets.NewService(ctx, option.WithCredentialsFile(credentials))
	if err != nil {
		return fmt.Errorf("failed to create Google Sheets service: %w", err)
	}

	// Read data from sheet
	readRange := fmt.Sprintf("%s!A:D", sheetName)
	resp, err := service.Spreadsheets.Values.Get(spreadsheetID, readRange).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("failed to read sheet: %w", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("Sheet is empty")
		return nil
	}

	// Print header
	fmt.Println("\nUsers in Google Sheet:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-15s %-30s %-20s %s\n", "ID", "EMAIL", "ADDED", "REDEEMED")
	fmt.Println("------------------------------------------------------------")

	// Print rows (skip header)
	count := 0
	for i, row := range resp.Values {
		if i == 0 { // Skip header
			continue
		}

		// Extract values
		id := getStringValue(row, 0, "")
		email := getStringValue(row, 1, "")
		dateAdded := getStringValue(row, 2, "")
		consumed := getStringValue(row, 3, "Not Redeemed")

		if consumed == "" {
			consumed = "Not Redeemed"
		}

		fmt.Printf("%-15s %-30s %-20s %s\n", id, email, dateAdded, consumed)
		count++
	}

	if count == 0 {
		fmt.Println("No users found.")
	} else {
		fmt.Printf("\nTotal: %d users\n", count)
	}

	return nil
}

// getStringValue safely extracts a string value from a row
func getStringValue(row []interface{}, index int, defaultValue string) string {
	if index >= len(row) {
		return defaultValue
	}
	
	if str, ok := row[index].(string); ok {
		return str
	}
	
	return defaultValue
}

// checkUser checks if a user exists and is eligible
func checkUser(repo domain.Repository, email string) error {
	user, err := repo.FindByEmail(context.Background(), email)
	if err == domain.ErrUserNotFound {
		fmt.Printf("User %s not found in database\n", email)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking user: %w", err)
	}
	
	fmt.Printf("User found: %s\n", email)
	fmt.Printf("  ID:        %s\n", user.ID)
	fmt.Printf("  Added:     %s\n", user.DateAdded.Format("2006-01-02 15:04:05"))
	
	if user.Redeemed != nil {
		fmt.Printf("  Redeemed:  %s\n", user.Redeemed.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Status:    Already redeemed\n")
	} else {
		fmt.Printf("  Redeemed:  Not yet\n")
		fmt.Printf("  Status:    Eligible for cocktail\n")
	}
	
	return nil
}

// redeemUser marks a user as having redeemed their cocktail
func redeemUser(repo domain.Repository, email string) error {
	user, err := repo.FindByEmail(context.Background(), email)
	if err == domain.ErrUserNotFound {
		fmt.Printf("User %s not found in database\n", email)
		return nil
	} else if err != nil {
		return fmt.Errorf("error finding user: %w", err)
	}
	
	if user.Redeemed != nil {
		fmt.Printf("User %s has already redeemed their cocktail on %s\n", 
			email, user.Redeemed.Format("2006-01-02 15:04:05"))
		return nil
	}
	
	// Mark as redeemed
	user.Redeem()
	
	// Update in database
	err = repo.UpdateUser(context.Background(), user)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	
	fmt.Printf("User %s has successfully redeemed their cocktail on %s\n", 
		email, user.Redeemed.Format("2006-01-02 15:04:05"))
	
	return nil
}
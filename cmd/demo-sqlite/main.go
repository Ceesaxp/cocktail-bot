package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/repository"
)

func main() {
	// Parse command-line flags
	dbPath := flag.String("db", "users.db", "Path to SQLite database file")
	action := flag.String("action", "show", "Action to perform: create, add, show, check, redeem")
	email := flag.String("email", "", "Email address (for add, check, or redeem)")
	flag.Parse()

	// Initialize logger
	log := logger.New("info")
	log.Info("SQLite Demo", "action", *action, "db", *dbPath)

	// Create repository
	repo, err := repository.NewSQLiteRepository(*dbPath, log)
	if err != nil {
		log.Fatal("Failed to initialize repository", "error", err)
	}
	defer repo.Close()

	// Perform action
	switch *action {
	case "create":
		err = createDemoTable(*dbPath)
		if err != nil {
			log.Fatal("Failed to create demo data", "error", err)
		}
		log.Info("Created demo table and sample data")

	case "add":
		if *email == "" {
			log.Fatal("Email is required for add action")
		}
		err = addUser(*dbPath, *email)
		if err != nil {
			log.Fatal("Failed to add user", "error", err)
		}
		log.Info("Added user", "email", *email)

	case "show":
		err = showAllUsers(*dbPath)
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

// createDemoTable initializes the database with sample data
func createDemoTable(dbPath string) error {
	// Remove existing database if it exists
	os.Remove(dbPath)

	// Open database for initialization
	db, err := repository.OpenSQLiteForTesting(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Insert sample users
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	
	// Insert test data
	stmt, err := db.Prepare("INSERT INTO users (id, email, date_added, already_consumed) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	// Sample users
	users := []struct {
		id        string
		email     string
		dateAdded time.Time
		consumed  *time.Time
	}{
		{"1", "alice@example.com", now, nil},                 // Eligible
		{"2", "bob@example.com", now, &yesterday},            // Already redeemed
		{"3", "charlie@example.com", now.Add(-7*24*time.Hour), nil}, // Eligible, added a week ago
	}
	
	for _, user := range users {
		var consumed interface{}
		if user.consumed != nil {
			consumed = *user.consumed
		} else {
			consumed = nil
		}
		
		_, err = stmt.Exec(user.id, user.email, user.dateAdded, consumed)
		if err != nil {
			return fmt.Errorf("failed to insert user %s: %w", user.email, err)
		}
	}
	
	fmt.Printf("Created database with %d sample users\n", len(users))
	return nil
}

// addUser adds a new user to the database
func addUser(dbPath, email string) error {
	db, err := repository.OpenSQLiteForTesting(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	// Generate a simple ID (in production, use a proper ID generation mechanism)
	now := time.Now()
	id := fmt.Sprintf("%d", now.UnixNano())
	
	stmt, err := db.Prepare("INSERT INTO users (id, email, date_added, already_consumed) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(id, email, now, nil)
	if err != nil {
		return fmt.Errorf("failed to insert user: %w", err)
	}
	
	fmt.Printf("Added user: %s\n", email)
	return nil
}

// showAllUsers displays all users in the database
func showAllUsers(dbPath string) error {
	db, err := repository.OpenSQLiteForTesting(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()
	
	rows, err := db.Query("SELECT id, email, date_added, already_consumed FROM users ORDER BY date_added DESC")
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()
	
	fmt.Println("\nUsers in database:")
	fmt.Println("------------------------------------------------------------")
	fmt.Printf("%-10s %-30s %-20s %s\n", "ID", "EMAIL", "ADDED", "REDEEMED")
	fmt.Println("------------------------------------------------------------")
	
	count := 0
	for rows.Next() {
		var (
			id         string
			email      string
			dateAdded  time.Time
		)
		
		// SQLite returns nullable time
		var nullableConsumed any
		err = rows.Scan(&id, &email, &dateAdded, &nullableConsumed)
		if err != nil {
			return fmt.Errorf("failed to scan row: %w", err)
		}
		
		// Format consumed time or show "Not Redeemed"
		consumedStr := "Not Redeemed"
		if nullableConsumed != nil {
			if t, ok := nullableConsumed.(time.Time); ok {
				consumedStr = t.Format("2006-01-02 15:04:05")
			}
		}
		
		fmt.Printf("%-10s %-30s %-20s %s\n", 
			id, 
			email, 
			dateAdded.Format("2006-01-02 15:04:05"),
			consumedStr,
		)
		count++
	}
	
	if count == 0 {
		fmt.Println("No users found.")
	} else {
		fmt.Printf("\nTotal: %d users\n", count)
	}
	
	return nil
}

// checkUser checks if a user exists and is eligible
func checkUser(repo domain.Repository, email string) error {
	user, err := repo.FindByEmail(nil, email)
	if err == domain.ErrUserNotFound {
		fmt.Printf("User %s not found in database\n", email)
		return nil
	} else if err != nil {
		return fmt.Errorf("error checking user: %w", err)
	}
	
	fmt.Printf("User found: %s\n", email)
	fmt.Printf("  ID:        %s\n", user.ID)
	fmt.Printf("  Added:     %s\n", user.DateAdded.Format("2006-01-02 15:04:05"))
	
	if user.AlreadyConsumed != nil {
		fmt.Printf("  Redeemed:  %s\n", user.AlreadyConsumed.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Status:    Already redeemed\n")
	} else {
		fmt.Printf("  Redeemed:  Not yet\n")
		fmt.Printf("  Status:    Eligible for cocktail\n")
	}
	
	return nil
}

// redeemUser marks a user as having redeemed their cocktail
func redeemUser(repo domain.Repository, email string) error {
	user, err := repo.FindByEmail(nil, email)
	if err == domain.ErrUserNotFound {
		fmt.Printf("User %s not found in database\n", email)
		return nil
	} else if err != nil {
		return fmt.Errorf("error finding user: %w", err)
	}
	
	if user.AlreadyConsumed != nil {
		fmt.Printf("User %s has already redeemed their cocktail on %s\n", 
			email, user.AlreadyConsumed.Format("2006-01-02 15:04:05"))
		return nil
	}
	
	// Mark as redeemed
	user.Redeem()
	
	// Update in database
	err = repo.UpdateUser(nil, user)
	if err != nil {
		return fmt.Errorf("error updating user: %w", err)
	}
	
	fmt.Printf("User %s has successfully redeemed their cocktail on %s\n", 
		email, user.AlreadyConsumed.Format("2006-01-02 15:04:05"))
	
	return nil
}
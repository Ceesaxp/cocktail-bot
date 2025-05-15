package repository_test

import (
	"os"
	"testing"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/repository"
)

func TestSQLiteRepository(t *testing.T) {
	// Create a temporary database file
	dbPath := "test_sqlite.db"
	
	// Cleanup after test
	defer os.Remove(dbPath)
	
	// Create logger
	l := logger.New("info")
	
	// Create repository
	repo, err := repository.NewSQLiteRepository(dbPath, l)
	if err != nil {
		t.Fatalf("Failed to create SQLite repository: %v", err)
	}
	defer repo.Close()
	
	// Need to initialize the database with test data
	err = initTestData(repo, dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize test data: %v", err)
	}
	
	// Test finding a user by email
	t.Run("FindByEmail - Existing User", func(t *testing.T) {
		user, err := repo.FindByEmail(nil, "test1@example.com")
		if err != nil {
			t.Errorf("Error finding user: %v", err)
		}
		
		if user == nil {
			t.Fatal("User should not be nil")
		}
		
		if user.Email != "test1@example.com" {
			t.Errorf("Expected email test1@example.com, got %s", user.Email)
		}
		
		if user.Redeemed != nil {
			t.Errorf("User should not have consumed cocktail")
		}
	})
	
	t.Run("FindByEmail - Non-existent User", func(t *testing.T) {
		_, err := repo.FindByEmail(nil, "nonexistent@example.com")
		if err != domain.ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got: %v", err)
		}
	})
	
	t.Run("FindByEmail - Case Insensitive", func(t *testing.T) {
		user, err := repo.FindByEmail(nil, "TEST1@EXAMPLE.COM")
		if err != nil {
			t.Errorf("Error finding user: %v", err)
		}
		
		if user == nil {
			t.Fatal("User should not be nil")
		}
		
		if user.Email != "test1@example.com" {
			t.Errorf("Expected email test1@example.com, got %s", user.Email)
		}
	})
	
	t.Run("UpdateUser - Mark as Redeemed", func(t *testing.T) {
		// First get the user
		user, err := repo.FindByEmail(nil, "test1@example.com")
		if err != nil {
			t.Fatalf("Error finding user: %v", err)
		}
		
		// Mark as redeemed
		user.Redeem()
		
		// Update user
		err = repo.UpdateUser(nil, user)
		if err != nil {
			t.Errorf("Error updating user: %v", err)
		}
		
		// Verify update
		updatedUser, err := repo.FindByEmail(nil, "test1@example.com")
		if err != nil {
			t.Errorf("Error finding updated user: %v", err)
		}
		
		if updatedUser.Redeemed == nil {
			t.Errorf("User should have consumed cocktail")
		}
	})
	
	t.Run("UpdateUser - Non-existent User", func(t *testing.T) {
		nonExistentUser := &domain.User{
			ID:              "999",
			Email:           "nonexistent@example.com",
			DateAdded:       time.Now(),
			Redeemed: nil,
		}
		
		err := repo.UpdateUser(nil, nonExistentUser)
		if err != domain.ErrUserNotFound {
			t.Errorf("Expected ErrUserNotFound, got: %v", err)
		}
	})

	t.Run("AddUser - New User", func(t *testing.T) {
		newUser := &domain.User{
			ID:        "3",
			Email:     "newuser@example.com",
			DateAdded: time.Now(),
			Redeemed:  nil,
		}
		
		// Add the user
		err := repo.AddUser(nil, newUser)
		if err != nil {
			t.Errorf("Failed to add new user: %v", err)
		}
		
		// Verify user was added
		addedUser, err := repo.FindByEmail(nil, "newuser@example.com")
		if err != nil {
			t.Errorf("Failed to find added user: %v", err)
		}
		
		if addedUser == nil || addedUser.ID != "3" {
			t.Errorf("Added user data incorrect: %+v", addedUser)
		}
	})
	
	t.Run("AddUser - Duplicate Email", func(t *testing.T) {
		duplicateUser := &domain.User{
			ID:        "4",
			Email:     "newuser@example.com", // Same email as existing user
			DateAdded: time.Now(),
			Redeemed:  nil,
		}
		
		// Should get an error when trying to add a user with a duplicate email
		err := repo.AddUser(nil, duplicateUser)
		if err == nil {
			t.Errorf("Expected error when adding user with duplicate email")
		}
	})
}

// initTestData initializes the test database with sample data
func initTestData(repo domain.Repository, dbPath string) error {
	// We need a direct connection to insert test data since the Repository interface
	// doesn't have methods for creating users
	db, err := repository.OpenSQLiteForTesting(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	
	// Insert test users
	now := time.Now()
	stmt, err := db.Prepare("INSERT INTO users (id, email, date_added, redeemed) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	
	// User 1: Not redeemed
	_, err = stmt.Exec("1", "test1@example.com", now, nil)
	if err != nil {
		return err
	}
	
	// User 2: Already redeemed
	consumed := time.Now().Add(-24 * time.Hour)
	_, err = stmt.Exec("2", "test2@example.com", now, consumed)
	if err != nil {
		return err
	}
	
	return nil
}
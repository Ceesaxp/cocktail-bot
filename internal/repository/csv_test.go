package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Ceesaxp/cocktail-bot/internal/domain"
	"github.com/Ceesaxp/cocktail-bot/internal/logger"
	"github.com/Ceesaxp/cocktail-bot/internal/repository"
)

func TestCSVRepository(t *testing.T) {
	// Create a temporary CSV file
	tmpfile, err := os.CreateTemp("", "users*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Create test data
	now := time.Now()
	consumedTime := now.Add(-24 * time.Hour)

	// Write initial data
	initialData := "ID,Email,Date Added,Already Consumed\n" +
		"1,user1@example.com," + now.Format(time.RFC3339) + ",\n" +
		"2,user2@example.com," + now.Format(time.RFC3339) + "," + consumedTime.Format(time.RFC3339) + "\n"

	if _, err := tmpfile.Write([]byte(initialData)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create logger
	l := logger.New("info")

	// Create repository
	repo, err := repository.NewCSVRepository(tmpfile.Name(), l)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}
	defer repo.Close()

	// Test FindByEmail for existing user (not redeemed)
	ctx := context.Background()
	user1, err := repo.FindByEmail(ctx, "user1@example.com")
	if err != nil {
		t.Errorf("Failed to find user1: %v", err)
	}

	if user1 == nil {
		t.Fatalf("User1 should not be nil")
	}

	if user1.ID != "1" || user1.Email != "user1@example.com" || user1.AlreadyConsumed != nil {
		t.Errorf("User1 data incorrect: %+v", user1)
	}

	// Test FindByEmail for existing user (already redeemed)
	user2, err := repo.FindByEmail(ctx, "user2@example.com")
	if err != nil {
		t.Errorf("Failed to find user2: %v", err)
	}

	if user2 == nil {
		t.Fatalf("User2 should not be nil")
	}

	if user2.ID != "2" || user2.Email != "user2@example.com" || user2.AlreadyConsumed == nil {
		t.Errorf("User2 data incorrect: %+v", user2)
	}

	// Test FindByEmail for non-existing user
	_, err = repo.FindByEmail(ctx, "nonexistent@example.com")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}

	// Test FindByEmail with case-insensitive email
	user1CaseInsensitive, err := repo.FindByEmail(ctx, "USER1@EXAMPLE.COM")
	if err != nil {
		t.Errorf("Failed to find user1 with case-insensitive email: %v", err)
	}

	if user1CaseInsensitive == nil || user1CaseInsensitive.ID != "1" {
		t.Errorf("Case-insensitive email lookup failed")
	}

	// Test UpdateUser
	user1.Redeem()
	err = repo.UpdateUser(ctx, user1)
	if err != nil {
		t.Errorf("Failed to update user1: %v", err)
	}

	// Verify update worked
	updatedUser1, err := repo.FindByEmail(ctx, "user1@example.com")
	if err != nil {
		t.Errorf("Failed to find updated user1: %v", err)
	}

	if updatedUser1 == nil || updatedUser1.AlreadyConsumed == nil {
		t.Errorf("User1 should have been updated with redemption time")
	}

	// Test UpdateUser for non-existing user
	nonExistentUser := &domain.User{
		ID:        "999",
		Email:     "nonexistent@example.com",
		DateAdded: time.Now(),
	}

	err = repo.UpdateUser(ctx, nonExistentUser)
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound for updating non-existent user, got: %v", err)
	}
}

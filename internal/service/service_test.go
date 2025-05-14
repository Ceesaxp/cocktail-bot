package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/ratelimit"
	"github.com/ceesaxp/cocktail-bot/internal/service"
)

// mockRepository is a mock implementation of domain.Repository
type mockRepository struct {
	users map[string]*domain.User
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users: make(map[string]*domain.User),
	}
}

func (r *mockRepository) FindByEmail(ctx any, email string) (*domain.User, error) {
	user, exists := r.users[email]
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

func (r *mockRepository) UpdateUser(ctx any, user *domain.User) error {
	_, exists := r.users[user.Email]
	if !exists {
		return domain.ErrUserNotFound
	}
	// Update user
	userCopy := *user
	if user.AlreadyConsumed != nil {
		timeCopy := *user.AlreadyConsumed
		userCopy.AlreadyConsumed = &timeCopy
	}
	r.users[user.Email] = &userCopy
	return nil
}

func (r *mockRepository) Close() error {
	return nil
}

// mockRepositoryFactory is a factory that creates a mock repository
type mockRepositoryFactory struct {
	repo *mockRepository
}

func (f *mockRepositoryFactory) NewRepository(ctx any, cfg config.DatabaseConfig, logger *logger.Logger) (domain.Repository, error) {
	return f.repo, nil
}

func TestCheckEmailStatus(t *testing.T) {
	// Create mock repository
	mockRepo := newMockRepository()

	// Add test users
	now := time.Now()
	redeemTime := now.Add(-24 * time.Hour)

	mockRepo.users["user1@example.com"] = &domain.User{
		ID:              "1",
		Email:           "user1@example.com",
		DateAdded:       now,
		AlreadyConsumed: nil,
	}

	mockRepo.users["user2@example.com"] = &domain.User{
		ID:              "2",
		Email:           "user2@example.com",
		DateAdded:       now,
		AlreadyConsumed: &redeemTime,
	}

	// Create logger
	l := logger.New("info")

	// Create a test service
	svc := service.NewForTest(mockRepo, ratelimit.New(10, 100), l)

	// Test cases
	testCases := []struct {
		name     string
		email    string
		expected string
	}{
		{"Eligible user", "user1@example.com", "eligible"},
		{"Already redeemed", "user2@example.com", "redeemed"},
		{"Non-existent user", "nonexistent@example.com", "not_found"},
		{"Case-insensitive", "USER1@EXAMPLE.COM", "eligible"},
	}

	// Using any as context
	ctx := context.Background()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status, user, err := svc.CheckEmailStatus(ctx, 12345, tc.email)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if status != tc.expected {
				t.Errorf("Expected status %s, got %s", tc.expected, status)
			}

			switch tc.expected {
			case "eligible":
				if user == nil || user.ID != "1" || user.IsRedeemed() {
					t.Errorf("Expected eligible user, got %+v", user)
				}
			case "redeemed":
				if user == nil || user.ID != "2" || !user.IsRedeemed() {
					t.Errorf("Expected redeemed user, got %+v", user)
				}
			case "not_found":
				if user != nil {
					t.Errorf("Expected nil user for not_found, got %+v", user)
				}
			}
		})
	}
}

func TestRedeemCocktail(t *testing.T) {
	// Create mock repository
	mockRepo := newMockRepository()

	// Add test users
	now := time.Now()
	redeemTime := now.Add(-24 * time.Hour)

	mockRepo.users["eligible@example.com"] = &domain.User{
		ID:              "1",
		Email:           "eligible@example.com",
		DateAdded:       now,
		AlreadyConsumed: nil,
	}

	mockRepo.users["redeemed@example.com"] = &domain.User{
		ID:              "2",
		Email:           "redeemed@example.com",
		DateAdded:       now,
		AlreadyConsumed: &redeemTime,
	}

	// Create logger
	l := logger.New("info")

	// Create a test service
	svc := service.NewForTest(mockRepo, ratelimit.New(10, 100), l)

	// Using any as context
	ctx := context.Background()

	// Test redeeming eligible user
	redemptionTime, err := svc.RedeemCocktail(ctx, 12345, "eligible@example.com")
	if err != nil {
		t.Errorf("Failed to redeem eligible user: %v", err)
	}

	if redemptionTime.IsZero() {
		t.Errorf("Expected non-zero redemption time")
	}

	// Verify the user was updated in the repository
	updatedUser, err := mockRepo.FindByEmail(ctx, "eligible@example.com")
	if err != nil {
		t.Errorf("Failed to find updated user: %v", err)
	}

	if updatedUser == nil || updatedUser.AlreadyConsumed == nil {
		t.Errorf("User should have been updated with redemption time")
	}

	// Test redeeming already redeemed user
	oldRedeemTime, err := svc.RedeemCocktail(ctx, 12345, "redeemed@example.com")
	if err != nil {
		t.Errorf("Failed to redeem already redeemed user: %v", err)
	}

	// Fix the comparison - we don't want to compare with redemption time from the first test
	// but rather with the original redeemTime
	if oldRedeemTime.Format(time.RFC3339) != redeemTime.Format(time.RFC3339) {
		t.Errorf("Expected original redemption time %s, got %s", 
			redeemTime.Format(time.RFC3339), oldRedeemTime.Format(time.RFC3339))
	}

	// Test redeeming non-existent user
	_, err = svc.RedeemCocktail(ctx, 12345, "nonexistent@example.com")
	if err != domain.ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got: %v", err)
	}
}

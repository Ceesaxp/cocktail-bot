package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

// mockService implements ServiceInterface for testing
type mockService struct {
	findEmailStatus   string
	findEmailUser     *domain.User
	findEmailError    error
	redeemError       error
	updateUserError   error
	updateUserCalled  bool
	updateUserPayload *domain.User
	addUserError      error
	addUserCalled     bool
	addUserPayload    *domain.User
}

func (s *mockService) CheckEmailStatus(ctx any, userID int64, email string) (string, *domain.User, error) {
	return s.findEmailStatus, s.findEmailUser, s.findEmailError
}

func (s *mockService) RedeemCocktail(ctx any, userID int64, email string) (time.Time, error) {
	if s.redeemError != nil {
		return time.Time{}, s.redeemError
	}
	return time.Now(), nil
}

func (s *mockService) UpdateUser(ctx any, user *domain.User) error {
	s.updateUserCalled = true
	s.updateUserPayload = user
	return s.updateUserError
}

func (s *mockService) AddUser(ctx any, user *domain.User) error {
	s.addUserCalled = true
	s.addUserPayload = user
	return s.addUserError
}

func (s *mockService) Close() error {
	return nil
}

// createTestServer creates a server for testing
func createTestServer(t *testing.T, svc ServiceInterface) (*Server, *httptest.Server) {
	// Create test configuration
	cfg := &config.Config{
		API: config.APIConfig{
			Enabled:          true,
			Port:             8080,
			AuthTokens:       []string{"test_token"},
			RateLimitPerMin:  60,
			RateLimitPerHour: 600,
		},
	}

	// Create test logger
	log := logger.New("debug")

	// Create server
	server, err := New(cfg, svc, log)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Create test HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/email", server.handleEmail)
	mux.HandleFunc("/api/health", server.handleHealth)

	ts := httptest.NewServer(mux)
	return server, ts
}

func TestHealthEndpoint(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if respBody["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %v", respBody["status"])
	}
}

func TestEmailEndpoint_MissingAuth(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Create request without Authorization header
	payload := map[string]string{"email": "test@example.com"}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/email", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestEmailEndpoint_InvalidAuth(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Create request with invalid token
	payload := map[string]string{"email": "test@example.com"}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/email", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid_token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", resp.StatusCode)
	}
}

func TestEmailEndpoint_InvalidEmail(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Create request with invalid email
	payload := map[string]string{"email": "not-an-email"}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/email", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", resp.StatusCode)
	}

	var errorResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if !strings.Contains(strings.ToLower(errorResp.Error), "invalid email") {
		t.Errorf("Error message should contain 'invalid email', got: %s", errorResp.Error)
	}
}

func TestEmailEndpoint_EmailExists(t *testing.T) {
	// Create mock service that returns "eligible" status (email exists)
	existingUser := &domain.User{
		ID:        "existing_123",
		Email:     "test@example.com",
		DateAdded: time.Now(),
	}

	svc := &mockService{
		findEmailStatus: "eligible",
		findEmailUser:   existingUser,
	}

	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Create request with valid email that exists
	payload := map[string]string{"email": "test@example.com"}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/email", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409, got %d", resp.StatusCode)
	}

	var emailResp EmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if emailResp.Status != "exists" {
		t.Errorf("Expected status 'exists', got %s", emailResp.Status)
	}

	if emailResp.ID != existingUser.ID {
		t.Errorf("Expected ID %s, got %s", existingUser.ID, emailResp.ID)
	}
}

func TestEmailEndpoint_ValidNew(t *testing.T) {
	svc := &mockService{
		findEmailStatus: "not_found",
	}

	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Create request with valid new email
	payload := map[string]string{"email": "new@example.com"}
	jsonData, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", ts.URL+"/api/v1/email", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test_token")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var emailResp EmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if emailResp.Status != "created" {
		t.Errorf("Expected status 'created', got %s", emailResp.Status)
	}

	if emailResp.ID == "" {
		t.Errorf("Expected non-empty ID")
	}

	// Verify AddUser was called (instead of UpdateUser, since we've updated the implementation)
	if !svc.addUserCalled {
		t.Errorf("Expected AddUser to be called")
	}

	// Verify email was normalized
	if svc.addUserPayload != nil && svc.addUserPayload.Email != "new@example.com" {
		t.Errorf("Expected email to be normalized to 'new@example.com', got %s", svc.addUserPayload.Email)
	}
}

func TestServer_Start_Stop(t *testing.T) {
	// Mock service
	svc := &mockService{}

	// Create test configuration
	cfg := &config.Config{
		API: config.APIConfig{
			Enabled: true,
			Port:    8099, // Use a different port to avoid conflicts
		},
	}

	// Create logger
	log := logger.New("debug")

	// Create server
	server, err := New(cfg, svc, log)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Start server
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop server
	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

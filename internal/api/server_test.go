package api

import (
	"bytes"
	"encoding/json"
	"io"
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
	findEmailStatus      string
	findEmailUser        *domain.User
	findEmailError       error
	redeemError          error
	updateUserError      error
	updateUserCalled     bool
	updateUserPayload    *domain.User
	addUserError         error
	addUserCalled        bool
	addUserPayload       *domain.User
	generateReportUsers  []*domain.User
	generateReportError  error
	generateReportCalled bool
	generateReportType   string
	generateReportFrom   time.Time
	generateReportTo     time.Time
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

func (s *mockService) GenerateReport(ctx any, reportType string, fromDate, toDate time.Time) ([]*domain.User, error) {
	s.generateReportCalled = true
	s.generateReportType = reportType
	s.generateReportFrom = fromDate
	s.generateReportTo = toDate
	return s.generateReportUsers, s.generateReportError
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
	mux.HandleFunc("/api/v1/report/redeemed", server.handleReportRedeemed)
	mux.HandleFunc("/api/v1/report/added", server.handleReportAdded)
	mux.HandleFunc("/api/v1/report/all", server.handleReportAll)
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

func TestReportEndpoints_Unauthorized(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	endpoints := []string{
		"/api/v1/report/redeemed",
		"/api/v1/report/added",
		"/api/v1/report/all",
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint, func(t *testing.T) {
			// Test with no auth header
			req, _ := http.NewRequest("GET", ts.URL+endpoint, nil)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp.StatusCode)
			}

			// Test with invalid token
			req, _ = http.NewRequest("GET", ts.URL+endpoint, nil)
			req.Header.Set("Authorization", "Bearer invalid_token")
			resp, err = client.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", resp.StatusCode)
			}
		})
	}
}

func TestReportEndpoint_Success(t *testing.T) {
	// Create test users
	now := time.Now()
	testUsers := []*domain.User{
		{
			ID:        "1",
			Email:     "user1@example.com",
			DateAdded: now.AddDate(0, 0, -1),
			Redeemed:  nil,
		},
		{
			ID:        "2",
			Email:     "user2@example.com",
			DateAdded: now.AddDate(0, 0, -2),
			Redeemed:  &now,
		},
	}

	endpoints := []struct {
		path       string
		reportType string
	}{
		{"/api/v1/report/redeemed", "redeemed"},
		{"/api/v1/report/added", "added"},
		{"/api/v1/report/all", "all"},
	}

	for _, endpoint := range endpoints {
		t.Run(endpoint.path, func(t *testing.T) {
			// Create mock service that returns test users
			svc := &mockService{
				generateReportUsers: testUsers,
			}

			_, ts := createTestServer(t, svc)
			defer ts.Close()

			// Test with valid token
			req, _ := http.NewRequest("GET", ts.URL+endpoint.path, nil)
			req.Header.Set("Authorization", "Bearer test_token")
			
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Error making request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status 200, got %d", resp.StatusCode)
			}

			// Parse response
			var reportResp ReportResponse
			if err := json.NewDecoder(resp.Body).Decode(&reportResp); err != nil {
				t.Fatalf("Error decoding response: %v", err)
			}

			// Check response fields
			if reportResp.Type != endpoint.reportType {
				t.Errorf("Expected report type %s, got %s", endpoint.reportType, reportResp.Type)
			}

			if reportResp.Count != len(testUsers) {
				t.Errorf("Expected count %d, got %d", len(testUsers), reportResp.Count)
			}

			if len(reportResp.Users) != len(testUsers) {
				t.Errorf("Expected %d users, got %d", len(testUsers), len(reportResp.Users))
			}

			// Verify service was called with correct params
			if !svc.generateReportCalled {
				t.Error("Expected GenerateReport to be called")
			}

			if svc.generateReportType != endpoint.reportType {
				t.Errorf("Expected report type %s, got %s", endpoint.reportType, svc.generateReportType)
			}
		})
	}
}

func TestReportEndpoint_DateParams(t *testing.T) {
	svc := &mockService{
		generateReportUsers: []*domain.User{},
	}

	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Test with custom date parameters
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/report/all?from=2023-01-01&to=2023-12-31", nil)
	req.Header.Set("Authorization", "Bearer test_token")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check that the service was called with the correct date params
	expectedFrom, _ := time.Parse("2006-01-02", "2023-01-01")
	expectedTo, _ := time.Parse("2006-01-02", "2023-12-31")
	// Add almost 24 hours to the "to" date for inclusive range
	expectedTo = expectedTo.Add(24*time.Hour - time.Second)

	sameDay := func(t1, t2 time.Time) bool {
		return t1.Year() == t2.Year() && t1.Month() == t2.Month() && t1.Day() == t2.Day()
	}

	if !sameDay(svc.generateReportFrom, expectedFrom) {
		t.Errorf("Expected from date %v, got %v", expectedFrom, svc.generateReportFrom)
	}

	if !sameDay(svc.generateReportTo, expectedTo) {
		t.Errorf("Expected to date %v, got %v", expectedTo, svc.generateReportTo)
	}
}

func TestReportEndpoint_CSVFormat(t *testing.T) {
	// Create test users
	now := time.Now()
	testUsers := []*domain.User{
		{
			ID:        "1",
			Email:     "user1@example.com",
			DateAdded: now.AddDate(0, 0, -1),
			Redeemed:  nil,
		},
		{
			ID:        "2",
			Email:     "user2@example.com",
			DateAdded: now.AddDate(0, 0, -2),
			Redeemed:  &now,
		},
	}

	svc := &mockService{
		generateReportUsers: testUsers,
	}

	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Test CSV format
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/report/all?format=csv", nil)
	req.Header.Set("Authorization", "Bearer test_token")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("Expected Content-Type 'text/csv', got '%s'", contentType)
	}

	// Check for content disposition header
	contentDisposition := resp.Header.Get("Content-Disposition")
	if !strings.Contains(contentDisposition, "attachment") || !strings.Contains(contentDisposition, "filename=") {
		t.Errorf("Expected Content-Disposition header with attachment and filename, got '%s'", contentDisposition)
	}

	// Read the CSV data
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}

	// Check CSV content
	csvContent := string(body)
	if !strings.Contains(csvContent, "ID,Email,DateAdded,Redeemed") {
		t.Error("CSV header not found in response")
	}

	for _, user := range testUsers {
		if !strings.Contains(csvContent, user.ID) || !strings.Contains(csvContent, user.Email) {
			t.Errorf("User data not found in CSV: %s, %s", user.ID, user.Email)
		}
	}
}

func TestReportEndpoint_InvalidDate(t *testing.T) {
	svc := &mockService{}
	_, ts := createTestServer(t, svc)
	defer ts.Close()

	// Test with invalid date format
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/report/all?from=invalid-date", nil)
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

	// Check error response
	var errorResp ErrorResponse
	if err := json.NewDecoder(resp.Body).Decode(&errorResp); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if !strings.Contains(strings.ToLower(errorResp.Error), "invalid date format") {
		t.Errorf("Expected error message to contain 'invalid date format', got: %s", errorResp.Error)
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

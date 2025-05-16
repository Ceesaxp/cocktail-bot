package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
	"github.com/ceesaxp/cocktail-bot/internal/ratelimit"
	"github.com/ceesaxp/cocktail-bot/internal/utils"
)

// Server represents the REST API server
type Server struct {
	config       *config.Config
	logger       *logger.Logger
	service      ServiceInterface
	httpServer   *http.Server
	limiter      *ratelimit.Limiter
	authProvider *AuthProvider
	running      bool
}

// ServiceInterface defines the required methods from the service layer
type ServiceInterface interface {
	CheckEmailStatus(ctx any, userID int64, email string) (string, *domain.User, error)
	RedeemCocktail(ctx any, userID int64, email string) (time.Time, error)
	UpdateUser(ctx any, user *domain.User) error
	AddUser(ctx any, user *domain.User) error
	GenerateReport(ctx any, reportType string, fromDate, toDate time.Time) ([]*domain.User, error)
	Close() error
}

// EmailRequest represents the JSON payload for email submission
type EmailRequest struct {
	Email string `json:"email"`
}

// EmailResponse represents the JSON response for email submission
type EmailResponse struct {
	ID      string `json:"id,omitempty"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ErrorResponse represents the JSON response for errors
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// ReportResponse represents the JSON response for report requests
type ReportResponse struct {
	Type      string         `json:"type"`
	From      string         `json:"from"`
	To        string         `json:"to"`
	Count     int            `json:"count"`
	Users     []*domain.User `json:"users,omitempty"`
	Generated time.Time      `json:"generated"`
}

// New creates a new API server
func New(cfg *config.Config, svc ServiceInterface, log *logger.Logger) (*Server, error) {
	// Load authentication tokens if configured in tokens file
	if err := cfg.LoadAuthTokens(); err != nil {
		log.Warn("Error loading auth tokens from file", "error", err)
		// Continue with tokens from config.yaml
	}
	
	if len(cfg.API.AuthTokens) == 0 {
		log.Warn("No API tokens configured - API authentication will be unavailable")
	} else {
		log.Info("API tokens configured", "count", len(cfg.API.AuthTokens))
	}

	// Create a dedicated rate limiter for API requests
	limiter := ratelimit.New(cfg.API.RateLimitPerMin, cfg.API.RateLimitPerHour)

	// Create auth provider with tokens from config
	authProvider := NewAuthProvider(cfg.API.AuthTokens)

	mux := http.NewServeMux()

	// Configure bind address
	bindAddr := fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port)
	log.Info("API will bind to", "address", bindAddr)

	server := &Server{
		config:       cfg,
		logger:       log,
		service:      svc,
		limiter:      limiter,
		authProvider: authProvider,
		httpServer: &http.Server{
			Addr:    bindAddr,
			Handler: mux,
		},
	}

	// Register routes
	mux.HandleFunc("/api/v1/email", server.handleEmail)
	mux.HandleFunc("/api/v1/report/redeemed", server.handleReportRedeemed)
	mux.HandleFunc("/api/v1/report/added", server.handleReportAdded)
	mux.HandleFunc("/api/v1/report/all", server.handleReportAll)
	mux.HandleFunc("/api/health", server.handleHealth)

	return server, nil
}

// Start starts the API server
func (s *Server) Start() error {
	if s.running {
		return errors.New("server is already running")
	}

	// Check if API is enabled
	if !s.config.API.Enabled {
		s.logger.Info("API server is disabled in configuration")
		return nil
	}

	s.running = true
	s.logger.Info("Starting API server", "port", s.config.API.Port)

	// Start server in a separate goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("API server error", "error", err)
		}
	}()

	return nil
}

// Stop stops the API server
func (s *Server) Stop() error {
	if !s.running {
		return nil
	}

	s.logger.Info("Stopping API server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	s.running = false
	// Close rate limiter resources
	s.limiter.Close()

	return nil
}

// handleEmail handles the email submission endpoint
func (s *Server) handleEmail(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	// Validate Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		s.writeErrorResponse(w, "Invalid Content-Type", http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	// Authenticate request
	apiKey := r.Header.Get("Authorization")
	if len(apiKey) > 7 && strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = apiKey[7:] // Remove 'Bearer ' prefix
	}

	if !s.authProvider.Authenticate(apiKey) {
		s.writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized, "Invalid or missing authentication token")
		return
	}

	// Apply rate limiting
	clientIP := getClientIP(r)
	clientID := int64(HashCode(clientIP)) // Convert IP to a numeric ID for rate limiter

	if !s.limiter.Allow(clientID) {
		s.writeErrorResponse(w, "Too Many Requests", http.StatusTooManyRequests, "Rate limit exceeded")

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit-Minute", strconv.Itoa(s.config.API.RateLimitPerMin))
		w.Header().Set("X-RateLimit-Remaining-Minute", strconv.Itoa(s.limiter.RemainingMinute(clientID)))
		return
	}

	// Decode request
	var req EmailRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		s.writeErrorResponse(w, "Invalid request", http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	// Validate email
	if !utils.IsValidEmail(req.Email) {
		s.writeErrorResponse(w, "Invalid email", http.StatusBadRequest, "The provided email address is not valid")
		return
	}

	// Normalize email
	email := utils.NormalizeEmail(req.Email)

	// Check if email already exists
	ctx := context.Background()
	status, user, err := s.service.CheckEmailStatus(ctx, clientID, email)
	if err != nil {
		s.logger.Error("Error checking email status", "email", email, "error", err)
		s.writeErrorResponse(w, "Internal server error", http.StatusInternalServerError, "Error processing request")
		return
	}

	// Handle based on status
	switch status {
	case "eligible", "redeemed":
		// Email already exists, return conflict status
		response := EmailResponse{
			Status:  "exists",
			Message: "Email already exists in database",
		}
		if user != nil {
			response.ID = user.ID
		}
		s.writeJSONResponse(w, response, http.StatusConflict)
		return

	case "rate_limited":
		s.writeErrorResponse(w, "Too Many Requests", http.StatusTooManyRequests, "Rate limit exceeded")
		return

	case "unavailable":
		s.writeErrorResponse(w, "Service Unavailable", http.StatusServiceUnavailable, "Database is temporarily unavailable")
		return

	case "not_found":
		// Continue with adding the email
		break
	}

	// Generate a new user with a unique ID
	newUser := &domain.User{
		ID:        GenerateUniqueID(),
		Email:     email,
		DateAdded: time.Now(),
		Redeemed:  nil,
	}

	// Store in database using service's AddUser method for new users
	if err := s.service.AddUser(ctx, newUser); err != nil {
		s.logger.Error("Error adding email to database", "email", email, "error", err)
		s.writeErrorResponse(w, "Internal server error", http.StatusInternalServerError, "Error storing email")
		return
	}

	// Log successful addition
	s.logger.Info("Email added via API", "email", email, "id", newUser.ID)

	// Return success
	response := EmailResponse{
		ID:      newUser.ID,
		Status:  "created",
		Message: "Email added successfully",
	}
	s.writeJSONResponse(w, response, http.StatusCreated)
}

// handleReportRedeemed handles the redeemed report endpoint
func (s *Server) handleReportRedeemed(w http.ResponseWriter, r *http.Request) {
	s.handleReport(w, r, "redeemed")
}

// handleReportAdded handles the added report endpoint
func (s *Server) handleReportAdded(w http.ResponseWriter, r *http.Request) {
	s.handleReport(w, r, "added")
}

// handleReportAll handles the all report endpoint
func (s *Server) handleReportAll(w http.ResponseWriter, r *http.Request) {
	s.handleReport(w, r, "all")
}

// handleReport is a generic handler for all report types
func (s *Server) handleReport(w http.ResponseWriter, r *http.Request, reportType string) {
	// Only allow GET method
	if r.Method != http.MethodGet {
		s.writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Only GET method is allowed")
		return
	}

	// Authenticate request
	apiKey := r.Header.Get("Authorization")
	if len(apiKey) > 7 && strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = apiKey[7:] // Remove 'Bearer ' prefix
	}

	if !s.authProvider.Authenticate(apiKey) {
		s.writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized, "Invalid or missing authentication token")
		return
	}

	// Apply rate limiting
	clientIP := getClientIP(r)
	clientID := int64(HashCode(clientIP))

	if !s.limiter.Allow(clientID) {
		s.writeErrorResponse(w, "Too Many Requests", http.StatusTooManyRequests, "Rate limit exceeded")
		return
	}

	// Parse date range parameters
	fromDate, toDate, err := parseDateParams(r)
	if err != nil {
		s.writeErrorResponse(w, "Invalid date format", http.StatusBadRequest, err.Error())
		return
	}

	// Set content type
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json" // Default format is JSON
	}

	// Generate report
	ctx := context.Background()
	users, err := s.service.GenerateReport(ctx, reportType, fromDate, toDate)
	if err != nil {
		s.logger.Error("Error generating report", "type", reportType, "error", err)
		s.writeErrorResponse(w, "Internal server error", http.StatusInternalServerError, "Error generating report")
		return
	}

	// Format-specific response
	if format == "csv" {
		s.writeCSVReport(w, users, reportType)
	} else {
		// Prepare JSON response
		response := ReportResponse{
			Type:      reportType,
			From:      fromDate.Format(time.RFC3339),
			To:        toDate.Format(time.RFC3339),
			Count:     len(users),
			Users:     users,
			Generated: time.Now(),
		}
		s.writeJSONResponse(w, response, http.StatusOK)
	}
}

// writeCSVReport writes the report as a CSV file
func (s *Server) writeCSVReport(w http.ResponseWriter, users []*domain.User, reportType string) {
	// Set headers for CSV download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-report-%s.csv\"", 
		reportType, time.Now().Format("2006-01-02")))
	
	// Write CSV header
	if _, err := w.Write([]byte("ID,Email,DateAdded,Redeemed\n")); err != nil {
		s.logger.Error("Error writing CSV header", "error", err)
		return
	}
	
	// Write each row
	for _, user := range users {
		redeemedStr := ""
		if user.Redeemed != nil {
			redeemedStr = user.Redeemed.Format(time.RFC3339)
		}
		
		row := fmt.Sprintf("%s,%s,%s,%s\n", 
			user.ID, 
			user.Email, 
			user.DateAdded.Format(time.RFC3339), 
			redeemedStr)
		
		if _, err := w.Write([]byte(row)); err != nil {
			s.logger.Error("Error writing CSV row", "error", err)
			return
		}
	}
}

// parseDateParams parses the from and to query parameters
func parseDateParams(r *http.Request) (time.Time, time.Time, error) {
	// Default dates (last 7 days)
	fromDate := time.Now().AddDate(0, 0, -7)
	toDate := time.Now()
	
	// Parse 'from' parameter if provided
	fromParam := r.URL.Query().Get("from")
	if fromParam != "" {
		parsedFrom, err := time.Parse("2006-01-02", fromParam)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'from' date format. Use YYYY-MM-DD")
		}
		fromDate = parsedFrom
	}
	
	// Parse 'to' parameter if provided
	toParam := r.URL.Query().Get("to")
	if toParam != "" {
		parsedTo, err := time.Parse("2006-01-02", toParam)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid 'to' date format. Use YYYY-MM-DD")
		}
		// Set time to end of day for inclusive range
		toDate = parsedTo.Add(24*time.Hour - time.Second)
	}
	
	// Validate date range
	if fromDate.After(toDate) {
		return time.Time{}, time.Time{}, fmt.Errorf("'from' date cannot be after 'to' date")
	}
	
	return fromDate, toDate, nil
}

// handleHealth handles the health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"version": "1.0.0",
	}); err != nil {
		s.logger.Error("Error encoding health check response", "error", err)
	}
}

// writeJSONResponse writes a JSON response with the given status code
func (s *Server) writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error("Error encoding JSON response", "error", err)
	}
}

// writeErrorResponse writes a JSON error response with the given status code
func (s *Server) writeErrorResponse(w http.ResponseWriter, message string, statusCode int, details string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(ErrorResponse{
		Error:   message,
		Code:    statusCode,
		Details: details,
	}); err != nil {
		s.logger.Error("Error encoding JSON error response", "error", err)
	}
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header
	if xForwardedFor := r.Header.Get("X-Forwarded-For"); xForwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, the first one is the original client
		ips := strings.Split(xForwardedFor, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check for X-Real-IP header
	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	// Extract IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr // Return as is if we can't split
	}
	return ip
}

// HashCode converts a string to a 64-bit integer hash
// This is a simple implementation and is not cryptographically secure
func HashCode(s string) int64 {
	var h int64 = 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int64(s[i])
	}
	return h
}

// GenerateUniqueID generates a unique ID for a new user
func GenerateUniqueID() string {
	now := time.Now()
	return fmt.Sprintf("api_%d", now.UnixNano())
}


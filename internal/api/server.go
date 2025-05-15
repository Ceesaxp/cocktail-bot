package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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
	tokenMutex   sync.RWMutex
	stopWatcher  chan struct{}
}

// ServiceInterface defines the required methods from the service layer
type ServiceInterface interface {
	CheckEmailStatus(ctx any, userID int64, email string) (string, *domain.User, error)
	RedeemCocktail(ctx any, userID int64, email string) (time.Time, error)
	UpdateUser(ctx any, user *domain.User) error
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

// New creates a new API server
func New(cfg *config.Config, svc ServiceInterface, log *logger.Logger) (*Server, error) {
	// Load authentication tokens from file if necessary
	if err := cfg.LoadAuthTokens(); err != nil {
		return nil, fmt.Errorf("failed to load auth tokens: %w", err)
	}
	log.Info("Tokens loaded", fmt.Sprintf("%v", cfg.API.AuthTokens))

	// Create a dedicated rate limiter for API requests
	limiter := ratelimit.New(cfg.API.RateLimitPerMin, cfg.API.RateLimitPerHour)

	// Create auth provider
	authProvider := NewAuthProvider(cfg.API.AuthTokens)

	mux := http.NewServeMux()

	server := &Server{
		config:       cfg,
		logger:       log,
		service:      svc,
		limiter:      limiter,
		authProvider: authProvider,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%d", cfg.API.Port),
			Handler: mux,
		},
		stopWatcher: nil,
	}

	// Register routes
	mux.HandleFunc("/api/v1/email", server.handleEmail)
	mux.HandleFunc("/api/health", server.handleHealth)
	mux.HandleFunc("/api/reload-tokens", server.handleReloadTokens)

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

	// Start the token file watcher
	s.stopWatcher = make(chan struct{})
	go s.watchTokensFile()

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

	// Stop the token file watcher
	if s.stopWatcher != nil {
		close(s.stopWatcher)
	}

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

	// Store in database using service
	if err := s.service.UpdateUser(ctx, newUser); err != nil {
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

// watchTokensFile watches the tokens file for changes and reloads it when it changes
func (s *Server) watchTokensFile() {
	// Set up file checking at a regular interval
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Get the initial file info
	var lastModTime time.Time
	tokensFile := s.config.API.TokensFile

	if info, err := os.Stat(tokensFile); err == nil {
		lastModTime = info.ModTime()
	}

	for {
		select {
		case <-ticker.C:
			// Check if the file has changed
			info, err := os.Stat(tokensFile)
			if err != nil {
				s.logger.Error("Error checking tokens file", "file", tokensFile, "error", err)
				continue
			}

			// If the file has been modified, reload the tokens
			if info.ModTime().After(lastModTime) {
				s.logger.Info("Tokens file changed, reloading", "file", tokensFile)
				if err := s.reloadTokens(); err != nil {
					s.logger.Error("Error reloading tokens", "error", err)
				} else {
					lastModTime = info.ModTime()
				}
			}
		case <-s.stopWatcher:
			return
		}
	}
}

// reloadTokens reloads the tokens from the file
func (s *Server) reloadTokens() error {
	// Reload tokens from file
	if err := s.config.LoadAuthTokens(); err != nil {
		return err
	}

	// Update auth provider with new tokens
	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()

	// Create a new auth provider with the updated tokens
	s.authProvider = NewAuthProvider(s.config.API.AuthTokens)
	s.logger.Info("Tokens reloaded", fmt.Sprintf("%v", s.config.API.AuthTokens))
	return nil
}

// handleReloadTokens handles the manual token reload endpoint
func (s *Server) handleReloadTokens(w http.ResponseWriter, r *http.Request) {
	// Only allow POST method
	if r.Method != http.MethodPost {
		s.writeErrorResponse(w, "Method not allowed", http.StatusMethodNotAllowed, "Only POST method is allowed")
		return
	}

	// Authenticate request (only admin can reload tokens)
	apiKey := r.Header.Get("Authorization")
	if len(apiKey) > 7 && strings.HasPrefix(strings.ToLower(apiKey), "bearer ") {
		apiKey = apiKey[7:] // Remove 'Bearer ' prefix
	}

	if !s.authProvider.Authenticate(apiKey) {
		s.writeErrorResponse(w, "Unauthorized", http.StatusUnauthorized, "Invalid or missing authentication token")
		return
	}

	// Reload tokens
	if err := s.reloadTokens(); err != nil {
		s.writeErrorResponse(w, "Internal server error", http.StatusInternalServerError, fmt.Sprintf("Error reloading tokens: %v", err))
		return
	}

	// Return success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Tokens reloaded successfully",
	}); err != nil {
		s.logger.Error("Error encoding token reload response", "error", err)
	}
}

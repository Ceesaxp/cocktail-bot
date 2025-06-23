package webui

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/api"
	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/domain"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

type Server struct {
	config       *config.Config
	logger       *logger.Logger
	httpServer   *http.Server
	authProvider *api.AuthProvider
	templates    *template.Template
	apiURL       string
	apiToken     string // Store first available token for API calls
	running      bool
}

func New(cfg *config.Config, log *logger.Logger) (*Server, error) {
	// Initialize templates
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	// Load authentication tokens if configured in tokens file
	if err := cfg.LoadAuthTokens(); err != nil {
		log.Warn("Error loading auth tokens from file", "error", err)
	}

	// Share auth tokens with API - WebUI uses the same tokens
	if len(cfg.API.AuthTokens) == 0 {
		log.Warn("No auth tokens configured - WebUI authentication will be unavailable")
	} else {
		log.Info("WebUI auth tokens configured", "count", len(cfg.API.AuthTokens))
	}

	// Create auth provider with tokens from config (shared with API)
	authProvider := api.NewAuthProvider(cfg.API.AuthTokens)

	// Store first token for API calls
	var apiToken string
	if len(cfg.API.AuthTokens) > 0 {
		apiToken = cfg.API.AuthTokens[0]
	}

	// Create HTTP server
	mux := http.NewServeMux()

	// Configure bind address
	bindAddr := fmt.Sprintf("%s:%d", cfg.WebUI.Host, cfg.WebUI.Port)
	log.Info("Web UI will bind to", "address", bindAddr)

	// Determine API URL (internal communication)
	apiURL := fmt.Sprintf("http://%s:%d", cfg.API.Host, cfg.API.Port)
	if cfg.API.Host == "" || cfg.API.Host == "0.0.0.0" {
		apiURL = fmt.Sprintf("http://localhost:%d", cfg.API.Port)
	}

	server := &Server{
		config:       cfg,
		logger:       log,
		templates:    tmpl,
		authProvider: authProvider,
		apiURL:       apiURL,
		apiToken:     apiToken,
		httpServer: &http.Server{
			Addr:    bindAddr,
			Handler: mux,
		},
	}

	// Register routes
	// Static files
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Test endpoint to debug
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("WebUI is running"))
	})

	// Web pages with authentication
	mux.HandleFunc("/", server.authMiddleware(server.handleDashboard))
	mux.HandleFunc("/users", server.authMiddleware(server.handleAllUsers))
	mux.HandleFunc("/redeemed", server.authMiddleware(server.handleRedeemedUsers))

	// Authentication
	mux.HandleFunc("/login", server.handleLogin)
	mux.HandleFunc("/logout", server.handleLogout)

	return server, nil
}

func (s *Server) Start() error {
	if s.running {
		return fmt.Errorf("server is already running")
	}

	// Check if Web UI is enabled
	if !s.config.WebUI.Enabled {
		s.logger.Info("Web UI server is disabled in configuration")
		return nil
	}

	s.running = true
	s.logger.Info("Starting Web UI server", "port", s.config.WebUI.Port)

	// Start server in a separate goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Web UI server error", "error", err)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	if !s.running {
		return nil
	}

	s.logger.Info("Stopping Web UI server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("error shutting down server: %w", err)
	}

	s.running = false
	return nil
}

// Middleware for authentication
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check if user is authenticated via cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			// Redirect to login page
			s.logger.Debug("No auth token cookie, redirecting to login", "path", r.URL.Path)
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}

		// Validate the token
		if !s.authProvider.Authenticate(cookie.Value) {
			// Clear invalid cookie and redirect to login
			http.SetCookie(w, &http.Cookie{
				Name:     "auth_token",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
			})
			http.Redirect(w, r, "/login?redirect="+r.URL.Path, http.StatusSeeOther)
			return
		}

		// Pass request to next handler
		next(w, r)
	}
}

// handleLogin handles the login page and form submission
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" && s.authProvider.Authenticate(cookie.Value) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Get redirect URL from query parameter
	redirect := r.URL.Query().Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	// Handle POST (login form submission)
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing form", http.StatusBadRequest)
			return
		}

		token := r.FormValue("token")

		// Authenticate token
		if s.authProvider.Authenticate(token) {
			// Set auth cookie with the token
			cookie := &http.Cookie{
				Name:     "auth_token",
				Value:    token,
				Path:     "/",
				MaxAge:   86400, // 24 hours
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			}
			http.SetCookie(w, cookie)

			// Redirect to requested page
			http.Redirect(w, r, redirect, http.StatusSeeOther)
			return
		}

		// Authentication failed
		s.renderLoginPage(w, "Invalid authentication token", redirect)
		return
	}

	// Display login page
	s.renderLoginPage(w, "", redirect)
}

// handleLogout handles user logout
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear auth token cookie
	cookie := &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1, // Delete cookie
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// handleDashboard displays the dashboard page
func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Calculate date ranges
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7).Format("2006-01-02")
	oneMonthAgo := now.AddDate(0, -1, 0).Format("2006-01-02")
	oneYearAgo := now.AddDate(-1, 0, 0).Format("2006-01-02")
	today := now.Format("2006-01-02")

	// Fetch data from API endpoints in parallel
	type result struct {
		name string
		data any
		err  error
	}

	results := make(chan result, 4)

	// Fetch all users (last year)
	go func() {
		resp, err := s.callAPI("/api/v1/report/all", map[string]string{"from": oneYearAgo, "to": today})
		results <- result{"all", resp, err}
	}()

	// Fetch redeemed users (last year)
	go func() {
		resp, err := s.callAPI("/api/v1/report/redeemed", map[string]string{"from": oneYearAgo, "to": today})
		results <- result{"redeemed", resp, err}
	}()

	// Fetch recently added users (last month)
	go func() {
		resp, err := s.callAPI("/api/v1/report/added", map[string]string{"from": oneMonthAgo, "to": today})
		results <- result{"last_month", resp, err}
	}()

	// Fetch recently added users (last week)
	go func() {
		resp, err := s.callAPI("/api/v1/report/added", map[string]string{"from": oneWeekAgo, "to": today})
		results <- result{"last_week", resp, err}
	}()

	// Collect results
	stats := make(map[string]int)
	for range 4 {
		r := <-results
		if r.err != nil {
			s.logger.Error("Error fetching data", "endpoint", r.name, "error", r.err)
			continue
		}

		if reportResp, ok := r.data.(map[string]any); ok {
			if count, ok := reportResp["count"].(float64); ok {
				stats[r.name] = int(count)
			}
		}
	}

	// Ensure we have all stats
	if _, ok := stats["all"]; !ok {
		stats["all"] = 0
	}
	if _, ok := stats["redeemed"]; !ok {
		stats["redeemed"] = 0
	}
	// Rename for template compatibility
	stats["total"] = stats["all"]
	delete(stats, "all")

	// Render dashboard template
	data := map[string]any{
		"Stats": stats,
		"Title": "Dashboard",
		"User":  getUserFromCookie(r),
	}
	
	// Debug: log the data being passed
	s.logger.Debug("Rendering dashboard", "stats", stats, "user", getUserFromCookie(r))
	
	// Use the new render method
	err := s.renderTemplate(w, "dashboard.html", data)
	if err != nil {
		s.logger.Error("Error rendering dashboard", "error", err)
		http.Error(w, fmt.Sprintf("Error rendering page: %v", err), http.StatusInternalServerError)
	}
}

// handleAllUsers displays all users
func (s *Server) handleAllUsers(w http.ResponseWriter, r *http.Request) {
	// Get date range from query params or use defaults
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().AddDate(-1, 0, 0).Format("2006-01-02") // Last year
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	// Fetch all users from API
	resp, err := s.callAPI("/api/v1/report/all", map[string]string{"from": from, "to": to})
	if err != nil {
		s.logger.Error("Error getting all users", "error", err)
		http.Error(w, "Error loading user data", http.StatusInternalServerError)
		return
	}

	// Extract users from response
	var users []*domain.User
	if reportResp, ok := resp.(map[string]any); ok {
		if usersData, ok := reportResp["users"].([]any); ok {
			for _, u := range usersData {
				if userMap, ok := u.(map[string]any); ok {
					user := &domain.User{}
					if id, ok := userMap["id"].(string); ok {
						user.ID = id
					}
					if email, ok := userMap["email"].(string); ok {
						user.Email = email
					}
					if dateAdded, ok := userMap["date_added"].(string); ok {
						if t, err := time.Parse(time.RFC3339, dateAdded); err == nil {
							user.DateAdded = t
						}
					}
					if redeemed, ok := userMap["redeemed"].(string); ok && redeemed != "" {
						if t, err := time.Parse(time.RFC3339, redeemed); err == nil {
							user.Redeemed = &t
						}
					}
					users = append(users, user)
				}
			}
		}
	}

	// Render users page
	s.renderUsersPage(w, users, "All Users")
}

// handleRedeemedUsers displays users who have redeemed their cocktails
func (s *Server) handleRedeemedUsers(w http.ResponseWriter, r *http.Request) {
	// Get date range from query params or use defaults
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" {
		from = time.Now().AddDate(-1, 0, 0).Format("2006-01-02") // Last year
	}
	if to == "" {
		to = time.Now().Format("2006-01-02")
	}

	// Fetch redeemed users from API
	resp, err := s.callAPI("/api/v1/report/redeemed", map[string]string{"from": from, "to": to})
	if err != nil {
		s.logger.Error("Error getting redeemed users", "error", err)
		http.Error(w, "Error loading redeemed user data", http.StatusInternalServerError)
		return
	}

	// Extract users from response
	var users []*domain.User
	if reportResp, ok := resp.(map[string]any); ok {
		if usersData, ok := reportResp["users"].([]any); ok {
			for _, u := range usersData {
				if userMap, ok := u.(map[string]any); ok {
					user := &domain.User{}
					if id, ok := userMap["id"].(string); ok {
						user.ID = id
					}
					if email, ok := userMap["email"].(string); ok {
						user.Email = email
					}
					if dateAdded, ok := userMap["date_added"].(string); ok {
						if t, err := time.Parse(time.RFC3339, dateAdded); err == nil {
							user.DateAdded = t
						}
					}
					if redeemed, ok := userMap["redeemed"].(string); ok && redeemed != "" {
						if t, err := time.Parse(time.RFC3339, redeemed); err == nil {
							user.Redeemed = &t
						}
					}
					users = append(users, user)
				}
			}
		}
	}

	// Render redeemed users page
	s.renderUsersPage(w, users, "Redeemed Cocktails")
}

// getUserFromCookie gets the token identifier from the auth cookie
// Since we're using tokens, we'll return a generic "Admin" identifier
func getUserFromCookie(r *http.Request) string {
	_, err := r.Cookie("auth_token")
	if err != nil {
		return ""
	}
	// For display purposes, return "Admin" when authenticated
	return "Admin"
}

// renderTemplate renders a template with the layout
func (s *Server) renderTemplate(w http.ResponseWriter, templateName string, data map[string]any) error {
	// Create a buffer to capture any template errors
	var buf bytes.Buffer
	
	// The layout expects a "content" template to be defined
	// We need to parse both the layout and the specific template together
	
	// Parse layout first
	layoutContent := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cocktail Bot - {{.Title}}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css">
    <link rel="stylesheet" href="/static/css/styles.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.3.0/dist/chart.umd.min.js"></script>
</head>
<body>
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
        <div class="container">
            <a class="navbar-brand" href="/">🍹 Cocktail Bot</a>
            <button class="navbar-toggler" type="button" data-bs-toggle="collapse" data-bs-target="#navbarNav">
                <span class="navbar-toggler-icon"></span>
            </button>
            <div class="collapse navbar-collapse" id="navbarNav">
                <ul class="navbar-nav me-auto">
                    <li class="nav-item">
                        <a class="nav-link" href="/">Dashboard</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/users">All Users</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/redeemed">Redeemed Cocktails</a>
                    </li>
                </ul>
                {{if .User}}
                <div class="d-flex">
                    <span class="navbar-text me-3">
                        Welcome, {{.User}}
                    </span>
                    <a href="/logout" class="btn btn-outline-light btn-sm">Logout</a>
                </div>
                {{end}}
            </div>
        </div>
    </nav>

    <div class="container mt-4">
        <h1 class="mb-4">{{.Title}}</h1>
        {{template "content" .}}
    </div>

    <footer class="footer mt-auto py-3 bg-light">
        <div class="container text-center">
            <span class="text-muted">Cocktail Bot Admin Interface</span>
        </div>
    </footer>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`

	// Execute template with layout
	if templateName == "dashboard.html" {
		fullTemplate := layoutContent + `
{{define "content"}}` + getDashboardContent() + `{{end}}`
		
		tmpl, err := template.New("layout").Parse(fullTemplate)
		if err != nil {
			return fmt.Errorf("error parsing template: %w", err)
		}
		
		err = tmpl.Execute(&buf, data)
		if err != nil {
			return fmt.Errorf("error executing template: %w", err)
		}
	} else {
		// For other templates, use the embedded ones
		err := s.templates.ExecuteTemplate(&buf, templateName, data)
		if err != nil {
			return err
		}
	}
	
	// Write the response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := w.Write(buf.Bytes())
	return err
}

// getDashboardContent returns the dashboard template content
func getDashboardContent() string {
	return `<div class="row">
    <!-- Stats Cards -->
    <div class="col-md-3 mb-4">
        <div class="card text-center h-100 border-primary">
            <div class="card-header bg-primary text-white">
                Total Users
            </div>
            <div class="card-body">
                <h2 class="card-title">{{.Stats.total}}</h2>
                <p class="card-text">Total registered users</p>
            </div>
        </div>
    </div>

    <div class="col-md-3 mb-4">
        <div class="card text-center h-100 border-success">
            <div class="card-header bg-success text-white">
                Redeemed Cocktails
            </div>
            <div class="card-body">
                <h2 class="card-title">{{.Stats.redeemed}}</h2>
                <p class="card-text">Users who redeemed their cocktails</p>
            </div>
        </div>
    </div>

    <div class="col-md-3 mb-4">
        <div class="card text-center h-100 border-info">
            <div class="card-header bg-info text-white">
                Last Month
            </div>
            <div class="card-body">
                <h2 class="card-title">{{.Stats.last_month}}</h2>
                <p class="card-text">New users in the last 30 days</p>
            </div>
        </div>
    </div>

    <div class="col-md-3 mb-4">
        <div class="card text-center h-100 border-warning">
            <div class="card-header bg-warning text-dark">
                Last Week
            </div>
            <div class="card-body">
                <h2 class="card-title">{{.Stats.last_week}}</h2>
                <p class="card-text">New users in the last 7 days</p>
            </div>
        </div>
    </div>
</div>

<!-- Quick Actions -->
<div class="row mt-2">
    <div class="col-12">
        <div class="card">
            <div class="card-header">
                Quick Actions
            </div>
            <div class="card-body">
                <div class="d-flex gap-2 flex-wrap">
                    <a href="/users" class="btn btn-primary">View All Users</a>
                    <a href="/redeemed" class="btn btn-success">View Redeemed Cocktails</a>
                </div>
            </div>
        </div>
    </div>
</div>`
}

// renderLoginPage renders the login page with optional error message
func (s *Server) renderLoginPage(w http.ResponseWriter, errorMsg string, redirect string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	errorHTML := ""
	if errorMsg != "" {
		errorHTML = `<div class="alert alert-danger" role="alert">` + errorMsg + `</div>`
	}

	loginHTML := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cocktail Bot - Login</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css">
</head>
<body class="bg-light d-flex align-items-center min-vh-100">
    <div class="container">
        <div class="row justify-content-center">
            <div class="col-md-6 col-lg-4">
                <div class="card shadow">
                    <div class="card-header bg-primary text-white">
                        <h3 class="card-title text-center mb-0">🍹 Cocktail Bot Login</h3>
                    </div>
                    <div class="card-body">
                        ` + errorHTML + `
                        <form method="POST" action="/login">
                            <input type="hidden" name="redirect" value="` + redirect + `">
                            <div class="mb-3">
                                <label for="token" class="form-label">Authentication Token</label>
                                <input type="password" class="form-control" id="token" name="token" placeholder="Enter your API token" required autofocus>
                                <small class="form-text text-muted">Use the same token as configured for API access</small>
                            </div>
                            <div class="d-grid">
                                <button type="submit" class="btn btn-primary">Login</button>
                            </div>
                        </form>
                    </div>
                    <div class="card-footer text-center">
                        <small class="text-muted">Enter your authentication token to access the dashboard</small>
                    </div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`

	w.Write([]byte(loginHTML))
}

// renderUsersPage renders a page with a list of users
func (s *Server) renderUsersPage(w http.ResponseWriter, users []*domain.User, title string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	// Build user rows HTML
	var userRows string
	for _, user := range users {
		redeemedText := "No"
		redeemedClass := ""
		if user.Redeemed != nil {
			redeemedText = user.Redeemed.Format("Jan 02, 2006 15:04")
			redeemedClass = "text-success"
		}
		
		userRows += fmt.Sprintf(`
		<tr>
			<td>%s</td>
			<td>%s</td>
			<td>%s</td>
			<td class="%s">%s</td>
		</tr>`, user.ID, user.Email, user.DateAdded.Format("Jan 02, 2006 15:04"), redeemedClass, redeemedText)
	}
	
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cocktail Bot - %s</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css">
    <link rel="stylesheet" href="/static/css/styles.css">
</head>
<body>
    <nav class="navbar navbar-expand-lg navbar-dark bg-dark">
        <div class="container">
            <a class="navbar-brand" href="/">🍹 Cocktail Bot</a>
            <div class="collapse navbar-collapse" id="navbarNav">
                <ul class="navbar-nav me-auto">
                    <li class="nav-item">
                        <a class="nav-link" href="/">Dashboard</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/users">All Users</a>
                    </li>
                    <li class="nav-item">
                        <a class="nav-link" href="/redeemed">Redeemed Cocktails</a>
                    </li>
                </ul>
                <div class="d-flex">
                    <span class="navbar-text me-3">Welcome, Admin</span>
                    <a href="/logout" class="btn btn-outline-light btn-sm">Logout</a>
                </div>
            </div>
        </div>
    </nav>

    <div class="container mt-4">
        <h1 class="mb-4">%s</h1>
        <div class="card">
            <div class="card-header">
                Total: %d users
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Email</th>
                                <th>Date Added</th>
                                <th>Redeemed</th>
                            </tr>
                        </thead>
                        <tbody>
                            %s
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>

    <footer class="footer mt-auto py-3 bg-light">
        <div class="container text-center">
            <span class="text-muted">Cocktail Bot Admin Interface</span>
        </div>
    </footer>

    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/js/bootstrap.bundle.min.js"></script>
</body>
</html>`, title, title, len(users), userRows)
	
	w.Write([]byte(html))
}

// callAPI makes a request to the API and returns the parsed JSON response
func (s *Server) callAPI(endpoint string, params map[string]string) (any, error) {
	// Build URL with query parameters
	req, err := http.NewRequest("GET", s.apiURL+endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// Add authentication header
	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Accept", "application/json")

	// Make the request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse JSON response
	var result any
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// Check for error response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return result, nil
}

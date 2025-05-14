package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/config"
	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

func main() {
	fmt.Println("Cocktail Bot Configuration Demo")
	fmt.Println("==============================")

	// Parse command line arguments
	configPath := flag.String("config", "", "Path to config file (defaults to config.yaml)")
	createExample := flag.Bool("create-example", false, "Create example config file")
	flag.Parse()

	// Create example config file if requested
	if *createExample {
		createExampleConfig()
		return
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create logger from config
	log := logger.FromConfig(cfg.LogLevel)
	log.Info("Configuration loaded successfully")

	// Display configuration
	fmt.Println("Configuration Summary:")
	fmt.Printf("- Log Level: %s\n", cfg.LogLevel)
	fmt.Printf("- Telegram User: %s\n", cfg.Telegram.User)
	fmt.Printf("- Telegram Token: %s\n", maskToken(cfg.Telegram.Token))
	fmt.Printf("- Database Type: %s\n", cfg.Database.Type)
	fmt.Printf("- Connection String: %s\n", cfg.Database.ConnectionString)
	fmt.Printf("- Rate Limit (per minute): %d\n", cfg.RateLimiting.RequestsPerMinute)
	fmt.Printf("- Rate Limit (per hour): %d\n", cfg.RateLimiting.RequestsPerHour)

	// Log some information
	log.Debug("Debug logging enabled", "timestamp", time.Now())
	log.Info("Database configured", "type", cfg.Database.Type, "connection", cfg.Database.ConnectionString)
	log.Warn("Rate limiting is active", "per_minute", cfg.RateLimiting.RequestsPerMinute, "per_hour", cfg.RateLimiting.RequestsPerHour)

	// Show supported database types
	fmt.Println("\nSupported Database Types:")
	for _, dbType := range config.SupportedDatabaseTypes() {
		fmt.Printf("- %s\n", dbType)
	}

	// Create component-specific loggers
	dbLogger := log.WithPrefix("DATABASE")
	apiLogger := log.WithPrefix("API")

	dbLogger.Info("Database logger initialized")
	apiLogger.Info("API logger initialized")

	// Environment info
	if cfg.IsProdEnvironment() {
		log.Info("Running in PRODUCTION environment")
	} else {
		log.Info("Running in DEVELOPMENT environment")
	}

	fmt.Println("\nDemo completed successfully!")
}

// createExampleConfig creates an example config.yaml file
func createExampleConfig() {
	content := `# Cocktail Bot Configuration Example
# Generated on ` + time.Now().Format("2006-01-02 15:04:05") + `

# Log level (debug, info, warn, error)
log_level: info

# Telegram settings
telegram:
  # Bot token (get from BotFather)
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  # Bot username
  user: "your_bot_username"

# Database settings
database:
  # Database type (csv, sqlite, googlesheet, postgresql, mysql, mongodb)
  type: "csv"
  # Connection string or path
  connection_string: "./data/users.csv"

# Rate limiting settings
rate_limiting:
  # Maximum requests per minute per user
  requests_per_minute: 10
  # Maximum requests per hour per user
  requests_per_hour: 100
`

	err := os.WriteFile("config.example.yaml", []byte(content), 0644)
	if err != nil {
		fmt.Printf("Error creating example config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Example configuration file created: config.example.yaml")
}

// maskToken masks a token for display
func maskToken(token string) string {
	if len(token) <= 8 {
		return "********"
	}
	return token[:4] + "..." + token[len(token)-4:]
}

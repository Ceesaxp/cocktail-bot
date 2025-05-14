package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary test config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
log_level: debug
telegram:
  token: "test-token"
  user: "test-user"
database:
  type: "csv"
  connection_string: "./data/test.csv"
rate_limiting:
  requests_per_minute: 5
  requests_per_hour: 50
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading from file
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check values
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", cfg.LogLevel)
	}

	if cfg.Telegram.Token != "test-token" {
		t.Errorf("Expected Telegram.Token to be 'test-token', got '%s'", cfg.Telegram.Token)
	}

	if cfg.Telegram.User != "test-user" {
		t.Errorf("Expected Telegram.User to be 'test-user', got '%s'", cfg.Telegram.User)
	}

	if cfg.Database.Type != "csv" {
		t.Errorf("Expected Database.Type to be 'csv', got '%s'", cfg.Database.Type)
	}

	if cfg.Database.ConnectionString != "./data/test.csv" {
		t.Errorf("Expected Database.ConnectionString to be './data/test.csv', got '%s'", cfg.Database.ConnectionString)
	}

	if cfg.RateLimiting.RequestsPerMinute != 5 {
		t.Errorf("Expected RateLimiting.RequestsPerMinute to be 5, got %d", cfg.RateLimiting.RequestsPerMinute)
	}

	if cfg.RateLimiting.RequestsPerHour != 50 {
		t.Errorf("Expected RateLimiting.RequestsPerHour to be 50, got %d", cfg.RateLimiting.RequestsPerHour)
	}
}

func TestLoadFromEnvironment(t *testing.T) {
	// Create a temporary test config file with defaults
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
log_level: info
telegram:
  token: "default-token"
  user: "default-user"
database:
  type: "csv"
  connection_string: "./data/default.csv"
rate_limiting:
  requests_per_minute: 10
  requests_per_hour: 100
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Set environment variables
	os.Setenv("COCKTAILBOT_LOG_LEVEL", "debug")
	os.Setenv("COCKTAILBOT_TELEGRAM_TOKEN", "env-token")
	os.Setenv("COCKTAILBOT_DATABASE_TYPE", "sqlite")
	os.Setenv("COCKTAILBOT_RATE_LIMITING_REQUESTS_PER_MINUTE", "20")
	defer func() {
		os.Unsetenv("COCKTAILBOT_LOG_LEVEL")
		os.Unsetenv("COCKTAILBOT_TELEGRAM_TOKEN")
		os.Unsetenv("COCKTAILBOT_DATABASE_TYPE")
		os.Unsetenv("COCKTAILBOT_RATE_LIMITING_REQUESTS_PER_MINUTE")
	}()

	// Test loading with environment variables
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check values (environment should override file)
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel to be 'debug', got '%s'", cfg.LogLevel)
	}

	if cfg.Telegram.Token != "env-token" {
		t.Errorf("Expected Telegram.Token to be 'env-token', got '%s'", cfg.Telegram.Token)
	}

	if cfg.Database.Type != "sqlite" {
		t.Errorf("Expected Database.Type to be 'sqlite', got '%s'", cfg.Database.Type)
	}

	if cfg.RateLimiting.RequestsPerMinute != 20 {
		t.Errorf("Expected RateLimiting.RequestsPerMinute to be 20, got %d", cfg.RateLimiting.RequestsPerMinute)
	}

	// This should still be the default value from the file
	if cfg.Telegram.User != "default-user" {
		t.Errorf("Expected Telegram.User to be 'default-user', got '%s'", cfg.Telegram.User)
	}
}

func TestGetConfigPath(t *testing.T) {
	// Test with provided path
	providedPath := "provided/path/config.yaml"
	result := GetConfigPath(providedPath)
	if result != providedPath {
		t.Errorf("Expected GetConfigPath to return '%s', got '%s'", providedPath, result)
	}

	// Test with default (since test environment won't have the files)
	result = GetConfigPath("")
	if result != "config.yaml" {
		t.Errorf("Expected GetConfigPath to return 'config.yaml', got '%s'", result)
	}
}

func TestDatabaseHelpers(t *testing.T) {
	cfg := New()
	cfg.Database.Type = "PostgreSQL"

	// Test case conversion
	if cfg.GetDatabaseType() != "postgresql" {
		t.Errorf("Expected GetDatabaseType to return 'postgresql', got '%s'", cfg.GetDatabaseType())
	}

	// Test supported types
	supported := SupportedDatabaseTypes()
	if len(supported) != 6 {
		t.Errorf("Expected 6 supported database types, got %d", len(supported))
	}
	
	// Check if specific types are included
	expectedTypes := map[string]bool{
		"csv":         true,
		"sqlite":      true,
		"googlesheet": true,
		"postgresql":  true,
		"mysql":       true,
		"mongodb":     true,
	}
	
	for _, dbType := range supported {
		if !expectedTypes[dbType] {
			t.Errorf("Unexpected database type: %s", dbType)
		}
	}
}

func TestNewConfig(t *testing.T) {
	cfg := New()
	
	// Check default values
	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}
	
	if cfg.Database.Type != "csv" {
		t.Errorf("Expected default Database.Type to be 'csv', got '%s'", cfg.Database.Type)
	}
	
	if cfg.Database.ConnectionString != "./data/users.csv" {
		t.Errorf("Expected default Database.ConnectionString to be './data/users.csv', got '%s'", cfg.Database.ConnectionString)
	}
	
	if cfg.RateLimiting.RequestsPerMinute != 10 {
		t.Errorf("Expected default RateLimiting.RequestsPerMinute to be 10, got %d", cfg.RateLimiting.RequestsPerMinute)
	}
	
	if cfg.RateLimiting.RequestsPerHour != 100 {
		t.Errorf("Expected default RateLimiting.RequestsPerHour to be 100, got %d", cfg.RateLimiting.RequestsPerHour)
	}
}

func TestIsProdEnvironment(t *testing.T) {
	cfg := New()
	
	// Test with no environment set
	os.Unsetenv("COCKTAILBOT_ENVIRONMENT")
	if cfg.IsProdEnvironment() {
		t.Error("Expected IsProdEnvironment to return false with no env set")
	}
	
	// Test with production set
	os.Setenv("COCKTAILBOT_ENVIRONMENT", "production")
	if !cfg.IsProdEnvironment() {
		t.Error("Expected IsProdEnvironment to return true with 'production'")
	}
	
	// Test with prod set
	os.Setenv("COCKTAILBOT_ENVIRONMENT", "prod")
	if !cfg.IsProdEnvironment() {
		t.Error("Expected IsProdEnvironment to return true with 'prod'")
	}
	
	// Test with different case
	os.Setenv("COCKTAILBOT_ENVIRONMENT", "PRODUCTION")
	if !cfg.IsProdEnvironment() {
		t.Error("Expected IsProdEnvironment to return true with 'PRODUCTION'")
	}
	
	// Clean up
	os.Unsetenv("COCKTAILBOT_ENVIRONMENT")
}
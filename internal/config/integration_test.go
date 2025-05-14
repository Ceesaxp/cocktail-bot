package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ceesaxp/cocktail-bot/internal/logger"
)

func TestConfigWithLogger(t *testing.T) {
	// Create a temporary test config file with log level info
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
log_level: info
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

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Create logger based on config
	log := logger.NewWithWriter(cfg.LogLevel, &buf)

	// Test logging at different levels
	log.Debug("This is a debug message") // Should not appear with info level
	log.Info("This is an info message")  // Should appear
	log.Error("This is an error message") // Should appear

	output := buf.String()

	// Debug shouldn't be logged at info level
	if debugFound := bytes.Contains([]byte(output), []byte("DEBUG")); debugFound {
		t.Error("Debug message should not appear when log level is set to info")
	}

	// Info should be logged at info level
	if !bytes.Contains([]byte(output), []byte("INFO")) {
		t.Error("Info message should appear when log level is set to info")
	}

	// Error should be logged at info level
	if !bytes.Contains([]byte(output), []byte("ERROR")) {
		t.Error("Error message should appear when log level is set to info")
	}

	// Now change log level in config and test again
	cfg.LogLevel = "debug"
	buf.Reset()
	log = logger.NewWithWriter(cfg.LogLevel, &buf)

	log.Debug("This is a debug message") // Should now appear with debug level
	
	output = buf.String()
	
	// Debug should now be logged at debug level
	if !bytes.Contains([]byte(output), []byte("DEBUG")) {
		t.Error("Debug message should appear when log level is set to debug")
	}
}

func TestLoggerFromConfig(t *testing.T) {
	// Create a temporary test config file with log level info
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	configContent := `
log_level: error
telegram:
  token: "test-token"
  user: "test-user"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Use FromConfig helper
	log := logger.FromConfig(cfg.LogLevel)
	
	// Set output to buffer for testing
	log2 := logger.NewWithWriter(log.GetLevel(), &buf)
	
	// Test that level was correctly set to error
	log2.Debug("Debug message")
	log2.Info("Info message")
	log2.Warn("Warn message")
	log2.Error("Error message")
	
	output := buf.String()
	
	// Only error should be logged
	if bytes.Contains([]byte(output), []byte("DEBUG")) || 
	   bytes.Contains([]byte(output), []byte("INFO")) || 
	   bytes.Contains([]byte(output), []byte("WARN")) {
		t.Error("Only ERROR messages should appear with error log level")
	}
	
	if !bytes.Contains([]byte(output), []byte("ERROR")) {
		t.Error("ERROR message should appear with error log level")
	}
}
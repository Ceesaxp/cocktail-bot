package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Environment variable prefix for configuration overrides
	envPrefix = "COCKTAILBOT_"
)

// Config represents the application configuration
type Config struct {
	LogLevel    string         `yaml:"log_level"`
	Telegram    TelegramConfig `yaml:"telegram"`
	Database    DatabaseConfig `yaml:"database"`
	RateLimiting RateLimitConfig `yaml:"rate_limiting"`
	Language    LanguageConfig `yaml:"language"`
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	Token string `yaml:"token"`
	User  string `yaml:"user"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Type             string `yaml:"type"`
	ConnectionString string `yaml:"connection_string"`
}

// RateLimitConfig holds rate limiting settings
type RateLimitConfig struct {
	RequestsPerMinute int `yaml:"requests_per_minute"`
	RequestsPerHour   int `yaml:"requests_per_hour"`
}

// LanguageConfig holds language settings
type LanguageConfig struct {
	DefaultLanguage string   `yaml:"default_language"`
	Enabled         []string `yaml:"enabled"`
}

// New creates a new default configuration
func New() *Config {
	return &Config{
		LogLevel: "info",
		Telegram: TelegramConfig{},
		Database: DatabaseConfig{
			Type:             "csv",
			ConnectionString: "./data/users.csv",
		},
		RateLimiting: RateLimitConfig{
			RequestsPerMinute: 10,
			RequestsPerHour:   100,
		},
		Language: LanguageConfig{
			DefaultLanguage: "en",
			Enabled:         []string{"en", "es", "fr", "de", "ru"},
		},
	}
}

// Load loads configuration from file and environment variables
func Load(path string) (*Config, error) {
	// Create default config
	cfg := New()

	// If path is provided, load from file
	if path != "" {
		err := loadFromFile(path, cfg)
		if err != nil {
			return nil, err
		}
	}

	// Override with environment variables
	loadFromEnvironment(cfg)

	return cfg, nil
}

// loadFromFile loads configuration from the YAML file
func loadFromFile(path string, cfg *Config) error {
	// Ensure the file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse YAML
	return yaml.Unmarshal(data, cfg)
}

// loadFromEnvironment overrides configuration with environment variables
func loadFromEnvironment(cfg *Config) {
	// Log level
	if value := os.Getenv(envPrefix + "LOG_LEVEL"); value != "" {
		cfg.LogLevel = value
	}

	// Telegram
	if value := os.Getenv(envPrefix + "TELEGRAM_TOKEN"); value != "" {
		cfg.Telegram.Token = value
	}
	if value := os.Getenv(envPrefix + "TELEGRAM_USER"); value != "" {
		cfg.Telegram.User = value
	}

	// Database
	if value := os.Getenv(envPrefix + "DATABASE_TYPE"); value != "" {
		cfg.Database.Type = value
	}
	if value := os.Getenv(envPrefix + "DATABASE_CONNECTION_STRING"); value != "" {
		cfg.Database.ConnectionString = value
	}

	// Rate limiting
	if value := os.Getenv(envPrefix + "RATE_LIMITING_REQUESTS_PER_MINUTE"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			cfg.RateLimiting.RequestsPerMinute = intValue
		}
	}
	if value := os.Getenv(envPrefix + "RATE_LIMITING_REQUESTS_PER_HOUR"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			cfg.RateLimiting.RequestsPerHour = intValue
		}
	}

	// Language
	if value := os.Getenv(envPrefix + "LANGUAGE_DEFAULT"); value != "" {
		cfg.Language.DefaultLanguage = value
	}
	if value := os.Getenv(envPrefix + "LANGUAGE_ENABLED"); value != "" {
		languages := strings.Split(value, ",")
		cfg.Language.Enabled = make([]string, 0, len(languages))
		// Trim spaces from language codes and filter out empty strings
		for _, lang := range languages {
			trimmed := strings.TrimSpace(lang)
			if trimmed != "" {
				cfg.Language.Enabled = append(cfg.Language.Enabled, trimmed)
			}
		}
	}
}

// GetConfigPath returns the config file path based on the provided path or default
func GetConfigPath(configPath string) string {
	if configPath != "" {
		// Use provided path
		return configPath
	}

	// Use default paths
	candidates := []string{
		"config.yaml",
		"config.yml",
		filepath.Join("config", "config.yaml"),
		filepath.Join("config", "config.yml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Return default if no file found
	return "config.yaml"
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// In a real implementation, this would check for required values
	// and validate formats
	return nil
}

// IsProdEnvironment checks if the current environment is production
func (c *Config) IsProdEnvironment() bool {
	env := os.Getenv(envPrefix + "ENVIRONMENT")
	return strings.ToLower(env) == "production" || strings.ToLower(env) == "prod"
}

// GetDatabaseType returns the database type (lowercase)
func (c *Config) GetDatabaseType() string {
	return strings.ToLower(c.Database.Type)
}

// GetDefaultLanguage returns the default language
func (c *Config) GetDefaultLanguage() string {
	if c.Language.DefaultLanguage == "" {
		return "en"
	}
	return c.Language.DefaultLanguage
}

// GetEnabledLanguages returns the list of enabled languages
func (c *Config) GetEnabledLanguages() []string {
	if len(c.Language.Enabled) == 0 {
		return []string{"en"}
	}
	// Make a copy to avoid potential modification of the original slice
	result := make([]string, len(c.Language.Enabled))
	copy(result, c.Language.Enabled)
	return result
}

// IsLanguageEnabled checks if a specific language is enabled
func (c *Config) IsLanguageEnabled(lang string) bool {
	for _, l := range c.Language.Enabled {
		if strings.EqualFold(l, lang) {
			return true
		}
	}
	return false
}

// SupportedDatabaseTypes returns a list of supported database types
func SupportedDatabaseTypes() []string {
	return []string{
		"csv",
		"sqlite",
		"googlesheet",
		"postgresql",
		"mysql",
		"mongodb",
	}
}

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
	LogLevel     string          `yaml:"log_level"`
	Telegram     TelegramConfig  `yaml:"telegram"`
	Database     DatabaseConfig  `yaml:"database"`
	RateLimiting RateLimitConfig `yaml:"rate_limiting"`
	Language     LanguageConfig  `yaml:"language"`
	API          APIConfig       `yaml:"api"`
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

// APIConfig holds REST API configuration
type APIConfig struct {
	Enabled          bool     `yaml:"enabled"`
	Port             int      `yaml:"port"`
	AuthTokens       []string `yaml:"auth_tokens"`
	TokensFile       string   `yaml:"tokens_file"`
	RateLimitPerMin  int      `yaml:"rate_limit_per_min"`
	RateLimitPerHour int      `yaml:"rate_limit_per_hour"`
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
			Enabled:         []string{"en", "es", "fr", "de", "ru", "sr"},
		},
		API: APIConfig{
			Enabled:          false,
			Port:             8080,
			AuthTokens:       []string{},
			TokensFile:       "./api_tokens.yaml",
			RateLimitPerMin:  30,
			RateLimitPerHour: 300,
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

	// API
	if value := os.Getenv(envPrefix + "API_ENABLED"); value != "" {
		cfg.API.Enabled = strings.ToLower(value) == "true" || value == "1"
	}
	if value := os.Getenv(envPrefix + "API_PORT"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			cfg.API.Port = intValue
		}
	}
	if value := os.Getenv(envPrefix + "API_TOKENS_FILE"); value != "" {
		cfg.API.TokensFile = value
	}
	if value := os.Getenv(envPrefix + "API_RATE_LIMIT_PER_MIN"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			cfg.API.RateLimitPerMin = intValue
		}
	}
	if value := os.Getenv(envPrefix + "API_RATE_LIMIT_PER_HOUR"); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil && intValue > 0 {
			cfg.API.RateLimitPerHour = intValue
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

// LoadAuthTokens loads API authentication tokens from the configured tokens file
func (c *Config) LoadAuthTokens() error {
	// Skip if API is not enabled
	if !c.API.Enabled {
		return nil
	}

	// If tokens file is not specified, nothing to load
	if c.API.TokensFile == "" {
		return nil
	}

	// Check if the file exists
	if _, err := os.Stat(c.API.TokensFile); os.IsNotExist(err) {
		return nil // File doesn't exist, not an error
	}

	// Read the file
	data, err := os.ReadFile(c.API.TokensFile)
	if err != nil {
		return err
	}

	// Parse YAML into a temporary struct
	var tokensConfig struct {
		AuthTokens []string `yaml:"auth_tokens"`
	}

	err = yaml.Unmarshal(data, &tokensConfig)
	if err != nil {
		return err
	}

	// Update config with loaded tokens
	if len(tokensConfig.AuthTokens) > 0 {
		c.API.AuthTokens = tokensConfig.AuthTokens
	}

	return nil
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

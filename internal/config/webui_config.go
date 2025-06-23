package config

// WebUIConfig contains configuration for the web UI
type WebUIConfig struct {
	// Enable or disable the web UI
	Enabled bool `yaml:"enabled" env:"WEBUI_ENABLED"`

	// Host to bind to (default is 0.0.0.0)
	Host string `yaml:"host" env:"WEBUI_HOST"`

	// Port to listen on
	Port int `yaml:"port" env:"WEBUI_PORT"`

	// Session secret for cookies (optional, auto-generated if not provided)
	SessionSecret string `yaml:"session_secret" env:"WEBUI_SESSION_SECRET"`

	// Template directory path (optional for embedded templates)
	TemplateDir string `yaml:"template_dir" env:"WEBUI_TEMPLATE_DIR"`

	// Static files directory path (optional for embedded static files)
	StaticDir string `yaml:"static_dir" env:"WEBUI_STATIC_DIR"`
}

// DefaultWebUIConfig returns the default WebUI configuration
func DefaultWebUIConfig() WebUIConfig {
	return WebUIConfig{
		Enabled:       false,
		Host:          "0.0.0.0",
		Port:          8081,
		SessionSecret: "",
		TemplateDir:   "", // Empty means use embedded templates
		StaticDir:     "", // Empty means use embedded static files
	}
}

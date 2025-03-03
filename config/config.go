package config

// Config holds the application configuration
type Config struct {
	DeepseekAPIKey string
	APIEndpoint    string
	FileHandler    interface{}
}

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		DeepseekAPIKey: "", // Set this from environment variable or config file
		APIEndpoint:    "https://api.deepseek.io/v1/chat/completions", // Updated to correct endpoint
		FileHandler:    nil,
	}
}
package config

import "time"

// Config holds the application configuration
type Config struct {
	DeepseekAPIKey string
	APIEndpoint    string
	FileHandler    interface{}
	APIRateLimit   time.Duration // Duration to wait between API calls
	MaxRetries     int           // Maximum number of retries for failed API calls
}

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		DeepseekAPIKey: "", // Set this from environment variable or config file
		APIEndpoint:    "https://api.deepseek.com/chat/completions", // Updated to correct endpoint
		FileHandler:    nil,
		APIRateLimit:   time.Second * 1, // Default: 1 second between API calls
		MaxRetries:     3,               // Default: retry 3 times
	}
}
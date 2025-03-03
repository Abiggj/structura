package config

import (
	"github.com/Abiggj/structura/types"
	"time"
)

// Config holds the application configuration
type Config struct {
	// API Configuration
	APIType        types.APIType
	APIModel       string
	DeepseekAPIKey string
	OpenAIAPIKey   string
	GeminiAPIKey   string
	
	// API Endpoints
	DeepseekEndpoint string
	OpenAIEndpoint   string
	GeminiEndpoint   string
	
	// Common Config
	FileHandler    interface{}
	APIRateLimit   time.Duration // Duration to wait between API calls
	MaxRetries     int           // Maximum number of retries for failed API calls
}

// NewConfig creates a new configuration
func NewConfig() *Config {
	return &Config{
		// Default API settings
		APIType:        types.APITypeDeepseek,
		APIModel:       "deepseek-chat",
		DeepseekAPIKey: "",
		OpenAIAPIKey:   "",
		GeminiAPIKey:   "",
		
		// API Endpoints
		DeepseekEndpoint: "https://api.deepseek.com/chat/completions",
		OpenAIEndpoint:   "https://api.openai.com/v1/chat/completions",
		GeminiEndpoint:   "https://generativelanguage.googleapis.com/v1/models/gemini-pro:generateContent",
		
		// Common Config
		FileHandler:    nil,
		APIRateLimit:   time.Second * 1, // Default: 1 second between API calls
		MaxRetries:     3,               // Default: retry 3 times
	}
}

// GetActiveEndpoint returns the API endpoint for the currently selected API type
func (c *Config) GetActiveEndpoint() string {
	switch c.APIType {
	case types.APITypeChatGPT:
		return c.OpenAIEndpoint
	case types.APITypeGemini:
		return c.GeminiEndpoint
	default:
		return c.DeepseekEndpoint
	}
}

// GetActiveAPIKey returns the API key for the currently selected API type
func (c *Config) GetActiveAPIKey() string {
	switch c.APIType {
	case types.APITypeChatGPT:
		return c.OpenAIAPIKey
	case types.APITypeGemini:
		return c.GeminiAPIKey
	default:
		return c.DeepseekAPIKey
	}
}
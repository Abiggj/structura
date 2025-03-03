package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Abiggj/structura/config"
	"github.com/Abiggj/structura/filehandler"
	"github.com/go-resty/resty/v2"
	"time"
)

// DeepseekClient is a client for the DeepSeek API
type DeepseekClient struct {
	Config      *config.Config
	Client      *resty.Client
	lastAPICall time.Time
}

// DeepseekMessage represents a message in the DeepSeek API request
type DeepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepseekRequest represents the structure of a request to DeepSeek API
type DeepseekRequest struct {
	Model    string            `json:"model"`
	Messages []DeepseekMessage `json:"messages"`
}

// DeepseekResponse represents the structure of a response from DeepSeek API
type DeepseekResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewDeepseekClient creates a new DeepSeek API client
func NewDeepseekClient(cfg *config.Config) *DeepseekClient {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.DeepseekAPIKey))

	return &DeepseekClient{
		Config:      cfg,
		Client:      client,
		lastAPICall: time.Now().Add(-cfg.APIRateLimit), // Initialize to allow immediate first call
	}
}

// APIError represents a structured API error
type APIError struct {
	StatusCode int
	Message    string
	IsRateLimit bool
	IsInvalidKey bool
	IsNetworkError bool
	RawResponse string
}

func (e *APIError) Error() string {
	return e.Message
}

// enforceRateLimit ensures the API rate limit is respected
func (dc *DeepseekClient) enforceRateLimit() {
	elapsed := time.Since(dc.lastAPICall)
	if elapsed < dc.Config.APIRateLimit {
		// Wait for the remaining time
		time.Sleep(dc.Config.APIRateLimit - elapsed)
	}
	dc.lastAPICall = time.Now()
}

// makeAPIRequest makes an API request with rate limiting and retries
func (dc *DeepseekClient) makeAPIRequest(req interface{}) (*resty.Response, error) {
	var lastErr error
	var resp *resty.Response

	for attempt := 0; attempt < dc.Config.MaxRetries; attempt++ {
		// Enforce rate limit before making the request
		dc.enforceRateLimit()

		// Make the request
		resp, err := dc.Client.R().
			SetBody(req).
			Post(dc.Config.APIEndpoint)

		if err == nil {
			// Handle successful response
			if resp.StatusCode() == 200 {
				return resp, nil
			}

			// Handle API-level errors
			apiErr := &APIError{
				StatusCode: resp.StatusCode(),
				RawResponse: resp.String(),
			}

			switch resp.StatusCode() {
			case 401:
				apiErr.Message = "API authentication failed: Invalid API key"
				apiErr.IsInvalidKey = true
				return nil, apiErr
			case 403:
				apiErr.Message = "API access forbidden: API key may be invalid or lacks necessary permissions"
				apiErr.IsInvalidKey = true
				return nil, apiErr
			case 429:
				apiErr.Message = "API rate limit exceeded, will retry"
				apiErr.IsRateLimit = true
				lastErr = apiErr
				// Wait longer before retrying rate limit errors
				time.Sleep(time.Duration(attempt+1) * dc.Config.APIRateLimit)
				continue
			default:
				apiErr.Message = fmt.Sprintf("API request failed with status: %d, body: %s", resp.StatusCode(), resp.String())
				return nil, apiErr
			}
		} else {
			// Handle network errors
			lastErr = &APIError{
				Message: fmt.Sprintf("API request failed: %v", err),
				IsNetworkError: true,
			}
		}

		// Exponential backoff for retries
		if attempt < dc.Config.MaxRetries-1 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return resp, fmt.Errorf("API request failed after %d attempts", dc.Config.MaxRetries)
}

// GenerateDocumentation generates documentation for a file using DeepSeek API
func (dc *DeepseekClient) GenerateDocumentation(file filehandler.FileInfo) (string, error) {
	if dc.Config.DeepseekAPIKey == "" {
		return "", errors.New("DeepSeek API key is not set")
	}

	// Get project type from the config
	projectType := "generic"
	if fileHandler, ok := dc.Config.FileHandler.(*filehandler.FileHandler); ok && fileHandler != nil {
		projectType = string(fileHandler.ProjectType)
	}

	// Prepare the prompt
	prompt := fmt.Sprintf(
		"Please create comprehensive Markdown documentation for the following %s file in a %s project. "+
			"Include information about its purpose, structure, important functions/classes, "+
			"and how it fits into the overall project architecture. "+
			"Format your response as clean Markdown with appropriate headers, code blocks, and explanations.\n\n"+
			"Project type: %s\n"+
			"File path: %s\n\n"+
			"File content:\n```\n%s\n```",
		filehandler.GetFileExtension(file.Path),
		projectType,
		projectType,
		file.Path,
		file.Content,
	)

	// Create the request
	req := DeepseekRequest{
		Model: "deepseek-chat",
		Messages: []DeepseekMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make the request with rate limiting and retries
	resp, err := dc.makeAPIRequest(req)
	if err != nil {
		// Provide more user-friendly errors based on error type
		if apiErr, ok := err.(*APIError); ok {
			if apiErr.IsInvalidKey {
				return "", errors.New("Invalid API key or authentication error. Please check your API key")
			}
			if apiErr.IsRateLimit {
				return "", errors.New("API rate limit exceeded. Please try again later")
			}
			if apiErr.IsNetworkError {
				return "", errors.New("Network error while connecting to API. Please check your internet connection")
			}
		}
		return "", err
	}

	// Parse the response
	var deepseekResp DeepseekResponse
	err = json.Unmarshal(resp.Body(), &deepseekResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(deepseekResp.Choices) == 0 {
		return "", errors.New("API response contains no choices")
	}

	return deepseekResp.Choices[0].Message.Content, nil
}
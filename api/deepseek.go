package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Abiggj/structura/config"
	"github.com/Abiggj/structura/filehandler"
	"github.com/Abiggj/structura/types"
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
			Post(dc.Config.DeepseekEndpoint)

		if err == nil {
			// Handle successful response
			if resp.StatusCode() == 200 {
				return resp, nil
			}

			// Handle API-level errors
			apiErr := &types.APIError{
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
			lastErr = &types.APIError{
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

	// Prepare the prompt with improved instructions for generating technical documentation
	prompt := fmt.Sprintf(
		"Analyze the following %s file in a %s project and generate structured technical documentation that follows these guidelines:\n\n"+
			"1. Begin with a concise summary of the file's purpose and role within the %s project.\n"+
			"2. Document all key structures, interfaces, and types with their fields and purpose.\n"+
			"3. Document each function and method including:\n"+
			"   - Parameters and their types\n"+
			"   - Return values and their significance\n"+
			"   - Error handling approach\n"+
			"   - Any side effects or state changes\n"+
			"4. Explain dependencies and interactions with other components.\n"+
			"5. Include only essential code snippets to illustrate complex logic or patterns.\n"+
			"6. Format as professional Markdown with appropriate headers, lists, and code blocks.\n\n"+
			"File path: %s\n\n"+
			"```%s\n%s\n```",
		filehandler.GetFileExtension(file.Path),
		projectType,
		projectType,
		file.Path,
		filehandler.GetFileExtension(file.Path),
		file.Content,
	)

	// Create the request
	req := DeepseekRequest{
		Model: dc.Config.APIModel,
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
		if apiErr, ok := err.(*types.APIError); ok {
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
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

// ChatGPTClient is a client for the OpenAI ChatGPT API
type ChatGPTClient struct {
	Config      *config.Config
	Client      *resty.Client
	lastAPICall time.Time
}

// ChatGPTMessage represents a message in the ChatGPT API request
type ChatGPTMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTRequest represents the structure of a request to ChatGPT API
type ChatGPTRequest struct {
	Model    string           `json:"model"`
	Messages []ChatGPTMessage `json:"messages"`
}

// ChatGPTResponse represents the structure of a response from ChatGPT API
type ChatGPTResponse struct {
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

// NewChatGPTClient creates a new ChatGPT API client
func NewChatGPTClient(cfg *config.Config) *ChatGPTClient {
	client := resty.New()
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", cfg.OpenAIAPIKey))

	return &ChatGPTClient{
		Config:      cfg,
		Client:      client,
		lastAPICall: time.Now().Add(-cfg.APIRateLimit), // Initialize to allow immediate first call
	}
}

// enforceRateLimit ensures the API rate limit is respected
func (cc *ChatGPTClient) enforceRateLimit() {
	elapsed := time.Since(cc.lastAPICall)
	if elapsed < cc.Config.APIRateLimit {
		// Wait for the remaining time
		time.Sleep(cc.Config.APIRateLimit - elapsed)
	}
	cc.lastAPICall = time.Now()
}

// makeAPIRequest makes an API request with rate limiting and retries
func (cc *ChatGPTClient) makeAPIRequest(req interface{}) (*resty.Response, error) {
	var lastErr error
	var resp *resty.Response

	for attempt := 0; attempt < cc.Config.MaxRetries; attempt++ {
		// Enforce rate limit before making the request
		cc.enforceRateLimit()

		// Make the request
		resp, err := cc.Client.R().
			SetBody(req).
			Post(cc.Config.OpenAIEndpoint)

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
				time.Sleep(time.Duration(attempt+1) * cc.Config.APIRateLimit)
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
		if attempt < cc.Config.MaxRetries-1 {
			time.Sleep(time.Duration(1<<uint(attempt)) * time.Second)
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return resp, fmt.Errorf("API request failed after %d attempts", cc.Config.MaxRetries)
}

// GenerateDocumentation generates documentation for a file using ChatGPT API
func (cc *ChatGPTClient) GenerateDocumentation(file filehandler.FileInfo) (string, error) {
	if cc.Config.OpenAIAPIKey == "" {
		return "", errors.New("OpenAI API key is not set")
	}

	// Get project type from the config
	projectType := "generic"
	if fileHandler, ok := cc.Config.FileHandler.(*filehandler.FileHandler); ok && fileHandler != nil {
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
	req := ChatGPTRequest{
		Model: cc.Config.OpenAIModel,
		Messages: []ChatGPTMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make the request with rate limiting and retries
	resp, err := cc.makeAPIRequest(req)
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
	var chatGPTResp ChatGPTResponse
	err = json.Unmarshal(resp.Body(), &chatGPTResp)
	if err != nil {
		return "", fmt.Errorf("failed to parse API response: %w", err)
	}

	if len(chatGPTResp.Choices) == 0 {
		return "", errors.New("API response contains no choices")
	}

	return chatGPTResp.Choices[0].Message.Content, nil
}
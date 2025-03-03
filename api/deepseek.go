package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Abiggj/structura/config"
	"github.com/Abiggj/structura/filehandler"
	"github.com/go-resty/resty/v2"
)

// DeepseekClient is a client for the DeepSeek API
type DeepseekClient struct {
	Config *config.Config
	Client *resty.Client
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
		Config: cfg,
		Client: client,
	}
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
		Model: "deepseek-coder-v2",
		Messages: []DeepseekMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	// Make the request
	resp, err := dc.Client.R().
		SetBody(req).
		Post(dc.Config.APIEndpoint)

	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API request failed with status: %d, body: %s", resp.StatusCode(), resp.String())
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
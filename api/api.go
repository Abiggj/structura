package api

import (
	"github.com/Abiggj/structura/filehandler"
)

// DocumentationClient defines the interface for documentation API clients
type DocumentationClient interface {
	GenerateDocumentation(file filehandler.FileInfo) (string, error)
}

// APIType represents the type of API to use
type APIType string

const (
	// APITypeDeepseek represents the DeepSeek API
	APITypeDeepseek APIType = "deepseek"
	// APITypeChatGPT represents the ChatGPT/OpenAI API
	APITypeChatGPT APIType = "chatgpt"
	// APITypeGemini represents the Google Gemini API
	APITypeGemini APIType = "gemini"
)

// APITypes returns a list of all supported API types
func APITypes() []APIType {
	return []APIType{
		APITypeDeepseek,
		APITypeChatGPT,
		APITypeGemini,
	}
}

// APIModelMap maps API types to their available models
var APIModelMap = map[APIType][]string{
	APITypeDeepseek: {"deepseek-chat", "deepseek-coder"},
	APITypeChatGPT:  {"gpt-3.5-turbo", "gpt-4", "gpt-4-turbo", "gpt-4o"},
	APITypeGemini:   {"gemini-pro", "gemini-1.5-pro"},
}
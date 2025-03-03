package api

import (
	"fmt"
	"github.com/Abiggj/structura/config"
	"github.com/Abiggj/structura/types"
)

// CreateDocumentationClient creates the appropriate documentation client based on the config
func CreateDocumentationClient(cfg *config.Config) (DocumentationClient, error) {
	switch cfg.APIType {
	case types.APITypeDeepseek:
		return NewDeepseekClient(cfg), nil
	case types.APITypeChatGPT:
		return NewChatGPTClient(cfg), nil
	case types.APITypeGemini:
		// Placeholder for future Gemini implementation
		return nil, fmt.Errorf("Gemini API support coming soon")
	default:
		return nil, fmt.Errorf("unsupported API type: %s", cfg.APIType)
	}
}
package api

import (
	"fmt"
	"github.com/Abiggj/structura/config"
)

// CreateDocumentationClient creates the appropriate documentation client based on the config
func CreateDocumentationClient(cfg *config.Config) (DocumentationClient, error) {
	switch cfg.APIType {
	case APITypeDeepseek:
		return NewDeepseekClient(cfg), nil
	case APITypeChatGPT:
		return NewChatGPTClient(cfg), nil
	case APITypeGemini:
		// Placeholder for future Gemini implementation
		return nil, fmt.Errorf("Gemini API support coming soon")
	default:
		return nil, fmt.Errorf("unsupported API type: %s", cfg.APIType)
	}
}
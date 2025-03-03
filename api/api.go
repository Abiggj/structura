package api

import (
	"github.com/Abiggj/structura/filehandler"
)

// DocumentationClient defines the interface for documentation API clients
type DocumentationClient interface {
	GenerateDocumentation(file filehandler.FileInfo) (string, error)
}
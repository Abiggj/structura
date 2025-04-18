package filehandler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectType represents the type of project
type ProjectType string

const (
	ProjectTypeGeneric ProjectType = "generic"
	ProjectTypeReact   ProjectType = "react"
	ProjectTypeNode    ProjectType = "node"
	ProjectTypePython  ProjectType = "python"
	ProjectTypeDjango  ProjectType = "django"
	ProjectTypeGo      ProjectType = "go"
	ProjectTypeJava    ProjectType = "java"
	ProjectTypeRuby    ProjectType = "ruby"
	ProjectTypeRails   ProjectType = "rails"
	ProjectTypeFlutter ProjectType = "flutter"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path    string
	Content string
	Size    int64
	IsDir   bool
}

// FileHandler handles file operations
type FileHandler struct {
	IgnoreDirs  []string
	IgnoreFiles []string
	ProjectType ProjectType
}

// NewFileHandler creates a new file handler
func NewFileHandler() *FileHandler {
	return &FileHandler{
		IgnoreDirs: []string{
			".git", "node_modules", "vendor", "dist", "build",
			".idea", ".vscode", ".github", ".cache", ".svn",
			".hg", ".bzr", "CVS", "__pycache__", ".sass-cache",
			".next", ".nuxt", ".output", "out", ".parcel-cache",
		},
		IgnoreFiles: []string{
			".DS_Store", "*.lock", "*.log", "*.wasm", "*.min.js",
			"*.min.css", "*.map", "*.ico", "*.svg", "*.png", "*.jpg",
			"*.jpeg", "*.gif", "*.webp", "*.ttf", "*.woff", "*.woff2",
			".env", "*.env", ".env.*", "*.yml", "*.yaml", "*.toml", "*.ini",
			"*.config", "*.conf", "Dockerfile", "docker-compose.yml",
			".gitignore", ".gitattributes", ".gitmodules", ".gitkeep",
			".npmrc", ".npmignore", ".eslintignore", ".prettierignore",
			".dockerignore", ".editorconfig", "thumbs.db", ".htaccess", 
			"*.swp", "*.swo", "*.bak", "*.tmp", "*.temp", "*.o", "*.obj",
			"*.suo", "*.user", "*.userosscache", "*.dbmdl", 
			"*.sh", "*README*", "*readme*",
		},
		ProjectType: ProjectTypeGeneric,
	}
}

// SetProjectType sets the project type and updates ignore rules
func (fh *FileHandler) SetProjectType(projectType ProjectType) {
	fh.ProjectType = projectType
	
	// Reset to default ignore rules first
	fh.IgnoreDirs = []string{
		".git", "node_modules", "vendor", "dist", "build",
		".idea", ".vscode", ".github", ".cache", ".svn",
		".hg", ".bzr", "CVS", "__pycache__", ".sass-cache",
		".next", ".nuxt", ".output", "out", ".parcel-cache",
	}
	fh.IgnoreFiles = []string{
		".DS_Store", "*.lock", "*.log", "*.wasm", "*.min.js",
		"*.min.css", "*.map", "*.ico", "*.svg", "*.png", "*.jpg",
		"*.jpeg", "*.gif", "*.webp", "*.ttf", "*.woff", "*.woff2",
		".env", "*.env", ".env.*", "*.yml", "*.yaml", "*.toml", "*.ini",
		"*.config", "*.conf", "Dockerfile", "docker-compose.yml",
		".gitignore", ".gitattributes", ".gitmodules", ".gitkeep",
		".npmrc", ".npmignore", ".eslintignore", ".prettierignore",
		".dockerignore", ".editorconfig", "thumbs.db", ".htaccess", 
		"*.swp", "*.swo", "*.bak", "*.tmp", "*.temp", "*.o", "*.obj",
		"*.suo", "*.user", "*.userosscache", "*.dbmdl",
		"*.sh", "*README*", "*readme*",
	}
	
	// Add project-specific ignore rules
	switch projectType {
	case ProjectTypeReact, ProjectTypeNode:
		fh.IgnoreDirs = append(fh.IgnoreDirs, "coverage", ".next")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "package.json", "package-lock.json", "*.config.js", "*.test.*")
	case ProjectTypePython, ProjectTypeDjango:
		fh.IgnoreDirs = append(fh.IgnoreDirs, "__pycache__", ".venv", "venv", "env", ".pytest_cache")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "*.pyc", "requirements.txt", ".env")
	case ProjectTypeGo:
		fh.IgnoreDirs = append(fh.IgnoreDirs, "bin")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "go.sum")
	case ProjectTypeJava:
		fh.IgnoreDirs = append(fh.IgnoreDirs, "target", "out", "bin")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "*.class", "*.jar", "pom.xml", "build.gradle")
	case ProjectTypeRuby, ProjectTypeRails:
		fh.IgnoreDirs = append(fh.IgnoreDirs, "tmp", "log")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "Gemfile.lock", "*.gem")
	case ProjectTypeFlutter:
		fh.IgnoreDirs = append(fh.IgnoreDirs, ".dart_tool", "build")
		fh.IgnoreFiles = append(fh.IgnoreFiles, "pubspec.lock", "*.g.dart")
	}
}

// ShouldIgnore checks if a file or directory should be ignored
func (fh *FileHandler) ShouldIgnore(path string) bool {
	basename := filepath.Base(path)

	// Check if it's in the ignore dirs list
	for _, dir := range fh.IgnoreDirs {
		if basename == dir {
			return true
		}
	}

	// Check file patterns
	for _, pattern := range fh.IgnoreFiles {
		if matched, _ := filepath.Match(pattern, basename); matched {
			return true
		}
	}

	return false
}

// TraverseDirectory walks through the directory and collects file information
func (fh *FileHandler) TraverseDirectory(rootDir string) ([]FileInfo, error) {
	var files []FileInfo

	// Clean and normalize the path for cross-platform compatibility
	rootDir = filepath.Clean(rootDir)
	
	// Check if directory exists before walking
	info, err := os.Stat(rootDir)
	if err != nil {
		return nil, fmt.Errorf("error accessing directory: %w", err)
	}
	
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", rootDir)
	}

	err = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored files and directories
		if fh.ShouldIgnore(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create FileInfo struct
		fileInfo := FileInfo{
			Path:  path,
			Size:  info.Size(),
			IsDir: info.IsDir(),
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only read reasonable sized files
		if info.Size() < 5*1024*1024 { // Less than 5MB
			content, err := os.ReadFile(path)
			if err == nil {
				fileInfo.Content = string(content)
			}
		}

		files = append(files, fileInfo)
		return nil
	})

	return files, err
}

// GetFileExtension returns the file extension without the dot
func GetFileExtension(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimPrefix(ext, ".")
}
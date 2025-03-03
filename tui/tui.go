package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Abiggj/structura/api"
	"github.com/Abiggj/structura/config"
	"github.com/Abiggj/structura/filehandler"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styling
var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D")).
			Padding(1, 0)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555"))

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#61AFEF"))

	progressBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)
			
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Bold(true).
			Padding(0, 1)
)

// Model represents the state of the TUI
type Model struct {
	config        *config.Config
	fileHandler   *filehandler.FileHandler
	apiClient     api.DocumentationClient
	state         State
	inputDir      string
	outputDir     string
	apiKey        string
	
	// API Selection
	apiTypes        []api.APIType
	selectedAPIType int
	apiModels       []string
	selectedModel   int
	
	// Project type selection
	projectType   filehandler.ProjectType
	projectTypes  []filehandler.ProjectType
	selectedType  int
	
	// Directory Selection
	dirEntries     []os.DirEntry
	selectedDir    int
	dirHistory     []string // For navigation history
	
	// Processing
	files         []filehandler.FileInfo
	processedFiles int
	currentFile   string
	errors        []string
	spinner       spinner.Model
	progress      progress.Model
	width         int
	height        int
}

// State represents the current state of the application
type State int

const (
	StateInit State = iota
	StateSelectAPIType
	StateSelectAPIModel
	StateEnterAPIKey
	StateSelectProjectType
	StateSelectInputDir
	StateEnterInputDir  // Fallback if selecting fails
	StateEnterOutputDir
	StateProcessing
	StateDone
)

// NewModel creates a new TUI model
func NewModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	p := progress.New(progress.WithDefaultGradient())

	// Define available project types
	projectTypes := []filehandler.ProjectType{
		filehandler.ProjectTypeGeneric,
		filehandler.ProjectTypeReact,
		filehandler.ProjectTypeNode,
		filehandler.ProjectTypePython,
		filehandler.ProjectTypeDjango,
		filehandler.ProjectTypeGo,
		filehandler.ProjectTypeJava,
		filehandler.ProjectTypeRuby,
		filehandler.ProjectTypeRails,
		filehandler.ProjectTypeFlutter,
	}
	
	// Set up API types
	apiTypes := api.APITypes()
	
	// Get current working directory for initial directory navigation
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "/"
	}
	
	// Create initial config
	cfg := config.NewConfig()
	
	return Model{
		config:          cfg,
		fileHandler:     filehandler.NewFileHandler(),
		state:           StateInit,
		spinner:         s,
		progress:        p,
		projectTypes:    projectTypes,
		projectType:     filehandler.ProjectTypeGeneric,
		selectedType:    0,
		apiTypes:        apiTypes,
		selectedAPIType: 0,
		apiModels:       api.APIModelMap[apiTypes[0]], // Default to first API type models
		selectedModel:   0,
		inputDir:        cwd,
		dirHistory:      []string{cwd},
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, tea.EnterAltScreen)
}

// Update updates the model based on messages
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}

		// Handle different states
		switch m.state {
		case StateInit:
			m.state = StateSelectAPIType
			return m, nil
			
		case StateSelectAPIType:
			switch msg.String() {
			case "up", "k":
				if m.selectedAPIType > 0 {
					m.selectedAPIType--
					// Update available models when API type changes
					m.apiModels = api.APIModelMap[m.apiTypes[m.selectedAPIType]]
					m.selectedModel = 0
				}
				return m, nil
			case "down", "j":
				if m.selectedAPIType < len(m.apiTypes)-1 {
					m.selectedAPIType++
					// Update available models when API type changes
					m.apiModels = api.APIModelMap[m.apiTypes[m.selectedAPIType]]
					m.selectedModel = 0
				}
				return m, nil
			case "enter":
				m.config.APIType = m.apiTypes[m.selectedAPIType]
				m.state = StateSelectAPIModel
				return m, nil
			}
			return m, nil
			
		case StateSelectAPIModel:
			switch msg.String() {
			case "up", "k":
				if m.selectedModel > 0 {
					m.selectedModel--
				}
				return m, nil
			case "down", "j":
				if m.selectedModel < len(m.apiModels)-1 {
					m.selectedModel++
				}
				return m, nil
			case "enter":
				m.config.APIModel = m.apiModels[m.selectedModel]
				m.state = StateEnterAPIKey
				return m, nil
			}
			return m, nil

		case StateEnterAPIKey:
			if msg.Type == tea.KeyEnter {
				// Set the appropriate API key based on the selected API type
				switch m.config.APIType {
				case api.APITypeChatGPT:
					m.config.OpenAIAPIKey = m.apiKey
				case api.APITypeGemini:
					m.config.GeminiAPIKey = m.apiKey
				default:
					m.config.DeepseekAPIKey = m.apiKey
				}
				
				// Create the appropriate API client
				var err error
				m.apiClient, err = api.CreateDocumentationClient(m.config)
				if err != nil {
					m.errors = append(m.errors, fmt.Sprintf("Error creating API client: %s", err))
					return m, nil
				}
				
				m.state = StateSelectProjectType
				return m, nil
			}
			
			// Handle backspace
			if msg.Type == tea.KeyBackspace && len(m.apiKey) > 0 {
				m.apiKey = m.apiKey[:len(m.apiKey)-1]
				return m, nil
			}
			
			if msg.Type == tea.KeyRunes {
				m.apiKey += string(msg.Runes)
			}
			return m, nil
			
		case StateSelectProjectType:
			switch msg.String() {
			case "up", "k":
				if m.selectedType > 0 {
					m.selectedType--
				}
				return m, nil
			case "down", "j":
				if m.selectedType < len(m.projectTypes)-1 {
					m.selectedType++
				}
				return m, nil
			case "enter":
				m.projectType = m.projectTypes[m.selectedType]
				m.fileHandler.SetProjectType(m.projectType)
				
				// Store the fileHandler in the config for the API client to access
				m.config.FileHandler = m.fileHandler
				
				// Load the directory entries for input directory selection
				if err := m.loadDirectoryEntries(m.inputDir); err != nil {
					m.errors = append(m.errors, fmt.Sprintf("Error loading directory: %s", err))
					m.state = StateEnterInputDir // Fallback to manual entry
				} else {
					m.state = StateSelectInputDir
				}
				return m, nil
			}
			return m, nil
			
		case StateSelectInputDir:
			switch msg.String() {
			case "up", "k":
				if m.selectedDir > 0 {
					m.selectedDir--
				}
				return m, nil
			case "down", "j":
				if m.selectedDir < len(m.dirEntries)-1 {
					m.selectedDir++
				}
				return m, nil
			case "enter":
				// If a directory is selected, navigate into it
				if m.selectedDir < len(m.dirEntries) && m.dirEntries[m.selectedDir].IsDir() {
					entry := m.dirEntries[m.selectedDir]
					
					if entry.Name() == ".." {
						// Go up one directory
						if len(m.dirHistory) > 1 {
							m.dirHistory = m.dirHistory[:len(m.dirHistory)-1]
							m.inputDir = m.dirHistory[len(m.dirHistory)-1]
						}
					} else {
						// Go into the selected directory
						newPath := filepath.Join(m.inputDir, entry.Name())
						m.inputDir = newPath
						m.dirHistory = append(m.dirHistory, newPath)
					}
					
					// Reload directory entries
					if err := m.loadDirectoryEntries(m.inputDir); err != nil {
						m.errors = append(m.errors, fmt.Sprintf("Error loading directory: %s", err))
					}
					return m, nil
				} else {
					// If not a directory, confirm this directory as the input dir
					m.state = StateEnterOutputDir
				}
				return m, nil
			case "escape":
				// Switch to manual entry mode
				m.state = StateEnterInputDir
				return m, nil
			}
			return m, nil

		case StateEnterInputDir:
			if msg.Type == tea.KeyEnter {
				// Clean and normalize the path
				cleanPath := filepath.Clean(m.inputDir)
				m.inputDir = cleanPath
				
				// Check if directory exists
				info, err := os.Stat(m.inputDir)
				if err != nil {
					m.errors = append(m.errors, fmt.Sprintf("Error accessing directory: %s", err))
					return m, nil
				}
				
				if !info.IsDir() {
					m.errors = append(m.errors, fmt.Sprintf("Path is not a directory: %s", m.inputDir))
					return m, nil
				}
				
				m.state = StateEnterOutputDir
				return m, nil
			}
			
			// Handle backspace
			if msg.Type == tea.KeyBackspace && len(m.inputDir) > 0 {
				m.inputDir = m.inputDir[:len(m.inputDir)-1]
				return m, nil
			}
			
			if msg.Type == tea.KeyRunes {
				m.inputDir += string(msg.Runes)
			}
			return m, nil

		case StateEnterOutputDir:
			if msg.Type == tea.KeyEnter {
				// Clean the path
				cleanPath := filepath.Clean(m.outputDir)
				m.outputDir = cleanPath
				
				// Create output directory if it doesn't exist
				if err := os.MkdirAll(m.outputDir, 0755); err != nil {
					m.errors = append(m.errors, fmt.Sprintf("Failed to create output directory: %s", err))
					return m, nil
				}
				
				// Start processing
				m.state = StateProcessing
				return m, tea.Batch(
					m.processFiles,
					m.spinner.Tick,
				)
			}
			
			// Handle backspace
			if msg.Type == tea.KeyBackspace && len(m.outputDir) > 0 {
				m.outputDir = m.outputDir[:len(m.outputDir)-1]
				return m, nil
			}
			
			if msg.Type == tea.KeyRunes {
				m.outputDir += string(msg.Runes)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 10
		return m, nil
		
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
		
	case progressMsg:
		// Update the progress
		cmd := m.progress.SetPercent(float64(m.processedFiles) / float64(len(m.files)))
		return m, cmd
		
	case fileProcessedMsg:
		m.processedFiles++
		m.currentFile = string(msg)
		
		progress := float64(m.processedFiles) / float64(len(m.files))
		if m.processedFiles >= len(m.files) {
			// Generate and save project structure and setup documentation
			m.generateStructureDocumentation()
			
			m.state = StateDone
			return m, m.progress.SetPercent(progress)
		}
		
		// Continue processing the next file
		return m, tea.Batch(
			m.progress.SetPercent(progress),
			continueProcessing(filesLoadedMsg{files: m.files}, m),
		)
		
	case fileErrorMsg:
		m.errors = append(m.errors, string(msg))
		m.processedFiles++
		
		progress := float64(m.processedFiles) / float64(len(m.files))
		if m.processedFiles >= len(m.files) {
			// Generate and save project structure and setup documentation
			m.generateStructureDocumentation()
			
			m.state = StateDone
			return m, m.progress.SetPercent(progress)
		}
		
		// Continue processing the next file
		return m, tea.Batch(
			m.progress.SetPercent(progress),
			continueProcessing(filesLoadedMsg{files: m.files}, m),
		)
		
	case filesLoadedMsg:
		m.files = msg.files
		// Start processing files
		return m, continueProcessing(msg, m)
	}

	return m, nil
}

// View renders the current state of the application
func (m Model) View() string {
	title := "Structura - Documentation Generator"
	
	switch m.state {
	case StateInit:
		return titleStyle.Render(title) + "\n\n" +
			"Press any key to start"
			
	case StateSelectAPIType:
		var options string
		for i, apiType := range m.apiTypes {
			option := string(apiType)
			if i == m.selectedAPIType {
				options += selectedStyle.Render("› " + option) + "\n"
			} else {
				options += "  " + option + "\n"
			}
		}
		
		return titleStyle.Render(title) + "\n\n" +
			"Select API type (use arrow keys and enter):\n\n" +
			options + "\n" +
			renderErrors(m.errors)
	
	case StateSelectAPIModel:
		var options string
		for i, model := range m.apiModels {
			if i == m.selectedModel {
				options += selectedStyle.Render("› " + model) + "\n"
			} else {
				options += "  " + model + "\n"
			}
		}
		
		apiTypeStr := string(m.config.APIType)
		return titleStyle.Render(title) + "\n\n" +
			fmt.Sprintf("Selected API: %s\n\n", apiTypeStr) +
			"Select model (use arrow keys and enter):\n\n" +
			options + "\n" +
			renderErrors(m.errors)
			
	case StateEnterAPIKey:
		apiTypeStr := string(m.config.APIType)
		return titleStyle.Render(title) + "\n\n" +
			fmt.Sprintf("Selected API: %s\n", apiTypeStr) + 
			fmt.Sprintf("Selected model: %s\n\n", m.config.APIModel) +
			fmt.Sprintf("Enter your %s API Key: %s\n\n", apiTypeStr, strings.Repeat("*", len(m.apiKey))) +
			renderErrors(m.errors)
			
	case StateSelectProjectType:
		var options string
		for i, projectType := range m.projectTypes {
			option := string(projectType)
			if i == m.selectedType {
				options += selectedStyle.Render("› " + option) + "\n"
			} else {
				options += "  " + option + "\n"
			}
		}
		
		apiTypeStr := string(m.config.APIType)
		return titleStyle.Render(title) + "\n\n" +
			fmt.Sprintf("Using: %s / %s\n\n", apiTypeStr, m.config.APIModel) +
			"Select project type (use arrow keys and enter):\n\n" +
			options + "\n" +
			renderErrors(m.errors)
			
	case StateSelectInputDir:
		var dirList string
		maxEntries := 15 // Maximum number of entries to show
		startIndex := 0
		
		// If there are many entries, center the selected one
		if len(m.dirEntries) > maxEntries && m.selectedDir > maxEntries/2 {
			startIndex = m.selectedDir - maxEntries/2
			if startIndex + maxEntries > len(m.dirEntries) {
				startIndex = len(m.dirEntries) - maxEntries
			}
			if startIndex < 0 {
				startIndex = 0
			}
		}
		
		endIndex := startIndex + maxEntries
		if endIndex > len(m.dirEntries) {
			endIndex = len(m.dirEntries)
		}
		
		// Add current path information
		dirList += infoStyle.Render("Current directory: " + m.inputDir) + "\n\n"
		
		// Add directory entries
		for i := startIndex; i < endIndex; i++ {
			entry := m.dirEntries[i]
			name := entry.Name()
			
			// Add indicator for directories
			if entry.IsDir() {
				name += "/"
			}
			
			if i == m.selectedDir {
				dirList += selectedStyle.Render("› " + name) + "\n"
			} else {
				dirList += "  " + name + "\n"
			}
		}
		
		// Show indication if more entries are available
		if len(m.dirEntries) > endIndex {
			dirList += "  ... " + fmt.Sprintf("(%d more)", len(m.dirEntries) - endIndex) + "\n"
		}
		
		// Add instructions
		dirList += "\n" + infoStyle.Render("Navigate with arrow keys, press Enter to select or enter a directory, Esc for manual input")
		
		return titleStyle.Render(title) + "\n\n" +
			"Select input directory:\n\n" +
			dirList + "\n\n" +
			renderErrors(m.errors)
			
	case StateEnterInputDir:
		return titleStyle.Render(title) + "\n\n" +
			"Enter the input directory path: " + m.inputDir + "\n\n" +
			renderErrors(m.errors)
			
	case StateEnterOutputDir:
		return titleStyle.Render(title) + "\n\n" +
			"Enter the output directory path: " + m.outputDir + "\n\n" +
			renderErrors(m.errors)
			
	case StateProcessing:
		progress := fmt.Sprintf("Processing %d/%d files", m.processedFiles, len(m.files))
		
		apiTypeStr := string(m.config.APIType)
		return titleStyle.Render(title) + "\n\n" +
			infoStyle.Render(fmt.Sprintf("API: %s / %s", apiTypeStr, m.config.APIModel)) + "\n" +
			infoStyle.Render("Processing files from: " + m.inputDir) + "\n" +
			infoStyle.Render("Saving documentation to: " + m.outputDir) + "\n" +
			infoStyle.Render("Project type: " + string(m.projectType)) + "\n\n" +
			m.spinner.View() + " " + progress + "\n" +
			progressBarStyle.Render(m.progress.View()) + "\n\n" +
			fileStyle.Render("Current file: " + m.currentFile) + "\n\n" +
			renderErrors(m.errors)
			
	case StateDone:
		apiTypeStr := string(m.config.APIType)
		return titleStyle.Render(title) + "\n\n" +
			infoStyle.Render(fmt.Sprintf("✓ Done! Processed %d files using %s", m.processedFiles, apiTypeStr)) + "\n" +
			infoStyle.Render("Documentation saved to: " + m.outputDir) + "\n" +
			infoStyle.Render("Project structure documentation: " + filepath.Join(m.outputDir, "PROJECT_STRUCTURE.md")) + "\n" +
			infoStyle.Render("Project setup documentation: " + filepath.Join(m.outputDir, "PROJECT_SETUP.md")) + "\n\n" +
			renderErrors(m.errors) + "\n\n" +
			"Press q to quit"
	}
	
	return ""
}

// processFiles processes all files in the input directory
func (m Model) processFiles() tea.Msg {
	// Traverse the directory
	files, err := m.fileHandler.TraverseDirectory(m.inputDir)
	if err != nil {
		return fileErrorMsg(fmt.Sprintf("Failed to traverse directory: %s", err))
	}
	
	// Return the files loaded message first
	return filesLoadedMsg{files: files}
}

// continueProcessing continues processing after files are loaded
func continueProcessing(msg tea.Msg, m Model) tea.Cmd {
	filesMsg, ok := msg.(filesLoadedMsg)
	if !ok {
		return nil
	}
	
	files := filesMsg.files
	
	// Process only one file at a time, so we can update the UI
	return func() tea.Msg {
		// Find the next file to process
		for i, file := range files {
			if i < m.processedFiles {
				continue // Skip already processed files
			}
			
			if file.IsDir {
				m.processedFiles++
				continue
			}
			
			// Update current file
			currentFile := file.Path
			
			// Create relative path for output
			relPath, err := filepath.Rel(m.inputDir, file.Path)
			if err != nil {
				return fileErrorMsg(fmt.Sprintf("Failed to get relative path for %s: %s", file.Path, err))
			}
			
			// Create output directory with the same structure as input
			outputPath := filepath.Join(m.outputDir, filepath.Dir(relPath))
			if err := os.MkdirAll(outputPath, 0755); err != nil {
				return fileErrorMsg(fmt.Sprintf("Failed to create directory %s: %s", outputPath, err))
			}
			
			// Output file path
			outputFile := filepath.Join(outputPath, filepath.Base(file.Path)+".md")
			
			// Check if the file has already been documented
			if _, err := os.Stat(outputFile); err == nil {
				// File already exists in the output directory, skip processing
				m.processedFiles++
				return fileProcessedMsg(currentFile + " (already documented, skipped)")
			}
			
			// Generate documentation
			doc, err := m.apiClient.GenerateDocumentation(file)
			if err != nil {
				return fileErrorMsg(fmt.Sprintf("Failed to generate documentation for %s: %s", file.Path, err))
			}
			
			// Write documentation to file
			if err := os.WriteFile(outputFile, []byte(doc), 0644); err != nil {
				return fileErrorMsg(fmt.Sprintf("Failed to write documentation to %s: %s", outputFile, err))
			}
			
			// Return a file processed message
			return fileProcessedMsg(currentFile)
		}
		
		// If we've processed all files, return nil
		if m.processedFiles >= len(files) {
			return nil
		}
		
		return nil
	}
}

// Message types
type progressMsg float64
type fileProcessedMsg string
type fileErrorMsg string
type filesLoadedMsg struct {
	files []filehandler.FileInfo
}

// renderErrors renders the error messages
func renderErrors(errors []string) string {
	if len(errors) == 0 {
		return ""
	}
	
	result := errorStyle.Render("Errors:") + "\n"
	for _, err := range errors {
		result += errorStyle.Render("- " + err) + "\n"
	}
	
	return result
}

// loadDirectoryEntries loads directory entries for the given path
func (m *Model) loadDirectoryEntries(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	
	// Filter to show only directories first, then files
	var dirs []os.DirEntry
	var files []os.DirEntry
	
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry)
		} else {
			files = append(files, entry)
		}
	}
	
	// Add special "parent directory" entry if not at root
	if path != "/" {
		parentEntry := &dirEntry{name: "..", isDir: true}
		dirs = append([]os.DirEntry{parentEntry}, dirs...)
	}
	
	// Combine directories and files
	m.dirEntries = append(dirs, files...)
	m.selectedDir = 0
	
	return nil
}

// Custom DirEntry implementation for special entries like ".."
type dirEntry struct {
	name  string
	isDir bool
}

func (d *dirEntry) Name() string               { return d.name }
func (d *dirEntry) IsDir() bool                { return d.isDir }
func (d *dirEntry) Type() os.FileMode          { return os.ModeDir }
func (d *dirEntry) Info() (os.FileInfo, error) { return nil, nil }

// generateStructureDocumentation creates documentation for the project structure and setup
func (m Model) generateStructureDocumentation() {
	// 1. Generate project structure documentation
	structureDoc := "# Project Structure\n\n"
	structureDoc += "This document provides an overview of the project's directory structure and organization.\n\n"
	
	// Create a map to track directories and their files
	dirMap := make(map[string][]string)
	
	// Organize files by directory
	for _, file := range m.files {
		if file.IsDir {
			continue
		}
		
		// Get directory path
		dir := filepath.Dir(file.Path)
		relDir, err := filepath.Rel(m.inputDir, dir)
		if err != nil {
			continue
		}
		
		if relDir == "." {
			relDir = "Root"
		}
		
		// Add file to directory map
		dirMap[relDir] = append(dirMap[relDir], filepath.Base(file.Path))
	}
	
	// Add directories and files to documentation
	structureDoc += "## Directory Structure\n\n"
	for dir, files := range dirMap {
		structureDoc += fmt.Sprintf("### %s\n\n", dir)
		
		// Add files in the directory
		if len(files) > 0 {
			structureDoc += "Files:\n"
			for _, file := range files {
				structureDoc += fmt.Sprintf("- `%s`\n", file)
			}
			structureDoc += "\n"
		}
	}
	
	// Write project structure documentation
	structureFilePath := filepath.Join(m.outputDir, "PROJECT_STRUCTURE.md")
	os.WriteFile(structureFilePath, []byte(structureDoc), 0644)
	
	// 2. Generate setup documentation
	setupDoc := "# Project Setup\n\n"
	setupDoc += "This document provides information on how to set up and run this project.\n\n"
	
	// Look for common setup files
	setupFiles := []string{
		"package.json", "go.mod", "requirements.txt", "Gemfile", 
		"pom.xml", "build.gradle", "Makefile", "pubspec.yaml",
		"composer.json", "setup.py", "CMakeLists.txt",
	}
	
	// Section for dependencies
	setupDoc += "## Dependencies\n\n"
	
	// Find and document setup files
	foundSetupFiles := false
	for _, file := range m.files {
		fileName := filepath.Base(file.Path)
		for _, setupFileName := range setupFiles {
			if fileName == setupFileName {
				foundSetupFiles = true
				setupDoc += fmt.Sprintf("### %s\n\n", fileName)
				setupDoc += "```\n"
				// Limit content size to avoid overly large documents
				content := file.Content
				if len(content) > 2000 {
					content = content[:2000] + "\n... (content truncated)"
				}
				setupDoc += content + "\n"
				setupDoc += "```\n\n"
			}
		}
	}
	
	if !foundSetupFiles {
		setupDoc += "No standard setup files found in the project.\n\n"
	}
	
	// Add installation and running instructions
	setupDoc += "## Installation\n\n"
	setupDoc += "Please follow these steps to install and set up the project:\n\n"
	setupDoc += "1. Clone the repository\n"
	setupDoc += "2. Install dependencies\n"
	
	// Add project type specific instructions
	switch m.projectType {
	case filehandler.ProjectTypeNode, filehandler.ProjectTypeReact:
		setupDoc += "   ```\n   npm install\n   ```\n"
	case filehandler.ProjectTypeGo:
		setupDoc += "   ```\n   go mod download\n   ```\n"
	case filehandler.ProjectTypePython, filehandler.ProjectTypeDjango:
		setupDoc += "   ```\n   pip install -r requirements.txt\n   ```\n"
	case filehandler.ProjectTypeRuby, filehandler.ProjectTypeRails:
		setupDoc += "   ```\n   bundle install\n   ```\n"
	case filehandler.ProjectTypeJava:
		setupDoc += "   ```\n   mvn install\n   ```\n"
	case filehandler.ProjectTypeFlutter:
		setupDoc += "   ```\n   flutter pub get\n   ```\n"
	}
	
	setupDoc += "\n## Running the Project\n\n"
	setupDoc += "Specific instructions for running this project will depend on its configuration.\n"
	
	// Write setup documentation
	setupFilePath := filepath.Join(m.outputDir, "PROJECT_SETUP.md")
	os.WriteFile(setupFilePath, []byte(setupDoc), 0644)
}
package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aniket_jhariya/structura/api"
	"github.com/aniket_jhariya/structura/config"
	"github.com/aniket_jhariya/structura/filehandler"
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
	apiClient     *api.DeepseekClient
	state         State
	inputDir      string
	outputDir     string
	apiKey        string
	projectType   filehandler.ProjectType
	projectTypes  []filehandler.ProjectType
	selectedType  int
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
	StateEnterAPIKey
	StateSelectProjectType
	StateEnterInputDir
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

	return Model{
		config:       config.NewConfig(),
		fileHandler:  filehandler.NewFileHandler(),
		state:        StateInit,
		spinner:      s,
		progress:     p,
		projectTypes: projectTypes,
		projectType:  filehandler.ProjectTypeGeneric,
		selectedType: 0,
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
			m.state = StateEnterAPIKey
			return m, nil

		case StateEnterAPIKey:
			if msg.Type == tea.KeyEnter {
				m.config.DeepseekAPIKey = m.apiKey
				m.apiClient = api.NewDeepseekClient(m.config)
				m.state = StateSelectProjectType
				return m, nil
			}
			m.apiKey += string(msg.Runes)
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
			m.state = StateDone
		}
		
		return m, m.progress.SetPercent(progress)
		
	case fileErrorMsg:
		m.errors = append(m.errors, string(msg))
		m.processedFiles++
		
		progress := float64(m.processedFiles) / float64(len(m.files))
		if m.processedFiles >= len(m.files) {
			m.state = StateDone
		}
		
		return m, m.progress.SetPercent(progress)
		
	case filesLoadedMsg:
		m.files = msg.files
		return m, nil
	}

	return m, nil
}

// View renders the current state of the application
func (m Model) View() string {
	switch m.state {
	case StateInit:
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			"Press any key to start"
			
	case StateEnterAPIKey:
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			"Enter your DeepSeek API Key: " + strings.Repeat("*", len(m.apiKey)) + "\n\n" +
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
		
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			"Select project type (use arrow keys and enter):\n\n" +
			options + "\n" +
			renderErrors(m.errors)
			
	case StateEnterInputDir:
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			"Enter the input directory path: " + m.inputDir + "\n\n" +
			renderErrors(m.errors)
			
	case StateEnterOutputDir:
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			"Enter the output directory path: " + m.outputDir + "\n\n" +
			renderErrors(m.errors)
			
	case StateProcessing:
		progress := fmt.Sprintf("Processing %d/%d files", m.processedFiles, len(m.files))
		
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			infoStyle.Render("Processing files from: " + m.inputDir) + "\n" +
			infoStyle.Render("Saving documentation to: " + m.outputDir) + "\n" +
			infoStyle.Render("Project type: " + string(m.projectType)) + "\n\n" +
			m.spinner.View() + " " + progress + "\n" +
			progressBarStyle.Render(m.progress.View()) + "\n\n" +
			fileStyle.Render("Current file: " + m.currentFile) + "\n\n" +
			renderErrors(m.errors)
			
	case StateDone:
		return titleStyle.Render("Structura - DeepSeek Documentation Generator") + "\n\n" +
			infoStyle.Render(fmt.Sprintf("✓ Done! Processed %d files", m.processedFiles)) + "\n" +
			infoStyle.Render("Documentation saved to: " + m.outputDir) + "\n\n" +
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
	
	// Update the model with the files
	tea.NewProgram(m).Send(filesLoadedMsg{files: files})
	
	// Process each file
	for _, file := range files {
		if file.IsDir {
			continue
		}
		
		// Generate documentation
		doc, err := m.apiClient.GenerateDocumentation(file)
		if err != nil {
			tea.NewProgram(m).Send(fileErrorMsg(fmt.Sprintf("Failed to generate documentation for %s: %s", file.Path, err)))
			continue
		}
		
		// Create relative path for output
		relPath, err := filepath.Rel(m.inputDir, file.Path)
		if err != nil {
			tea.NewProgram(m).Send(fileErrorMsg(fmt.Sprintf("Failed to get relative path for %s: %s", file.Path, err)))
			continue
		}
		
		// Create output directory
		outputPath := filepath.Join(m.outputDir, filepath.Dir(relPath))
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			tea.NewProgram(m).Send(fileErrorMsg(fmt.Sprintf("Failed to create directory %s: %s", outputPath, err)))
			continue
		}
		
		// Write documentation to file
		outputFile := filepath.Join(outputPath, filepath.Base(file.Path)+".md")
		if err := os.WriteFile(outputFile, []byte(doc), 0644); err != nil {
			tea.NewProgram(m).Send(fileErrorMsg(fmt.Sprintf("Failed to write documentation to %s: %s", outputFile, err)))
			continue
		}
		
		// Update the progress
		tea.NewProgram(m).Send(fileProcessedMsg(file.Path))
		
		// Sleep for a short time to avoid API rate limiting
		time.Sleep(100 * time.Millisecond)
	}
	
	return nil
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
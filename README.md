# Structura

Structura is a Terminal User Interface (TUI) based Go application that automatically generates Markdown documentation for any project directory by recursively analyzing each file using the DeepSeek AI API.

## Features

- Intuitive terminal interface built with BubbleTea
- Recursively traverses project directories
- Generates comprehensive Markdown documentation for each file
- Uses DeepSeek AI API to create intelligent code documentation
- Progress tracking with visual indicators
- Customizable file/directory ignore patterns

## Installation

Ensure you have Go installed (version 1.16 or higher), then run:

```bash
go install github.com/Abiggj/structura@latest
```

Or clone and build from source:

```bash
git clone https://github.com/Abiggj/structura.git
cd structura
go build -o structura
```

## Usage

1. Run the application:
   ```bash
   ./structura
   ```

2. Enter your DeepSeek API key when prompted.

3. Specify the input directory that contains the project you want to document.

4. Specify the output directory where the documentation will be saved.

5. Wait for the processing to complete. The application will show a progress bar and status updates.

6. Press 'q' to quit once the process is complete.

## Configuration

You can customize the ignore patterns for files and directories by modifying the `filehandler/filehandler.go` file:

```go
func NewFileHandler() *FileHandler {
    return &FileHandler{
        IgnoreDirs: []string{
            ".git", "node_modules", "vendor", "dist", "build",
            ".idea", ".vscode", ".github", ".cache",
        },
        IgnoreFiles: []string{
            ".DS_Store", "*.lock", "*.log", "*.wasm", "*.min.js",
            "*.min.css", "*.map", "*.ico", "*.svg", "*.png", "*.jpg",
            "*.jpeg", "*.gif", "*.webp", "*.ttf", "*.woff", "*.woff2",
        },
    }
}
```

## Dependencies

- [BubbleTea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal applications
- [Bubbles](https://github.com/charmbracelet/bubbles) - UI components for BubbleTea
- [Resty](https://github.com/go-resty/resty) - HTTP client for Go

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
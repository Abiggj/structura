package main

import (
	"fmt"
	"os"

	"github.com/aniket_jhariya/structura/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Create a new model
	m := tui.NewModel()

	// Initialize the program
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Start the program
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
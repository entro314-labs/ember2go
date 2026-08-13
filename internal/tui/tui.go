package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbletea"
)

func RunTUI(isoPath string) error {
	model := NewModel(isoPath)

	program := tea.NewProgram(model, tea.WithAltScreen())

	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

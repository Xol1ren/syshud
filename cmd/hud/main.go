package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"syshud/internal/sysinfo"
	"syshud/internal/ui"
)

func main() {
	collector, err := sysinfo.NewCollector()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing sysinfo collector: %v\n", err)
		os.Exit(1)
	}
	defer collector.Close()

	model := ui.NewModel(collector)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running HUD: %v\n", err)
		os.Exit(1)
	}
}


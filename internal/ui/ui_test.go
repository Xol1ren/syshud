package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"syshud/internal/sysinfo"
)

func TestRenderBar(t *testing.T) {
	tests := []struct {
		pct      float64
		barWidth int
	}{
		{pct: -10, barWidth: 16},
		{pct: 0, barWidth: 16},
		{pct: 25, barWidth: 16},
		{pct: 50, barWidth: 16},
		{pct: 85, barWidth: 16},
		{pct: 100, barWidth: 16},
		{pct: 120, barWidth: 16},
	}

	for _, tc := range tests {
		bar := RenderBar(tc.pct, tc.barWidth)
		visLen := lipgloss.Width(bar)
		// Should be '[' + barWidth + ']' = barWidth + 2 visual columns
		expectedLen := tc.barWidth + 2
		if visLen != expectedLen {
			t.Errorf("Expected bar visual width %d for pct %.1f, got %d", expectedLen, tc.pct, visLen)
		}
	}
}

func TestRenderCard(t *testing.T) {
	lines := []string{"Line 1", "Line 2"}
	width := 40
	height := 2

	card := RenderCard("TEST", lines, width, height)
	renderedLines := strings.Split(card, "\n")

	// height (2) + 2 borders = 4 lines
	if len(renderedLines) != height+2 {
		t.Fatalf("Expected %d lines, got %d", height+2, len(renderedLines))
	}

	for i, l := range renderedLines {
		visLen := lipgloss.Width(l)
		if visLen != width {
			t.Errorf("Line %d expected visual width %d, got %d: %q", i+1, width, visLen, l)
		}
	}
}

func TestModelLifecycle(t *testing.T) {
	collector, err := sysinfo.NewCollector()
	if err != nil {
		t.Skipf("Collector init skipped: %v", err)
	}
	defer collector.Close()

	m := NewModel(collector)

	// 1. Init command
	cmd := m.Init()
	if cmd == nil {
		t.Errorf("Expected non-nil Init tick command")
	}

	// 2. Resize
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = updated.(Model)
	if m.width != 100 || m.height != 30 {
		t.Errorf("Expected size 100x30, got %dx%d", m.width, m.height)
	}

	// 3. View inspection
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) < 10 || len(lines) > 12 {
		t.Errorf("Expected 10-12 terminal rows, got %d lines", len(lines))
	}

	// 4. Key exit
	updated, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)
	if !m.quitting {
		t.Errorf("Expected m.quitting to be true on 'q'")
	}
	if quitCmd == nil {
		t.Errorf("Expected tea.Quit command on 'q'")
	}
}

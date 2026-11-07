package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderCard formats a titled single card with precise borders and line padding.
// width is the total visual columns.
// height is the exact number of content lines (excluding top and bottom borders).
func RenderCard(title string, lines []string, width, height int) string {
	if width < 20 {
		width = 20
	}

	// 1. Top border: ┌─ TITLE ──────...──┐
	prefix := "┌─ "
	suffix := "┐"
	prefixLen := lipgloss.Width(prefix)
	suffixLen := lipgloss.Width(suffix)

	titleText := title + " "
	titleLen := lipgloss.Width(titleText)

	availDashes := width - prefixLen - titleLen - suffixLen
	if availDashes < 0 {
		// Truncate title so card width is strictly preserved
		maxTitleLen := width - prefixLen - suffixLen - 2
		if maxTitleLen > 3 {
			titleText = lipgloss.NewStyle().MaxWidth(maxTitleLen).Render(title) + " "
			titleLen = lipgloss.Width(titleText)
			availDashes = width - prefixLen - titleLen - suffixLen
			if availDashes < 0 {
				availDashes = 0
			}
		} else {
			titleText = ""
			availDashes = width - prefixLen - suffixLen
		}
	}

	topLine := borderStyle.Render(prefix) + titleText + borderStyle.Render(strings.Repeat("─", availDashes)+suffix)

	var sb strings.Builder
	sb.WriteString(topLine)
	sb.WriteString("\n")

	// 2. Content lines
	contentWidth := width - 2
	for i := 0; i < height; i++ {
		var line string
		if i < len(lines) {
			line = lines[i]
		}

		visLen := lipgloss.Width(line)
		if visLen > contentWidth {
			line = lipgloss.NewStyle().MaxWidth(contentWidth).Render(line)
			visLen = lipgloss.Width(line)
		}

		pad := contentWidth - visLen
		if pad < 0 {
			pad = 0
		}

		sb.WriteString(borderStyle.Render("│"))
		sb.WriteString(line)
		sb.WriteString(strings.Repeat(" ", pad))
		sb.WriteString(borderStyle.Render("│"))
		sb.WriteString("\n")
	}

	// 3. Bottom border: └────────────...──┘
	bottomDashes := width - 2
	if bottomDashes < 0 {
		bottomDashes = 0
	}
	bottomLine := borderStyle.Render("└" + strings.Repeat("─", bottomDashes) + "┘")
	sb.WriteString(bottomLine)

	return sb.String()
}

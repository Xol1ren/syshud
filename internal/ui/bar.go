package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderBar returns a styled ASCII progress bar string of the format:
// [████████░░░░░░░░]
func RenderBar(pct float64, barWidth int) string {
	if barWidth <= 0 {
		barWidth = 16
	}
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}

	filledLen := int(math.Round(pct / 100.0 * float64(barWidth)))
	if filledLen > barWidth {
		filledLen = barWidth
	}
	emptyLen := barWidth - filledLen

	var barColor lipgloss.TerminalColor
	switch {
	case pct >= 85:
		barColor = colorBarHigh
	case pct >= 60:
		barColor = colorBarMid
	default:
		barColor = colorBarLow
	}

	filledStyle := lipgloss.NewStyle().Foreground(barColor)
	emptyStyle := lipgloss.NewStyle().Foreground(colorBarOff)

	var b strings.Builder
	b.WriteString(brackStyle.Render("["))
	if filledLen > 0 {
		b.WriteString(filledStyle.Render(strings.Repeat("█", filledLen)))
	}
	if emptyLen > 0 {
		b.WriteString(emptyStyle.Render(strings.Repeat("░", emptyLen)))
	}
	b.WriteString(brackStyle.Render("]"))

	return b.String()
}


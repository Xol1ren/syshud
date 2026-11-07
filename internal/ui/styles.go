package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Border and title styles
	colorBorder = lipgloss.Color("240") // Dark gray
	colorTitle  = lipgloss.Color("39")  // Cyan
	colorBrack  = lipgloss.Color("245") // Dim white/gray for brackets

	borderStyle = lipgloss.NewStyle().Foreground(colorBorder)
	titleStyle  = lipgloss.NewStyle().Foreground(colorTitle).Bold(true)
	brackStyle  = lipgloss.NewStyle().Foreground(colorBrack)

	// Label and value styles
	colorLabel = lipgloss.Color("250") // Light gray
	colorValue = lipgloss.Color("255") // Bright white
	colorDim   = lipgloss.Color("244") // Muted gray for units and separators

	labelStyle = lipgloss.NewStyle().Foreground(colorLabel)
	valueStyle = lipgloss.NewStyle().Foreground(colorValue).Bold(true)
	dimStyle   = lipgloss.NewStyle().Foreground(colorDim)

	// Bar colors
	colorBarLow  = lipgloss.Color("42")  // Emerald green (< 60%)
	colorBarMid  = lipgloss.Color("214") // Amber (60% - 85%)
	colorBarHigh = lipgloss.Color("196") // Red (> 85%)
	colorBarOff  = lipgloss.Color("237") // Dim gray empty blocks

	// Footer style
	footerStyle = lipgloss.NewStyle().Foreground(colorDim)
)


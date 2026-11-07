package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"syshud/internal/sysinfo"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(1000*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// Model implements tea.Model for the syshud TUI.
type Model struct {
	collector *sysinfo.Collector
	stats     sysinfo.SystemStats
	width     int
	height    int
	quitting  bool
}

// NewModel creates an initialized Bubble Tea model with an initial telemetry sample.
func NewModel(collector *sysinfo.Collector) Model {
	m := Model{
		collector: collector,
		width:     80, // Default fallback
		height:    24,
	}
	_ = collector.Sample(&m.stats)
	return m
}

// Init starts the 1000ms periodic ticker.
func (m Model) Init() tea.Cmd {
	return tickCmd()
}

// Update handles terminal messages, resizing, and keyboard exits.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "Q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		_ = m.collector.Sample(&m.stats)
		return m, tickCmd()
	}

	return m, nil
}

// View formats the 2x2 grid HUD taking exactly 10-12 terminal rows.
func (m Model) View() string {
	if m.quitting {
		return ""
	}

	w := m.width
	if w <= 0 {
		w = 80
	}

	col1 := w / 2
	col2 := w - col1
	if col1 < 38 {
		col1 = 38
		col2 = 38
	}

	barWidth := 16
	if col1 >= 56 {
		barWidth = 20
	} else if col1 < 44 {
		barWidth = 12
	}

	// 1. CPU Card
	cpuTitle := titleStyle.Render("CPU") + " " + brackStyle.Render("[") + labelStyle.Render(m.stats.CPU.ModelName) + brackStyle.Render("]")
	cpuBar := RenderBar(m.stats.CPU.UsagePct, barWidth)
	cpuLine1 := fmt.Sprintf(" %s %s %s",
		labelStyle.Render("Usage:"),
		cpuBar,
		valueStyle.Render(fmt.Sprintf("%2.0f%%", m.stats.CPU.UsagePct)),
	)
	var cpuLine2 string
	if col1 >= 46 {
		cpuLine2 = fmt.Sprintf(" %s  %s    %s %s   %s %s",
			labelStyle.Render("Temp:"),
			valueStyle.Render(fmt.Sprintf("%2d°C", m.stats.CPU.TempC)),
			labelStyle.Render("Freq:"),
			valueStyle.Render(fmt.Sprintf("%.1f GHz", m.stats.CPU.FreqGHz)),
			labelStyle.Render("Power:"),
			valueStyle.Render(fmt.Sprintf("%.0fW", m.stats.CPU.PowerW)),
		)
	} else {
		cpuLine2 = fmt.Sprintf(" %s %s  %s %s  %s %s",
			labelStyle.Render("Temp:"),
			valueStyle.Render(fmt.Sprintf("%2d°C", m.stats.CPU.TempC)),
			labelStyle.Render("Freq:"),
			valueStyle.Render(fmt.Sprintf("%.1f GHz", m.stats.CPU.FreqGHz)),
			labelStyle.Render("Power:"),
			valueStyle.Render(fmt.Sprintf("%.0fW", m.stats.CPU.PowerW)),
		)
	}
	cardCPU := RenderCard(cpuTitle, []string{cpuLine1, cpuLine2}, col1, 2)

	// 2. MEMORY Card
	memTitle := titleStyle.Render("MEMORY")
	memLine1 := fmt.Sprintf(" %s   %s",
		labelStyle.Render("RAM:"),
		valueStyle.Render(fmt.Sprintf("%.1f / %.1f GB (%.0f%%)", m.stats.Mem.UsedGB, m.stats.Mem.TotalGB, m.stats.Mem.UsedPct)),
	)
	memLine2 := fmt.Sprintf(" %s %s %s %s %s",
		labelStyle.Render("Cache:"),
		valueStyle.Render(fmt.Sprintf("%.1f GB", m.stats.Mem.CacheGB)),
		dimStyle.Render("|"),
		labelStyle.Render("Free:"),
		valueStyle.Render(fmt.Sprintf("%.1f GB", m.stats.Mem.FreeGB)),
	)
	cardMem := RenderCard(memTitle, []string{memLine1, memLine2}, col2, 2)

	// 3. GPU Card
	var gpuTitle string
	var gpuLines []string
	if m.stats.GPU.Available {
		gpuTitle = titleStyle.Render("GPU") + " " + brackStyle.Render("[") + labelStyle.Render(m.stats.GPU.ModelName) + brackStyle.Render("]")
		gpuBar := RenderBar(float64(m.stats.GPU.UsagePct), barWidth)
		gpuLine1 := fmt.Sprintf(" %s %s %s",
			labelStyle.Render("Usage:"),
			gpuBar,
			valueStyle.Render(fmt.Sprintf("%2d%%", m.stats.GPU.UsagePct)),
		)
		gpuLine2 := fmt.Sprintf(" %s  %s",
			labelStyle.Render("VRAM:"),
			valueStyle.Render(fmt.Sprintf("%.1f / %.1f GB (%.0f%%)", m.stats.GPU.VRAMUsedGB, m.stats.GPU.VRAMTotalGB, m.stats.GPU.VRAMUsedPct)),
		)
		var gpuLine3 string
		if col1 >= 46 {
			gpuLine3 = fmt.Sprintf(" %s  %s    %s %s",
				labelStyle.Render("Temp:"),
				valueStyle.Render(fmt.Sprintf("%2d°C", m.stats.GPU.TempC)),
				labelStyle.Render("Power:"),
				valueStyle.Render(fmt.Sprintf("%.0fW", m.stats.GPU.PowerW)),
			)
		} else {
			gpuLine3 = fmt.Sprintf(" %s %s  %s %s",
				labelStyle.Render("Temp:"),
				valueStyle.Render(fmt.Sprintf("%2d°C", m.stats.GPU.TempC)),
				labelStyle.Render("Power:"),
				valueStyle.Render(fmt.Sprintf("%.0fW", m.stats.GPU.PowerW)),
			)
		}
		gpuLines = []string{gpuLine1, gpuLine2, gpuLine3}
	} else {
		gpuTitle = titleStyle.Render("GPU") + " " + brackStyle.Render("[") + labelStyle.Render("N/A") + brackStyle.Render("]")
		gpuLine1 := fmt.Sprintf(" %s %s", labelStyle.Render("Usage:"), dimStyle.Render("N/A (Disabled)"))
		gpuLine2 := fmt.Sprintf(" %s  %s", labelStyle.Render("VRAM:"), dimStyle.Render("N/A"))
		gpuLine3 := fmt.Sprintf(" %s  %s    %s %s", labelStyle.Render("Temp:"), dimStyle.Render("N/A"), labelStyle.Render("Power:"), dimStyle.Render("N/A"))
		gpuLines = []string{gpuLine1, gpuLine2, gpuLine3}
	}
	cardGPU := RenderCard(gpuTitle, gpuLines, col1, 3)

	// 4. DISK Card
	diskTitle := titleStyle.Render("DISK") + " " + brackStyle.Render("[") + labelStyle.Render(m.stats.Disk.MountPoint) + brackStyle.Render("]")
	diskBar := RenderBar(m.stats.Disk.UsedPct, barWidth)
	diskLine1 := fmt.Sprintf(" %s  %s",
		labelStyle.Render("Used:"),
		valueStyle.Render(fmt.Sprintf("%.2f TB / %.2f TB (%.0f%%)", m.stats.Disk.UsedTB, m.stats.Disk.TotalTB, m.stats.Disk.UsedPct)),
	)
	diskLine2 := fmt.Sprintf(" %s  %s",
		labelStyle.Render("Free:"),
		valueStyle.Render(fmt.Sprintf("%.2f TB", m.stats.Disk.FreeTB)),
	)
	diskLine3 := fmt.Sprintf(" %s   %s %s",
		labelStyle.Render("Bar:"),
		diskBar,
		valueStyle.Render(fmt.Sprintf("%2.0f%%", m.stats.Disk.UsedPct)),
	)
	cardDisk := RenderCard(diskTitle, []string{diskLine1, diskLine2, diskLine3}, col2, 3)

	// Assemble Grid
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, cardCPU, cardMem)
	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, cardGPU, cardDisk)

	// Footer: 1 row
	footerWidth := col1 + col2
	footerText := " syshud • tick: 1000ms • [q] quit"
	pad := footerWidth - lipgloss.Width(footerText)
	if pad < 0 {
		pad = 0
	}
	footer := footerStyle.Render(footerText + strings.Repeat(" ", pad))

	// Return 2x2 grid + footer (total 10 lines)
	return topRow + "\n" + bottomRow + "\n" + footer
}

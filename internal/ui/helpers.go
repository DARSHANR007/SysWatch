package ui

import "github.com/gdamore/tcell/v3"

// Color codes for status levels
const (
	ColorGood     = tcell.ColorGreen
	ColorWarning  = tcell.ColorYellow
	ColorCritical = tcell.ColorRed
	ColorDefault  = tcell.ColorWhite
)

// StatusColor returns appropriate color based on percentage
func StatusColor(percent float64) tcell.Color {
	if percent >= 90 {
		return ColorCritical
	}
	if percent >= 70 {
		return ColorWarning
	}
	return ColorGood
}

// ProgressBar creates a simple ASCII progress bar
func ProgressBar(percent float64, width int) string {
	if width < 3 {
		width = 3
	}
	filled := int((percent / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := "["
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	bar += "]"
	return bar
}

// CasualMessage returns a friendly message based on system state
func CasualMessage(cpuPercent, memPercent float64) string {
	if cpuPercent > 80 {
		return "CPU's working hard 🔥"
	}
	if memPercent > 85 {
		return "Memory's getting tight 💨"
	}
	if cpuPercent > 60 || memPercent > 60 {
		return "System's busy 🏃"
	}
	return "Running smooth ✓"
}

// StatusIndicator returns emoji or symbol based on bool
func StatusIndicator(online bool) string {
	if online {
		return "🟢"
	}
	return "🔴"
}

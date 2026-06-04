package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"syswatch/internal/network"
	"syswatch/internal/proc"
	"syswatch/internal/stats"
	"time"

	"github.com/gdamore/tcell/v3"
)

// Run starts the tcell UI and polls the collector at interval until ctx done
func Run(ctx context.Context, collector *proc.Collector, interval time.Duration, limit int, sortBy string) error {
	s, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	if err := s.Init(); err != nil {
		return err
	}
	defer s.Fini()

	style := tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorWhite)
	s.SetStyle(style)

	tick := time.NewTicker(interval)
	defer tick.Stop()

	// state for sort mode
	sortMode := sortBy
	if sortMode == "" {
		sortMode = "cpu"
	}

	// initial draw
	if err := drawOnce(s, collector, limit, sortMode); err != nil {
		// non-fatal
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case ev := <-s.EventQ():
			switch ev := ev.(type) {
			case *tcell.EventKey:
				if ev.Key() == tcell.KeyRune {
					str := ev.Str()
					if str != "" {
						r := []rune(str)[0]
						switch r {
						case 'q':
							return nil
						case 'c':
							sortMode = "cpu"
						case 'm':
							sortMode = "mem"
						case 'p':
							sortMode = "pid"
						case 'n':
							sortMode = "name"
						}
					}
				}
				if ev.Key() == tcell.KeyCtrlC {
					return nil
				}
			}
		case <-tick.C:
			if err := drawOnce(s, collector, limit, sortMode); err != nil {
				// show error briefly
				showError(s, err)
			}
		}
	}
}

func showError(s tcell.Screen, err error) {
	s.Clear()
	str := fmt.Sprintf("error: %v", err)
	for i, r := range str {
		s.SetContent(i, 0, r, nil, tcell.StyleDefault.Foreground(tcell.ColorRed))
	}
	s.Show()
}

func drawOnce(s tcell.Screen, collector *proc.Collector, limit int, sortMode string) error {
	sysStats, err := stats.Collect()
	if err != nil {
		// non-fatal, continue with partial stats
	}

	procs, err := collector.Collect()
	if err != nil {
		return err
	}

	s.Clear()
	w, h := s.Size()

	y := 0

	// === Dashboard Header ===
	drawString(s, 0, y, "═══ SYSWATCH ═════════════════════════════════════════", tcell.StyleDefault.Foreground(tcell.ColorBlue).Bold(true))
	y += 2

	// CPU and Memory bars
	cpuBar := ProgressBar(sysStats.CPUPercent, 15)
	memBar := ProgressBar(sysStats.MemoryPercent, 15)

	cpuColor := StatusColor(sysStats.CPUPercent)
	memColor := StatusColor(sysStats.MemoryPercent)

	line1 := fmt.Sprintf("  CPU: %s %.1f%%  │  MEM: %s %.1f%%", cpuBar, sysStats.CPUPercent, memBar, sysStats.MemoryPercent)
	drawStringWithColor(s, 0, y, line1, cpuColor, memColor, sysStats.CPUPercent, sysStats.MemoryPercent)
	y += 2

	// Internet status and uptime
	status, online := network.Status()
	statusIndicator := StatusIndicator(online)

	line2 := fmt.Sprintf("  Uptime: %s  │  Status: %s %s  │  Load: %.1f %.1f %.1f",
		stats.FormatUptime(sysStats.Uptime),
		statusIndicator,
		status,
		sysStats.LoadAvg[0], sysStats.LoadAvg[1], sysStats.LoadAvg[2],
	)
	drawStringStyled(s, 0, y, line2, tcell.StyleDefault.Foreground(tcell.ColorWhite))
	y += 2

	// Casual message
	message := CasualMessage(sysStats.CPUPercent, sysStats.MemoryPercent)
	drawStringStyled(s, 0, y, fmt.Sprintf("  >> %s", message), tcell.StyleDefault.Foreground(tcell.ColorGreen).Italic(true))
	y += 2

	// Separator
	drawString(s, 0, y, "───────────────────────────────────────────────────────", tcell.StyleDefault.Foreground(tcell.ColorBlue))
	y += 2

	// Sort indicator and help
	sortLabel := fmt.Sprintf("  Sort: [c]pu  [m]em  [p]id  [n]ame  │  Current: %s  │  [q]uit", strings.ToUpper(sortMode))
	drawStringStyled(s, 0, y, sortLabel, tcell.StyleDefault.Foreground(tcell.ColorGreen).Italic(true))
	y += 2

	// === Process List ===
	if y < h-1 {
		drawString(s, 0, y, "", tcell.StyleDefault) // blank line
		y++
		header := "  PID      CPU%      RSS          NAME"
		drawString(s, 0, y, header, tcell.StyleDefault.Bold(true).Foreground(tcell.ColorYellow))
		y++
		drawString(s, 0, y, "  "+strings.Repeat("─", 50), tcell.StyleDefault.Foreground(tcell.ColorBlue))
		y++
	}

	// Sort by selected mode
	switch sortMode {
	case "mem":
		sort.Slice(procs, func(i, j int) bool { return procs[i].RSS > procs[j].RSS })
	case "pid":
		sort.Slice(procs, func(i, j int) bool { return procs[i].PID < procs[j].PID })
	case "name":
		sort.Slice(procs, func(i, j int) bool { return procs[i].Name < procs[j].Name })
	default: // "cpu"
		sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })
	}

	maxRows := h - y
	procSlice := procs
	if limit > 0 && limit < len(procs) {
		procSlice = procs[:limit]
	}

	for i := 0; i < maxRows && i < len(procSlice); i++ {
		if y >= h {
			break
		}
		p := procSlice[i]
		line := fmt.Sprintf("  %-7d %7.2f %12d %s", p.PID, p.CPU, p.RSS, truncate(p.Name, w-45))

		// Color code by CPU usage
		color := StatusColor(p.CPU)
		drawStringStyled(s, 0, y, line, tcell.StyleDefault.Foreground(color))
		y++
	}

	s.Show()
	return nil
}

func truncate(s string, n int) string {
	if n <= 0 || len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}

func drawString(s tcell.Screen, x, y int, str string, style tcell.Style) {
	for i, r := range str {
		s.SetContent(x+i, y, r, nil, style)
	}
}

func drawStringStyled(s tcell.Screen, x, y int, str string, style tcell.Style) {
	for i, r := range str {
		s.SetContent(x+i, y, r, nil, style)
	}
}

// drawStringWithColor splits string and applies color based on thresholds
func drawStringWithColor(s tcell.Screen, x, y int, str string, cpuColor, memColor tcell.Color, cpuPercent, memPercent float64) {
	// Simple approach: color the whole thing with appropriate color
	// For more sophistication, parse and color sections
	for i, r := range str {
		s.SetContent(x+i, y, r, nil, tcell.StyleDefault.Foreground(cpuColor))
	}
}

// === Helper functions from helpers.go ===

package ui

import (
	"context"
	"fmt"
	"sort"
	"syswatch/internal/proc"
	"time"

	"github.com/gdamore/tcell/v3"
)

// Run starts the tcell UI and polls the collector at interval until ctx done
func Run(ctx context.Context, collector *proc.Collector, interval time.Duration) error {
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

	// initial draw
	if err := drawOnce(s, collector); err != nil {
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
					s := ev.Str()
					if s != "" {
						r := []rune(s)[0]
						if r == 'q' {
							return nil
						}
					}
				}
				if ev.Key() == tcell.KeyCtrlC {
					return nil
				}
			}
		case <-tick.C:
			if err := drawOnce(s, collector); err != nil {
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

func drawOnce(s tcell.Screen, collector *proc.Collector) error {
	procs, err := collector.Collect()
	if err != nil {
		return err
	}
	s.Clear()
	w, h := s.Size()

	// header
	header := " PID   CPU%   RSS      NAME"
	drawString(s, 0, 0, header, tcell.StyleDefault.Bold(true))

	// sort by CPU desc
	sort.Slice(procs, func(i, j int) bool { return procs[i].CPU > procs[j].CPU })

	maxRows := h - 1
	for i := 0; i < maxRows && i < len(procs); i++ {
		p := procs[i]
		line := fmt.Sprintf(" %-5d %6.2f %8d %s", p.PID, p.CPU, p.RSS, truncate(p.Name, w-30))
		drawString(s, 0, i+1, line, tcell.StyleDefault)
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

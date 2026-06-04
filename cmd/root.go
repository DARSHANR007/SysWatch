package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"syswatch/internal/proc"
	"syswatch/internal/ui"
	"time"

	"github.com/spf13/cobra"
)

var (
	refresh int
	limit   int
	sortBy  string
)

var rootCmd = &cobra.Command{
	Use:   "syswatch",
	Short: "Lightweight system monitor for Ubuntu",
	Long: `
╔════════════════════════════════════════════════════════════╗
║                     SYSWATCH v1.0.0                        ║
║           Lightweight Process Monitor for Ubuntu           ║
╚════════════════════════════════════════════════════════════╝

A minimal, fast system monitoring tool with a clean TUI.
Monitor CPU, memory, processes, and network status in real-time.

Usage:
  syswatch monitor              Start the interactive monitor
  syswatch monitor --refresh 2  Custom refresh interval (seconds)
  syswatch monitor --limit 20   Show top 20 processes only
  syswatch monitor --sort mem   Sort by memory by default

Keyboard Shortcuts (in monitor):
  [c] - Sort by CPU         [m] - Sort by Memory
  [p] - Sort by PID         [n] - Sort by Name
  [q] - Quit                Ctrl+C - Quit

Examples:
  syswatch monitor
  syswatch monitor -r 2 -l 30 -s mem
  syswatch monitor --refresh 5

Learn more: https://github.com/DARSHANR007/SysWatch
`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show help if no subcommand provided
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add monitor subcommand
	rootCmd.AddCommand(monitorCmd)

	// Monitor-specific flags
	monitorCmd.Flags().IntVarP(&refresh, "refresh", "r", 1, "refresh interval in seconds")
	monitorCmd.Flags().IntVarP(&limit, "limit", "l", 0, "max processes to display (0 = full screen)")
	monitorCmd.Flags().StringVarP(&sortBy, "sort", "s", "cpu", "sort by: cpu, mem, pid, name")
}

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Start the interactive process monitor",
	Long: `
Start the SysWatch interactive TUI monitor.

Watch processes in real-time with system stats including CPU, memory,
uptime, load average, and network status.

Usage:
  syswatch monitor [flags]

Flags:
  -r, --refresh int    Refresh interval in seconds (default 1)
  -l, --limit int      Max processes to display (default 0 = all)
  -s, --sort string    Default sort column: cpu, mem, pid, name (default "cpu")

Examples:
  syswatch monitor
  syswatch monitor --refresh 2
  syswatch monitor -l 20 -s mem
  syswatch monitor -r 5 -l 30 -s name
`,
	Run: func(cmd *cobra.Command, args []string) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// handle signals
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			cancel()
		}()

		collector := proc.NewCollector()
		refreshDur := time.Duration(refresh) * time.Second

		// run UI; it will poll collector each tick
		if err := ui.Run(ctx, collector, refreshDur, limit, sortBy); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

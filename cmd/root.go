package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"syswatch/internal/proc"
	"syswatch/internal/ui"

	"github.com/spf13/cobra"
)

var (
	refresh int
)

var rootCmd = &cobra.Command{
	Use:   "syswatch",
	Short: "Lightweight process monitor",
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
		if err := ui.Run(ctx, collector, refreshDur); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&refresh, "refresh", "r", 1, "refresh interval in seconds")
}

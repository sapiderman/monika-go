// Package cmd implements the monika-go command-line interface.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"monika-go/internal/alert"
	"monika-go/internal/config"
	"monika-go/internal/logger"
	"monika-go/internal/notification"
	"monika-go/internal/scheduler"

	"github.com/spf13/cobra"
)

var (
	cfgFile     string
	probeIDs    string
	repeatCount int
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "monika-go",
	Short: "Monika command line monitoring tool",
	Long:  `Monika-go is the golang port of the Monika command line monitoring tool.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log := logger.New("root")
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		if probeIDs != "" {
			if err := filterProbes(cfg, probeIDs); err != nil {
				return fmt.Errorf("--id: %w", err)
			}
		}
		cfg.Repeat = repeatCount
		log.Info("config loaded", logger.F("source", cfgFile), logger.F("probes", len(cfg.Probes)))
		return run(cfg, log)
	},
}

// run is the entry point for the prober engine.
// It is extracted from the cobra command so it can be tested independently.
func run(cfg *config.Config, log logger.Logger) error {
	sched := scheduler.New(cfg, log)

	// Initialize and register all configured notifiers
	for _, nCfg := range cfg.Notifications {
		notifier, err := notification.NewNotifier(nCfg, log)
		if err != nil {
			log.Error("failed to initialize notifier", logger.F("notifier_id", nCfg.ID), logger.Err(err))
			continue
		}

		sched.AlertManager().RegisterListener(func(event alert.TransitionEvent) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := notifier.Notify(ctx, event); err != nil {
				log.Error("failed to dispatch notification", logger.F("notifier_id", notifier.ID()), logger.Err(err))
			}
		})
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("scheduler start: %w", err)
	}

	log.Info("scheduler started")

	// Block until shutdown is signaled or --repeat runs are exhausted
	allDone := make(chan struct{})
	go func() {
		defer close(allDone)
		sched.Wait()
	}()

	select {
	case <-sigChan:
		log.Info("shutdown signal received, stopping scheduler gracefully...")
	case <-allDone:
		log.Info("all probes completed")
	}
	sched.Stop()

	return nil
}

// filterProbes keeps only the probes whose IDs appear in the comma-separated
// ids string, preserving config order. All referenced IDs must exist, per
// reference Monika semantics.
func filterProbes(cfg *config.Config, ids string) error {
	wanted := make(map[string]bool)
	var order []string
	for _, id := range strings.Split(ids, ",") {
		id = strings.TrimSpace(id)
		if id != "" && !wanted[id] {
			wanted[id] = true
			order = append(order, id)
		}
	}
	kept := make([]config.Probe, 0, len(order))
	for _, p := range cfg.Probes {
		if wanted[p.ID] {
			kept = append(kept, p)
			delete(wanted, p.ID)
		}
	}
	if len(wanted) > 0 {
		unknown := make([]string, 0, len(wanted))
		for _, id := range order {
			if wanted[id] {
				unknown = append(unknown, id)
			}
		}
		return fmt.Errorf("unknown probe id(s): %s", strings.Join(unknown, ", "))
	}
	cfg.Probes = kept
	return nil
}

func Execute() {
	logger.InitLogger()
	log := logger.New("root")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "monika.yaml", "config file path")
	rootCmd.PersistentFlags().StringVarP(&probeIDs, "id", "i", "", "comma-separated probe IDs to run (e.g. 1,3)")
	rootCmd.PersistentFlags().IntVarP(&repeatCount, "repeat", "r", 0, "number of times to run each probe before exiting (default: forever)")

	if err := rootCmd.Execute(); err != nil {
		log.Error("fatal", logger.Err(err))
		os.Exit(1)
	}
}

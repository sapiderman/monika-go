package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monika-go/internal/alert"
	"monika-go/internal/config"
	"monika-go/internal/logger"
	"monika-go/internal/notification"
	"monika-go/internal/scheduler"

	"github.com/spf13/cobra"
)

var cfgFile string

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

	// Block until a shutdown signal is received
	<-sigChan

	log.Info("shutdown signal received, stopping scheduler gracefully...")
	sched.Stop()

	return nil
}

func Execute() {
	logger.InitLogger()
	log := logger.New("root")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "monika.yaml", "config file path")

	if err := rootCmd.Execute(); err != nil {
		log.Error("fatal", logger.Err(err))
		os.Exit(1)
	}
}

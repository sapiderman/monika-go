package cmd

import (
	"fmt"
	"os"

	"monika-go/internal/config"
	"monika-go/internal/logger"

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
	_ = cfg
	_ = log
	// TODO: build and start prober engine from config
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

package cmd

import (
	"fmt"
	"os"

	"monika-go/internal/config"
	"monika-go/internal/logger"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "monika-go",
	Short: "Monika command line monitoring tool",
	Long:  `Monika-go is the golang port of the Monika command line monitoring tool.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		parsed, err := config.Parse(cfgFile)
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if err := config.Validate(parsed); err != nil {
			return fmt.Errorf("validating config: %w", err)
		}
		cfg = parsed
		logrus.Infof("Config loaded: %s (%d probes)", cfgFile, len(cfg.Probes))
		return nil
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		logrus.Errorf("Error: %v", err)
		os.Exit(1)
	}
}

func init() {
	// Initialize the logger
	logger.InitLogger()
	logrus.Info("root....")

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "monika.yaml", "config file path")
}

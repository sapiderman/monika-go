// Package cmd: main command package
package cmd

import (
	"monika-go/internal/config"
	"os"

	"monika-go/internal/logger"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "monika-go",
	Short: "Monika command line monitoring tool",
	Long:  `Monika-go is the golang port of the Monika command line monitoring tool.`,

	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {

	// Initialize the logger
	logger.InitLogger()
	logrus.Info("root....")

	err := config.LoadDefaultConfig()
	if err != nil {
		// handle error
		logrus.Errorf("Error parsing config file: %v", err)
		os.Exit(1)
	}

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.monika-go.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

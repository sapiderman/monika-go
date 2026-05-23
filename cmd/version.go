package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version number",
	Long:  `Returns the version number of the application.`,

	Run: func(cmd *cobra.Command, args []string) {
		// Hardcoded for now, should be set via ldflags
		fmt.Println("version: 0.0.1")
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

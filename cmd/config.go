package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Use a config file",
	Long: `Specify a yaml config file. If not specified, Monika will look for monika.yaml.
	Usage: monika --config my-project.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("config called")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func readConfigFile() error {
	// read config file
	viper.SetConfigName("monika") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	// viper.AddConfigPath("$HOME/.monika")   // adding home directory as first search path
	// viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	// err := viper.ReadInConfig() // Find and read the config file
	// if err != nil { // Handle errors reading the config file
	// 	fmt.Printf("Error reading config file, %s", err)
	// }

	return nil
}

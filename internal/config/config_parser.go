// Package config does teh actual config parsing
package config

import "github.com/spf13/viper"

func ParseConfig(fn string) error {
	// Set the default values
	// viper.SetDefault("app.version", "0.0.1")

	// read config file
	viper.SetConfigName("monika") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	// viper.AddConfigPath("$HOME/.monika")   // adding home directory as first search path
	// viper.AddConfigPath("/etc/appname/")  // path to look for the config file in
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		return err
	}

	return nil
}

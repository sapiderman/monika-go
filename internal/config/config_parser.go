// Package config does the	 actual config parsing
package config

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

// Config represents the structure of your YAML configuration
type Config struct {
	Probes        []Probe        `mapstructure:"probes"`
	Notifications []Notification `mapstructure:"notifications"`
}

// Probe represents a single probe configuration
type Probe struct {
	ID                 string    `mapstructure:"id"`
	Name               string    `mapstructure:"name"`
	Description        string    `mapstructure:"description"`
	Interval           int       `mapstructure:"interval"` // Added interval as it's a common field
	Requests           []Request `mapstructure:"requests"`
	Alerts             []Alert   `mapstructure:"alerts"` // Added for alerts example
	IncidentThreshold  int       `mapstructure:"incidentThreshold"`
	RecovveryThreshold int       `mapstructure:"recoveryThreshold"`
}

// Request represents a single HTTP request within a probe
type Request struct {
	URL               string            `mapstructure:"url"`
	Method            string            `mapstructure:"method"`
	Timeout           int               `mapstructure:"timeout"`
	Headers           map[string]string `mapstructure:"headers"`
	Body              map[string]any    `mapstructure:"body"` // Can be map or string for plain text/xml
	SaveBody          bool              `mapstructure:"saveBody"`
	AllowUnauthorized bool              `mapstructure:"allowUnauthorized"`
	FollowRedirects   bool              `mapstructure:"followRedirects"`
}

// Ping represents a ping configuration
type Ping struct {
	URI string `mapstructure:"uri"`
}

// Alert represents an alert configuration
type Alert struct {
	Assertion string `mapstructure:"assertion"`
	Message   string `mapstructure:"message"`
}

// Notification represents a notification channel configuration
type Notification struct {
	ID   string                 `mapstructure:"id"`
	Type string                 `mapstructure:"type"`
	Data map[string]interface{} `mapstructure:"data"`
}

func LoadDefaultConfig() error {
	// Set the default values
	viper.SetDefault("app.version", "0.0.1") // Set the default values

	_, err := ParseConfig("monika.yaml")
	if err != nil {
		log.Errorf("Error parsing config file: %v", err)
		return err
	}

	return nil
}

func ParseConfig(fn string) (*Config, error) {
	logf := log.WithField("fn", "ParseConfig")
	logf.Info("loading config...")

	// read config file
	viper.SetConfigName("monika") // name of config file (without extension)
	viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory
	// viper.AddConfigPath("$HOME/.monika")   // adding home directory as first search path
	// viper.AddConfigPath("/etc/appname/")  // path to look for the config file in

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found: %w", err)
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

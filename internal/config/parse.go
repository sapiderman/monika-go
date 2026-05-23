package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Parse reads a YAML file and returns a Config struct.
// Returns an error if the file cannot be read or contains unknown fields.
func Parse(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	return ParseBytes(data)
}

// ParseBytes unmarshals YAML bytes into a Config struct.
// Uses strict decoding to reject unknown fields.
func ParseBytes(data []byte) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true) // strict mode: reject unknown fields

	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

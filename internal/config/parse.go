package config

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const remoteConfigTimeout = 30 * time.Second

// Load reads, parses, and validates a config from a local file path or a remote HTTP/HTTPS URL.
// This is the primary entry point for config loading. Use this instead of calling
// Parse and Validate separately.
func Load(source string) (*Config, error) {
	cfg, err := Parse(source)
	if err != nil {
		return nil, err
	}
	if err := Validate(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Parse reads a YAML config from a local file path or a remote HTTP/HTTPS URL.
// Returns an error if the source cannot be read or contains unknown fields.
// For most callers, prefer Load which also validates the config.
func Parse(source string) (*Config, error) {
	var data []byte
	var err error

	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		data, err = fetchURL(source)
		if err != nil {
			return nil, fmt.Errorf("fetch remote config: %w", err)
		}
	} else {
		data, err = os.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	return ParseBytes(data)
}

// fetchURL downloads a remote config file over HTTP/HTTPS.
func fetchURL(url string) ([]byte, error) {
	client := &http.Client{Timeout: remoteConfigTimeout}

	resp, err := client.Get(url) //nolint:noctx // timeout is enforced via http.Client
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body from %s: %w", url, err)
	}

	return data, nil
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
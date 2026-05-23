package config

import (
	"testing"
)

func TestParseBytes(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    name: "Test Probe"
    description: "A test probe"
    interval: 10
    requests:
      - url: "https://example.com"
        method: "GET"
        timeout: 5000
    alerts:
      - assertion: "response.status != 200"
        message: "Status not 200"
notifications:
  - id: "notif-1"
    type: "slack"
    data:
      url: "https://slack.com/webhook"
`)

	cfg, err := ParseBytes(yamlData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(cfg.Probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(cfg.Probes))
	}
	if cfg.Probes[0].ID != "1" {
		t.Errorf("expected probe ID '1', got %q", cfg.Probes[0].ID)
	}
	if cfg.Probes[0].Requests == nil {
		t.Fatalf("expected requests to be parsed")
	}
	if len(*cfg.Probes[0].Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(*cfg.Probes[0].Requests))
	}
	if (*cfg.Probes[0].Requests)[0].URL != "https://example.com" {
		t.Errorf("expected request URL 'https://example.com', got %q", (*cfg.Probes[0].Requests)[0].URL)
	}
}

func TestParseBytes_UnknownField(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    timout: 5000 # typo, should be timeout or at request level
    requests:
      - url: "https://example.com"
`)

	_, err := ParseBytes(yamlData)
	if err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestParseBytes_UnknownFieldInRequest(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    requests:
      - url: "https://example.com"
        timout: 5000
`)
	_, err := ParseBytes(yamlData)
	if err == nil {
		t.Fatal("expected error for unknown field in request, got nil")
	}
}

func TestParseBytes_UnknownFieldTopLevel(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    requests:
      - url: "https://example.com"
unknown_top_field: true
`)
	_, err := ParseBytes(yamlData)
	if err == nil {
		t.Fatal("expected error for unknown top-level field, got nil")
	}
}

func TestParseBytes_EmptyFile(t *testing.T) {
	yamlData := []byte(``)
	_, err := ParseBytes(yamlData)
	if err == nil {
		t.Fatal("expected error for empty file, got nil")
	}
}

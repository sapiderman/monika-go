package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"monika-go/internal/assertion"
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
	requests, ok := cfg.Probes[0].Spec.(*HTTPSpec)
	if !ok {
		t.Fatalf("expected HTTPSpec, got %T", cfg.Probes[0].Spec)
	}
	if len(requests.Requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(requests.Requests))
	}
	if requests.Requests[0].URL != "https://example.com" {
		t.Errorf("expected request URL 'https://example.com', got %q", requests.Requests[0].URL)
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

const validYAML = `
probes:
  - id: "remote-1"
    name: "Remote Probe"
    interval: 30
    requests:
      - url: "https://example.com"
        method: "GET"
`

func TestParse_RemoteURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validYAML))
	}))
	defer ts.Close()

	cfg, err := Parse(ts.URL)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(cfg.Probes))
	}
	if cfg.Probes[0].ID != "remote-1" {
		t.Errorf("expected probe ID 'remote-1', got %q", cfg.Probes[0].ID)
	}
}

func TestParse_RemoteURL_Non200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := Parse(ts.URL)
	if err == nil {
		t.Fatal("expected error for non-200 response, got nil")
	}
}

func TestParse_RemoteURL_InvalidYAML(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not: valid: yaml: ["))
	}))
	defer ts.Close()

	_, err := Parse(ts.URL)
	if err == nil {
		t.Fatal("expected parse error for invalid YAML, got nil")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, _ = f.WriteString(validYAML)
	f.Close()

	cfg, err := Load(f.Name())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Probes) != 1 {
		t.Errorf("expected 1 probe, got %d", len(cfg.Probes))
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	// Valid YAML but no probes — validation should reject it.
	yamlData := []byte("probes: []\n")
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	_, _ = f.Write(yamlData)
	f.Close()

	_, err = Load(f.Name())
	if err == nil {
		t.Fatal("expected validation error for empty probes, got nil")
	}
	if !strings.Contains(err.Error(), "at least one probe") {
		t.Errorf("expected probe validation error, got: %v", err)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestParseBytes_AssertionCompiled(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    requests:
      - url: "https://example.com"
        method: GET
        alerts:
          - assertion: "response.status != 200"
            message: Status not 200
`)

	cfg, err := ParseBytes(yamlData)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	httpSpec, ok := cfg.Probes[0].Spec.(*HTTPSpec)
	if !ok {
		t.Fatalf("expected HTTPSpec, got %T", cfg.Probes[0].Spec)
	}
	alerts := httpSpec.Requests[0].Alerts
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	a := alerts[0]
	if a.Assertion == nil {
		t.Fatal("expected Assertion to be compiled, got nil")
	}
	if a.Assertion.String() != "response.status != 200" {
		t.Errorf("expected assertion string %q, got %q", "response.status != 200", a.Assertion.String())
	}
	// Evaluate the compiled assertion
	result := assertion.ProbeResult{Status: 500}
	if !a.Assertion.Evaluate(result) {
		t.Error("expected assertion to be true for status 500")
	}
}

func TestParseBytes_InvalidAssertion(t *testing.T) {
	yamlData := []byte(`
probes:
  - id: "1"
    requests:
      - url: "https://example.com"
        alerts:
          - assertion: "response.stats != 200"
            message: typo
`)

	_, err := ParseBytes(yamlData)
	if err == nil {
		t.Fatal("expected error for invalid assertion, got nil")
	}
	if !strings.Contains(err.Error(), "response.stats") {
		t.Errorf("expected error mentioning 'response.stats', got: %v", err)
	}
}

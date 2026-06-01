package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// --- Probe.UnmarshalYAML -----------------------------------------------------

func TestProbeUnmarshalYAML_HTTP(t *testing.T) {
	yamlData := `
id: "http-1"
name: HTTP Probe
interval: 30
requests:
  - url: "https://example.com"
    method: GET
    timeout: 5000
alerts:
  - assertion: "response.status != 200"
    message: "not ok"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if p.ID != "http-1" {
		t.Errorf("ID = %q, want http-1", p.ID)
	}
	if p.Name != "HTTP Probe" {
		t.Errorf("Name = %q, want HTTP Probe", p.Name)
	}
	if p.Interval != 30 {
		t.Errorf("Interval = %d, want 30", p.Interval)
	}

	spec, ok := p.Spec.(*HTTPSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *HTTPSpec", p.Spec)
	}
	if len(spec.Requests) != 1 {
		t.Fatalf("len(Requests) = %d, want 1", len(spec.Requests))
	}
	if spec.Requests[0].URL != "https://example.com" {
		t.Errorf("Request URL = %q", spec.Requests[0].URL)
	}

	if len(p.Alerts) != 1 {
		t.Fatalf("len(Alerts) = %d, want 1", len(p.Alerts))
	}
	if p.Alerts[0].Message != "not ok" {
		t.Errorf("Alert message = %q, want not ok", p.Alerts[0].Message)
	}
}

func TestProbeUnmarshalYAML_Ping(t *testing.T) {
	yamlData := `
id: "ping-1"
ping:
  - uri: "https://example.com"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*PingSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *PingSpec", p.Spec)
	}
	if spec.Kind() != KindPing {
		t.Errorf("Kind = %v, want ping", spec.Kind())
	}
	if len(spec.Targets) != 1 {
		t.Errorf("len(Targets) = %d, want 1", len(spec.Targets))
	}
	if spec.Targets[0].URI != "https://example.com" {
		t.Errorf("URI = %q", spec.Targets[0].URI)
	}
}

func TestProbeUnmarshalYAML_Socket(t *testing.T) {
	yamlData := `
id: "sock-1"
socket:
  - host: "localhost"
    port: 8080
    data: "ping"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*SocketSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *SocketSpec", p.Spec)
	}
	if spec.Kind() != KindSocket {
		t.Errorf("Kind = %v, want socket", spec.Kind())
	}
	if len(spec.Targets) != 1 {
		t.Errorf("len(Targets) = %d, want 1", len(spec.Targets))
	}
}

func TestProbeUnmarshalYAML_MongoDB(t *testing.T) {
	yamlData := `
id: "mongo-1"
mongo:
  - uri: "mongodb://localhost:27017"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*MongoDBSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *MongoDBSpec", p.Spec)
	}
	if spec.Kind() != KindMongoDB {
		t.Errorf("Kind = %v, want mongo", spec.Kind())
	}
}

func TestProbeUnmarshalYAML_Redis(t *testing.T) {
	yamlData := `
id: "redis-1"
redis:
  - uri: "redis://localhost:6379"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*RedisSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *RedisSpec", p.Spec)
	}
	if spec.Kind() != KindRedis {
		t.Errorf("Kind = %v, want redis", spec.Kind())
	}
}

func TestProbeUnmarshalYAML_Postgres(t *testing.T) {
	yamlData := `
id: "pg-1"
postgres:
  - uri: "postgres://localhost:5432/db"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*PostgresSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *PostgresSpec", p.Spec)
	}
	if spec.Kind() != KindPostgres {
		t.Errorf("Kind = %v, want postgres", spec.Kind())
	}
}

func TestProbeUnmarshalYAML_MariaDB(t *testing.T) {
	yamlData := `
id: "maria-1"
mariadb:
  - uri: "mariadb://localhost:3306/db"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*MariaDBSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *MariaDBSpec", p.Spec)
	}
	if spec.Kind() != KindMariaDB {
		t.Errorf("Kind = %v, want mariadb", spec.Kind())
	}
}

func TestProbeUnmarshalYAML_MySQL(t *testing.T) {
	yamlData := `
id: "mysql-1"
mysql:
  - uri: "mysql://localhost:3306/db"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*MySQLSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *MySQLSpec", p.Spec)
	}
	if spec.Kind() != KindMySQL {
		t.Errorf("Kind = %v, want mysql", spec.Kind())
	}
}

func TestProbeUnmarshalYAML_NoProbeType(t *testing.T) {
	yamlData := `
id: "no-type"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err == nil {
		t.Fatal("expected error for probe with no type, got nil")
	}
	if err.Error() != `probe "no-type": must specify exactly one probe type` {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProbeUnmarshalYAML_MultipleProbeTypes(t *testing.T) {
	yamlData := `
id: "multi"
requests:
  - url: "https://example.com"
ping:
  - uri: "https://example.com"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err == nil {
		t.Fatal("expected error for multiple probe types, got nil")
	}
	if err.Error() != `probe "multi": must specify exactly one probe type, found 2` {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProbeUnmarshalYAML_MySQLAlias(t *testing.T) {
	// mysql: should produce a MySQLSpec, not MariaDBSpec
	yamlData := `
id: "mysql-alias"
mysql:
  - uri: "mysql://localhost:3306/db"
`

	var p Probe
	err := yaml.Unmarshal([]byte(yamlData), &p)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	spec, ok := p.Spec.(*MySQLSpec)
	if !ok {
		t.Fatalf("Spec = %T, want *MySQLSpec", p.Spec)
	}
	if spec.Kind() != KindMySQL {
		t.Errorf("Kind = %v, want mysql", spec.Kind())
	}
}

// --- RequestBody.UnmarshalYAML -----------------------------------------------

func TestRequestBodyUnmarshalYAML_String(t *testing.T) {
	yamlData := `body: "plain text body"`

	var s struct {
		Body RequestBody `yaml:"body"`
	}
	err := yaml.Unmarshal([]byte(yamlData), &s)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if !s.Body.IsText() {
		t.Error("expected IsText() = true")
	}
	if s.Body.Text() != "plain text body" {
		t.Errorf("Text() = %q, want plain text body", s.Body.Text())
	}
	if s.Body.Form() != nil {
		t.Error("expected Form() = nil for text body")
	}
}

func TestRequestBodyUnmarshalYAML_Form(t *testing.T) {
	yamlData := `
body:
  username: "admin"
  password: "secret"
`

	var s struct {
		Body RequestBody `yaml:"body"`
	}
	err := yaml.Unmarshal([]byte(yamlData), &s)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if s.Body.IsText() {
		t.Error("expected IsText() = false for form body")
	}
	if s.Body.Text() != "" {
		t.Errorf("expected Text() = \"\" for form body, got %q", s.Body.Text())
	}
	form := s.Body.Form()
	if form == nil {
		t.Fatal("expected Form() to be non-nil for form body")
	}
	if form["username"] != "admin" {
		t.Errorf("form[username] = %v, want admin", form["username"])
	}
	if form["password"] != "secret" {
		t.Errorf("form[password] = %v, want secret", form["password"])
	}
}

func TestRequestBodyUnmarshalYAML_Empty(t *testing.T) {
	// No body field — should be zero value
	var s struct {
		Body RequestBody `yaml:"body"`
	}
	err := yaml.Unmarshal([]byte(`body:`), &s)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if s.Body.IsText() {
		t.Error("IsText() should be false for absent/empty body")
	}
	if s.Body.Text() != "" {
		t.Errorf("Text() = %q, want empty", s.Body.Text())
	}
	if s.Body.Form() != nil {
		t.Error("Form() should be nil for absent/empty body")
	}
}

func TestRequestBodyUnmarshalYAML_ZeroValue(t *testing.T) {
	var rb RequestBody
	if rb.IsText() {
		t.Error("zero-value RequestBody should not be text")
	}
	if rb.Text() != "" {
		t.Error("zero-value RequestBody should have empty text")
	}
	if rb.Form() != nil {
		t.Error("zero-value RequestBody should have nil form")
	}
}

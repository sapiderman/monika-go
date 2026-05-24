package config

import "monika-go/internal/assertion"

// Config represents the top-level monika.yaml configuration.
type Config struct {
	Probes        []Probe        `yaml:"probes"`
	Notifications []Notification `yaml:"notifications"`
}

// Probe represents a single monitoring target.
// Spec is populated by UnmarshalYAML and is always non-nil after successful parse.
type Probe struct {
	ID                string    `yaml:"id"`
	Name              string    `yaml:"name"`
	Description       string    `yaml:"description"`
	Interval          int       `yaml:"interval"`
	IncidentThreshold int       `yaml:"incidentThreshold"`
	RecoveryThreshold int       `yaml:"recoveryThreshold"`
	Alerts            []Alert   `yaml:"alerts"`
	Spec              ProbeSpec `yaml:"-"` // set by UnmarshalYAML
}

// RequestBody represents an HTTP request body.
// At YAML parse time, only string (raw text/XML/JSON) and
// mapping (form-encoded key-value pairs) are accepted.
type RequestBody struct {
	text string
	form map[string]any
}

// IsText reports whether the body is a raw text string.
func (b RequestBody) IsText() bool { return b.form == nil && b.text != "" }

// Text returns the body as a string. Empty if not a text body.
func (b RequestBody) Text() string { return b.text }

// Form returns the body as form-encoded key-value pairs. Nil if not a form body.
func (b RequestBody) Form() map[string]any { return b.form }

// Request represents a single HTTP request within a probe.
type Request struct {
	Method            string            `yaml:"method"`
	URL               string            `yaml:"url"`
	Timeout           int               `yaml:"timeout"`
	SaveBody          bool              `yaml:"saveBody"`
	AllowUnauthorized bool              `yaml:"allowUnauthorized"`
	FollowRedirects   int               `yaml:"followRedirects"`
	Headers           map[string]string `yaml:"headers"`
	Body              RequestBody       `yaml:"body"`
	Alerts            []Alert           `yaml:"alerts"`
	Interval          int               `yaml:"interval"`
}

// Ping represents an ICMP ping probe target.
type Ping struct {
	URI string `yaml:"uri"`
}

// Socket represents a TCP socket probe target.
type Socket struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Data string `yaml:"data"`
}

// MongoDB represents a MongoDB health check target.
type MongoDB struct {
	URI      string `yaml:"uri"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Redis represents a Redis health check target.
type Redis struct {
	URI      string `yaml:"uri"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Postgres represents a PostgreSQL health check target.
type Postgres struct {
	URI      string `yaml:"uri"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MariaDB represents a MariaDB/MySQL health check target.
type MariaDB struct {
	URI      string `yaml:"uri"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Alert defines an assertion that triggers a notification.
type Alert struct {
	ID        string                `yaml:"id"`
	Assertion *assertion.Assertion  `yaml:"assertion"`
	Message   string                `yaml:"message"`
}

// Notification represents a notification channel configuration.
type Notification struct {
	ID   string         `yaml:"id"`
	Type string         `yaml:"type"`
	Data map[string]any `yaml:"data"`
}

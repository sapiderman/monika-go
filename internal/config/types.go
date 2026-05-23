package config

// Config represents the top-level monika.yaml configuration.
type Config struct {
	Probes        []Probe        `yaml:"probes"`
	Notifications []Notification `yaml:"notifications"`
}

// Probe represents a single monitoring target.
// Exactly one of Requests, Ping, Socket, Mongo, Redis, Postgres, MariaDB must be set.
type Probe struct {
	ID                string      `yaml:"id"`
	Name              string      `yaml:"name"`
	Description       string      `yaml:"description"`
	Interval          int         `yaml:"interval"`
	IncidentThreshold int         `yaml:"incidentThreshold"`
	RecoveryThreshold int         `yaml:"recoveryThreshold"`
	Alerts            []Alert     `yaml:"alerts"`
	Requests          *[]Request  `yaml:"requests"`
	Ping              *[]Ping     `yaml:"ping"`
	Socket            *[]Socket   `yaml:"socket"`
	Mongo             *[]MongoDB  `yaml:"mongo"`
	Redis             *[]Redis    `yaml:"redis"`
	Postgres          *[]Postgres `yaml:"postgres"`
	MariaDB           *[]MariaDB  `yaml:"mariadb"`
	MySQL             *[]MariaDB  `yaml:"mysql"` // MySQL shares schema with MariaDB
}

// Request represents a single HTTP request within a probe.
type Request struct {
	Method            string            `yaml:"method"`
	URL               string            `yaml:"url"`
	Timeout           int               `yaml:"timeout"`
	SaveBody          bool              `yaml:"saveBody"`
	AllowUnauthorized bool              `yaml:"allowUnauthorized"`
	FollowRedirects   int               `yaml:"followRedirects"`
	Headers           map[string]string `yaml:"headers"`
	Body              any               `yaml:"body"`
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
	ID        string `yaml:"id"`
	Assertion string `yaml:"assertion"`
	Message   string `yaml:"message"`
}

// Notification represents a notification channel configuration.
type Notification struct {
	ID   string         `yaml:"id"`
	Type string         `yaml:"type"`
	Data map[string]any `yaml:"data"`
}

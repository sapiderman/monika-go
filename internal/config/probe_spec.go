package config

import "fmt"

// ProbeKind enumerates the supported probe types.
type ProbeKind string

const (
	KindHTTP     ProbeKind = "http"
	KindPing     ProbeKind = "ping"
	KindSocket   ProbeKind = "socket"
	KindMongoDB  ProbeKind = "mongo"
	KindRedis    ProbeKind = "redis"
	KindPostgres ProbeKind = "postgres"
	KindMariaDB  ProbeKind = "mariadb"
	KindMySQL    ProbeKind = "mysql"
)

// ProbeSpec is the discriminated union for probe type payloads.
// Exactly one concrete type satisfies this interface per Probe.
type ProbeSpec interface {
	Kind() ProbeKind
	Validate() error
}

// --- HTTPSpec ----------------------------------------------------------------

type HTTPSpec struct{ Requests []Request }

func (s *HTTPSpec) Kind() ProbeKind { return KindHTTP }

func (s *HTTPSpec) Validate() error {
	for i, req := range s.Requests {
		if req.URL == "" {
			return fmt.Errorf("request at index %d: url is required", i)
		}
		for j, a := range req.Alerts {
			if a.Assertion == nil {
				return fmt.Errorf("request at index %d: alert at index %d: assertion is required", i, j)
			}
		}
	}
	return nil
}

// --- PingSpec ----------------------------------------------------------------

type PingSpec struct{ Targets []Ping }

func (s *PingSpec) Kind() ProbeKind { return KindPing }

func (s *PingSpec) Validate() error {
	for i, ping := range s.Targets {
		if ping.URI == "" {
			return fmt.Errorf("ping at index %d: uri is required", i)
		}
	}
	return nil
}

// --- SocketSpec --------------------------------------------------------------

type SocketSpec struct{ Targets []Socket }

func (s *SocketSpec) Kind() ProbeKind { return KindSocket }

func (s *SocketSpec) Validate() error {
	for i, socket := range s.Targets {
		if socket.Host == "" || socket.Port <= 0 || socket.Data == "" {
			return fmt.Errorf("socket at index %d: host, port, and data are required", i)
		}
	}
	return nil
}

// --- MongoDBSpec -------------------------------------------------------------

type MongoDBSpec struct{ Targets []MongoDB }

func (s *MongoDBSpec) Kind() ProbeKind     { return KindMongoDB }
func (s *MongoDBSpec) Validate() error     { return nil }

// --- RedisSpec ---------------------------------------------------------------

type RedisSpec struct{ Targets []Redis }

func (s *RedisSpec) Kind() ProbeKind { return KindRedis }
func (s *RedisSpec) Validate() error { return nil }

// --- PostgresSpec ------------------------------------------------------------

type PostgresSpec struct{ Targets []Postgres }

func (s *PostgresSpec) Kind() ProbeKind { return KindPostgres }
func (s *PostgresSpec) Validate() error { return nil }

// --- MariaDBSpec -------------------------------------------------------------

type MariaDBSpec struct{ Targets []MariaDB }

func (s *MariaDBSpec) Kind() ProbeKind { return KindMariaDB }
func (s *MariaDBSpec) Validate() error { return nil }

// --- MySQLSpec ---------------------------------------------------------------

type MySQLSpec struct{ Targets []MariaDB }

func (s *MySQLSpec) Kind() ProbeKind { return KindMySQL }
func (s *MySQLSpec) Validate() error { return nil }
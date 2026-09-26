// Package prober implements probe checks: http, ping, socket, and database.
package prober

import (
	"context"
	"monika-go/internal/assertion"
	"monika-go/internal/config"
)

// Prober represents a single-use or scheduled monitoring execution interface.
type Prober interface {
	Probe(ctx context.Context) ([]RequestResult, error)
}

// RequestResult represents the outcome of executing a single request in the prober's chain.
type RequestResult struct {
	Result       assertion.ProbeResult // outcome from executing the request and evaluating assertions
	AlertPassed  bool                  // true if all request-level alerts passed (or no alerts)
	FailedAlerts []FailedAlert         // list of failed request-level alerts
}

// FailedAlert holds details of an alert that did not pass its assertion.
type FailedAlert struct {
	Assertion string // the original assertion expression
	Message   string // the alert message associated with it
}

// NewProber acts as a factory constructing the concrete prober based on the ProbeSpec kind.
func NewProber(spec config.ProbeSpec) Prober {
	switch spec.Kind() {
	case config.KindHTTP:
		if httpSpec, ok := spec.(*config.HTTPSpec); ok {
			return NewHTTPProber(httpSpec)
		}
	case config.KindPing:
		if pingSpec, ok := spec.(*config.PingSpec); ok {
			return NewPingProber(pingSpec)
		}
	case config.KindSocket:
		if socketSpec, ok := spec.(*config.SocketSpec); ok {
			return NewSocketProber(socketSpec)
		}
	case config.KindMongoDB:
		if mongoSpec, ok := spec.(*config.MongoDBSpec); ok {
			return NewMongoProber(mongoSpec)
		}
	case config.KindRedis:
		if redisSpec, ok := spec.(*config.RedisSpec); ok {
			return NewRedisProber(redisSpec)
		}
	case config.KindPostgres:
		if postgresSpec, ok := spec.(*config.PostgresSpec); ok {
			return NewPostgresProber(postgresSpec)
		}
	case config.KindMariaDB:
		if mariaSpec, ok := spec.(*config.MariaDBSpec); ok {
			return NewMariaDBProber(mariaSpec)
		}
	case config.KindMySQL:
		if mysqlSpec, ok := spec.(*config.MySQLSpec); ok {
			return NewMySQLProber(mysqlSpec)
		}
	default:
		return nil
	}
	return nil
}

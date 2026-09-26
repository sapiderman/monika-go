package prober

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"monika-go/internal/assertion"
	"monika-go/internal/config"
)

// DBProber monitors various relational/NoSQL databases by attempting connection or pinging.
type DBProber struct {
	kind  config.ProbeKind
	mongo *config.MongoDBSpec
	redis *config.RedisSpec
	pg    *config.PostgresSpec
	maria *config.MariaDBSpec
	mysql *config.MySQLSpec
}

// NewMongoProber constructs a DBProber for MongoDB.
func NewMongoProber(spec *config.MongoDBSpec) *DBProber {
	return &DBProber{kind: config.KindMongoDB, mongo: spec}
}

// NewRedisProber constructs a DBProber for Redis.
func NewRedisProber(spec *config.RedisSpec) *DBProber {
	return &DBProber{kind: config.KindRedis, redis: spec}
}

// NewPostgresProber constructs a DBProber for PostgreSQL.
func NewPostgresProber(spec *config.PostgresSpec) *DBProber {
	return &DBProber{kind: config.KindPostgres, pg: spec}
}

// NewMariaDBProber constructs a DBProber for MariaDB.
func NewMariaDBProber(spec *config.MariaDBSpec) *DBProber {
	return &DBProber{kind: config.KindMariaDB, maria: spec}
}

// NewMySQLProber constructs a DBProber for MySQL.
func NewMySQLProber(spec *config.MySQLSpec) *DBProber {
	return &DBProber{kind: config.KindMySQL, mysql: spec}
}

// Probe executes connection/ping checks for the configured database spec.
func (d *DBProber) Probe(ctx context.Context) ([]RequestResult, error) {
	switch d.kind {
	case config.KindMongoDB:
		return d.probeMongoDB(ctx)
	case config.KindRedis:
		return d.probeRedis(ctx)
	case config.KindPostgres:
		return d.probePostgres(ctx)
	case config.KindMariaDB:
		return d.probeMariaDB(ctx)
	case config.KindMySQL:
		return d.probeMySQL(ctx)
	default:
		return nil, fmt.Errorf("unknown DB prober kind: %s", d.kind)
	}
}

func (d *DBProber) probeMongoDB(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(d.mongo.Targets))
	for _, target := range d.mongo.Targets {
		host := target.Host
		port := target.Port
		if host == "" && port == 0 && target.URI != "" {
			host, port = parseHostPortFromURI(target.URI, 27017)
		}
		if host == "" {
			host = "localhost"
		}
		if port <= 0 {
			port = 27017
		}
		res := d.checkTCP(ctx, host, port, "MongoDB")
		results = append(results, res)
	}
	return results, nil
}

func (d *DBProber) probeRedis(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(d.redis.Targets))
	for _, target := range d.redis.Targets {
		host := target.Host
		port := target.Port
		if host == "" && port == 0 && target.URI != "" {
			host, port = parseHostPortFromURI(target.URI, 6379)
		}
		if host == "" {
			host = "localhost"
		}
		if port <= 0 {
			port = 6379
		}
		res := d.checkTCP(ctx, host, port, "Redis")
		results = append(results, res)
	}
	return results, nil
}

func (d *DBProber) probePostgres(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(d.pg.Targets))
	for _, target := range d.pg.Targets {
		host := target.Host
		port := target.Port
		if host == "" && port == 0 && target.URI != "" {
			host, port = parseHostPortFromURI(target.URI, 5432)
		}
		if host == "" {
			host = "localhost"
		}
		if port <= 0 {
			port = 5432
		}
		res := d.checkTCP(ctx, host, port, "Postgres")
		results = append(results, res)
	}
	return results, nil
}

func (d *DBProber) probeMariaDB(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(d.maria.Targets))
	for _, target := range d.maria.Targets {
		host := target.Host
		port := target.Port
		if host == "" && port == 0 && target.URI != "" {
			host, port = parseHostPortFromURI(target.URI, 3306)
		}
		if host == "" {
			host = "localhost"
		}
		if port <= 0 {
			port = 3306
		}
		res := d.checkTCP(ctx, host, port, "MariaDB")
		results = append(results, res)
	}
	return results, nil
}

func (d *DBProber) probeMySQL(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(d.mysql.Targets))
	for _, target := range d.mysql.Targets {
		host := target.Host
		port := target.Port
		if host == "" && port == 0 && target.URI != "" {
			host, port = parseHostPortFromURI(target.URI, 3306)
		}
		if host == "" {
			host = "localhost"
		}
		if port <= 0 {
			port = 3306
		}
		res := d.checkTCP(ctx, host, port, "MySQL")
		results = append(results, res)
	}
	return results, nil
}

func (d *DBProber) checkTCP(ctx context.Context, host string, port int, dbName string) RequestResult {
	address := fmt.Sprintf("%s:%d", host, port)

	dialCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var dialer net.Dialer
	start := time.Now()
	conn, err := dialer.DialContext(dialCtx, "tcp", address)
	duration := max(time.Since(start).Milliseconds(), 1) // floor 1ms: sub-ms success must not report 0

	var probeResult assertion.ProbeResult

	if err != nil {
		if dialCtx.Err() != nil {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("%s connect timeout to %s: %w", dbName, address, ErrTimeout),
			}
		} else {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("%s connect failed to %s: %w", dbName, address, ErrConnection),
			}
		}
	} else {
		conn.Close()
		probeResult = assertion.ProbeResult{
			Status:       0,
			ResponseTime: duration,
		}
	}

	return RequestResult{
		Result:      probeResult,
		AlertPassed: true,
	}
}

func parseHostPortFromURI(uriStr string, defaultPort int) (string, int) {
	if uriStr == "" {
		return "", 0
	}
	// Quick parse host and port from connection strings (e.g. mongodb://user:pass@host:port/db)
	cleaned := uriStr
	for _, prefix := range []string{"mongodb://", "redis://", "postgresql://", "postgres://", "mariadb://", "mysql://"} {
		if strings.HasPrefix(cleaned, prefix) {
			cleaned = cleaned[len(prefix):]
			break
		}
	}

	// Remove credentials if present
	if idx := strings.Index(cleaned, "@"); idx != -1 {
		cleaned = cleaned[idx+1:]
	}

	// Remove database path or query params
	if idx := strings.Index(cleaned, "/"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	if idx := strings.Index(cleaned, "?"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	host, portStr, err := net.SplitHostPort(cleaned)
	if err != nil {
		// Just host without port
		return cleaned, defaultPort
	}

	var port int
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil {
		return host, defaultPort
	}
	return host, port
}

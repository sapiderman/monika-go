package prober

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"monika-go/internal/config"
)

func TestPingProber_Probe(t *testing.T) {
	spec := &config.PingSpec{
		Targets: []config.Ping{
			{URI: "localhost"},
			{URI: "127.0.0.1"},
			{URI: "http://localhost"},
		},
	}
	p := NewPingProber(spec)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	results, err := p.Probe(ctx)
	if err != nil {
		t.Fatalf("Ping probe failed: %v", err)
	}

	if len(results) != len(spec.Targets) {
		t.Errorf("expected %d results, got %d", len(spec.Targets), len(results))
	}

	for _, res := range results {
		if res.Result.Err != nil {
			// Some minimal containers/environments might block ping, but checking if we ran it without panic or setup errors
			t.Logf("Ping target returned execution error (expected on some restricted CI/CD systems): %v", res.Result.Err)
		} else {
			if res.Result.ResponseTime <= 0 {
				t.Error("expected positive response time for successful ping")
			}
		}
	}
}

func TestPingProber_EmptyHost(t *testing.T) {
	spec := &config.PingSpec{
		Targets: []config.Ping{
			{URI: ""},
		},
	}
	p := NewPingProber(spec)

	results, err := p.Probe(context.Background())
	if err == nil {
		t.Error("expected error for empty target URI, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results on empty host failure, got %d", len(results))
	}
}

func TestSocketProber_Probe(t *testing.T) {
	// Start a local TCP listener to test SocketProber
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test TCP listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, errAccept := listener.Accept()
			if errAccept != nil {
				return
			}
			buf := make([]byte, 1024)
			n, errRead := conn.Read(buf)
			if errRead == nil && n > 0 {
				_, _ = conn.Write([]byte("echo: " + string(buf[:n])))
			}
			conn.Close()
		}
	}()

	spec := &config.SocketSpec{
		Targets: []config.Socket{
			{Host: addr.IP.String(), Port: addr.Port, Data: "hello"},
		},
	}
	p := NewSocketProber(spec)

	results, err := p.Probe(context.Background())
	if err != nil {
		t.Fatalf("Socket probe failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if res.Result.Err != nil {
		t.Fatalf("expected successful socket connection, got error: %v", res.Result.Err)
	}

	if !strings.Contains(res.Result.Body, "echo: hello") {
		t.Errorf("expected body to contain echo: hello, got: %q", res.Result.Body)
	}
}

func TestDBProber_Probe(t *testing.T) {
	// Let's verify our database TCP checker without needing actual running databases.
	// We will spin up a TCP listener and configure each db type target to hit that port.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start test TCP listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, errAccept := listener.Accept()
			if errAccept != nil {
				return
			}
			conn.Close()
		}
	}()

	// MongoDB
	mongoSpec := &config.MongoDBSpec{
		Targets: []config.MongoDB{
			{Host: addr.IP.String(), Port: addr.Port},
			{URI: "mongodb://127.0.0.1:" + string(rune(addr.Port))}, // parseHostPortFromURI test fallback
		},
	}
	pMongo := NewMongoProber(mongoSpec)
	resMongo, err := pMongo.Probe(context.Background())
	if err != nil {
		t.Errorf("MongoDB probe failed: %v", err)
	}
	if len(resMongo) != 2 {
		t.Errorf("expected 2 MongoDB results, got %d", len(resMongo))
	}
	if resMongo[0].Result.Err != nil {
		t.Errorf("expected successful connection to TCP listener, got: %v", resMongo[0].Result.Err)
	}

	// Redis
	redisSpec := &config.RedisSpec{
		Targets: []config.Redis{
			{Host: addr.IP.String(), Port: addr.Port},
		},
	}
	pRedis := NewRedisProber(redisSpec)
	resRedis, err := pRedis.Probe(context.Background())
	if err != nil {
		t.Errorf("Redis probe failed: %v", err)
	}
	if resRedis[0].Result.Err != nil {
		t.Errorf("expected successful connection to TCP listener, got: %v", resRedis[0].Result.Err)
	}

	// Postgres
	pgSpec := &config.PostgresSpec{
		Targets: []config.Postgres{
			{Host: addr.IP.String(), Port: addr.Port},
		},
	}
	pPg := NewPostgresProber(pgSpec)
	resPg, err := pPg.Probe(context.Background())
	if err != nil {
		t.Errorf("Postgres probe failed: %v", err)
	}
	if resPg[0].Result.Err != nil {
		t.Errorf("expected successful connection to TCP listener, got: %v", resPg[0].Result.Err)
	}

	// MariaDB & MySQL
	mariaSpec := &config.MariaDBSpec{
		Targets: []config.MariaDB{
			{Host: addr.IP.String(), Port: addr.Port},
		},
	}
	pMaria := NewMariaDBProber(mariaSpec)
	resMaria, err := pMaria.Probe(context.Background())
	if err != nil {
		t.Errorf("MariaDB probe failed: %v", err)
	}
	if resMaria[0].Result.Err != nil {
		t.Errorf("expected successful connection to TCP listener, got: %v", resMaria[0].Result.Err)
	}

	mysqlSpec := &config.MySQLSpec{
		Targets: []config.MariaDB{
			{Host: addr.IP.String(), Port: addr.Port},
		},
	}
	pMySQL := NewMySQLProber(mysqlSpec)
	resMySQL, err := pMySQL.Probe(context.Background())
	if err != nil {
		t.Errorf("MySQL probe failed: %v", err)
	}
	if resMySQL[0].Result.Err != nil {
		t.Errorf("expected successful connection to TCP listener, got: %v", resMySQL[0].Result.Err)
	}
}

func TestParseHostPortFromURI(t *testing.T) {
	tests := []struct {
		uri         string
		defaultPort int
		wantHost    string
		wantPort    int
	}{
		{"mongodb://user:pass@1.2.3.4:27017/db", 27017, "1.2.3.4", 27017},
		{"redis://localhost", 6379, "localhost", 6379},
		{"postgresql://user:pass@localhost:5432/dbname?sslmode=disable", 5432, "localhost", 5432},
		{"redis://:password@redis-server:6380", 6379, "redis-server", 6380},
		{"", 1234, "", 0},
	}

	for _, tt := range tests {
		gotHost, gotPort := parseHostPortFromURI(tt.uri, tt.defaultPort)
		if gotHost != tt.wantHost || gotPort != tt.wantPort {
			t.Errorf("parseHostPortFromURI(%q, %d) = (%q, %d), want (%q, %d)",
				tt.uri, tt.defaultPort, gotHost, gotPort, tt.wantHost, tt.wantPort)
		}
	}
}

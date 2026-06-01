package prober

import (
	"testing"

	"monika-go/internal/config"
)

func TestNewProber_HTTP(t *testing.T) {
	spec := &config.HTTPSpec{
		Requests: []config.Request{{URL: "https://example.com"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for HTTP spec")
	}
}

func TestNewProber_NilSpec(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil spec")
		}
	}()
	NewProber(nil)
}

func TestNewProber_Ping(t *testing.T) {
	spec := &config.PingSpec{
		Targets: []config.Ping{{URI: "https://example.com"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for Ping spec")
	}
}

func TestNewProber_Socket(t *testing.T) {
	spec := &config.SocketSpec{
		Targets: []config.Socket{{Host: "localhost", Port: 8080, Data: "ping"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for Socket spec")
	}
}

func TestNewProber_MongoDB(t *testing.T) {
	spec := &config.MongoDBSpec{
		Targets: []config.MongoDB{{URI: "mongodb://localhost:27017"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for MongoDB spec")
	}
}

func TestNewProber_Redis(t *testing.T) {
	spec := &config.RedisSpec{
		Targets: []config.Redis{{URI: "redis://localhost:6379"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for Redis spec")
	}
}

func TestNewProber_Postgres(t *testing.T) {
	spec := &config.PostgresSpec{
		Targets: []config.Postgres{{URI: "postgres://localhost:5432/db"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for Postgres spec")
	}
}

func TestNewProber_MariaDB(t *testing.T) {
	spec := &config.MariaDBSpec{
		Targets: []config.MariaDB{{URI: "mariadb://localhost:3306/db"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for MariaDB spec")
	}
}

func TestNewProber_MySQL(t *testing.T) {
	spec := &config.MySQLSpec{
		Targets: []config.MariaDB{{URI: "mysql://localhost:3306/db"}},
	}
	p := NewProber(spec)
	if p == nil {
		t.Fatal("NewProber returned nil for MySQL spec")
	}
}

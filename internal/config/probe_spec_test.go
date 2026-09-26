package config

import "testing"

func TestSpecKind(t *testing.T) {
	tests := []struct {
		name string
		spec ProbeSpec
		want ProbeKind
	}{
		{"http", &HTTPSpec{}, KindHTTP},
		{"ping", &PingSpec{}, KindPing},
		{"socket", &SocketSpec{}, KindSocket},
		{"mongo", &MongoDBSpec{}, KindMongoDB},
		{"redis", &RedisSpec{}, KindRedis},
		{"postgres", &PostgresSpec{}, KindPostgres},
		{"mariadb", &MariaDBSpec{}, KindMariaDB},
		{"mysql", &MySQLSpec{}, KindMySQL},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.spec.Kind(); got != tt.want {
				t.Errorf("Kind() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSpecValidate(t *testing.T) {
	tests := []struct {
		name    string
		spec    ProbeSpec
		wantErr bool
	}{
		{"http ok", &HTTPSpec{Requests: []Request{{URL: "https://example.com"}}}, false},
		{"http missing url", &HTTPSpec{Requests: []Request{{}}}, true},
		{"http alert without assertion", &HTTPSpec{Requests: []Request{{URL: "u", Alerts: []Alert{{}}}}}, true},
		// DB specs validate at connect time; empty targets are accepted.
		{"mongo", &MongoDBSpec{}, false},
		{"redis", &RedisSpec{Targets: []Redis{{}}}, false},
		{"postgres", &PostgresSpec{}, false},
		{"mariadb", &MariaDBSpec{}, false},
		{"mysql", &MySQLSpec{Targets: []MariaDB{{}}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.spec.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

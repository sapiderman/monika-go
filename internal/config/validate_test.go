package config

import (
	"strings"
	"testing"
)

func ptrSlice[T any](s ...T) *[]T { return &s }

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name: "valid config",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "https://example.com"}}}},
				},
			},
			wantErr: "",
		},
		{
			name: "valid config with ping probe",
			config: Config{
				Probes: []Probe{
					{ID: "p1", Spec: &PingSpec{Targets: []Ping{{URI: "https://example.com"}}}},
				},
			},
			wantErr: "",
		},
		{
			name: "valid config with socket probe",
			config: Config{
				Probes: []Probe{
					{ID: "s1", Spec: &SocketSpec{Targets: []Socket{{Host: "localhost", Port: 8080, Data: "ping"}}}},
				},
			},
			wantErr: "",
		},
		{
			name: "valid config with database probes",
			config: Config{
				Probes: []Probe{
					{ID: "m1", Spec: &MongoDBSpec{Targets: []MongoDB{{URI: "mongodb://localhost:27017"}}}},
					{ID: "r1", Spec: &RedisSpec{Targets: []Redis{{URI: "redis://localhost:6379"}}}},
					{ID: "pg1", Spec: &PostgresSpec{Targets: []Postgres{{URI: "postgres://localhost:5432/db"}}}},
					{ID: "maria1", Spec: &MariaDBSpec{Targets: []MariaDB{{URI: "mariadb://localhost:3306/db"}}}},
				},
			},
			wantErr: "",
		},
		{
			name: "multiple probes with multiple requests",
			config: Config{
				Probes: []Probe{
					{
						ID:   "multi-1",
						Spec: &HTTPSpec{Requests: []Request{{URL: "https://example.com/api"}, {URL: "https://example.com/health"}}},
					},
					{
						ID:   "multi-2",
						Spec: &PingSpec{Targets: []Ping{{URI: "https://example.com"}}},
					},
				},
			},
			wantErr: "",
		},
		{
			name:    "empty config",
			config:  Config{},
			wantErr: "config must define at least one probe",
		},
		{
			name: "missing probe id",
			config: Config{
				Probes: []Probe{
					{Spec: &HTTPSpec{Requests: []Request{{URL: "https://example.com"}}}},
				},
			},
			wantErr: "probe at index 0: id is required",
		},
		{
			name: "duplicate probe id",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "https://example.com"}}}},
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "https://example.com"}}}},
				},
			},
			wantErr: `duplicate probe id: "1"`,
		},
		{
			name: "no probe type",
			config: Config{
				Probes: []Probe{
					{ID: "1"},
				},
			},
			wantErr: `probe "1": must specify exactly one probe type`,
		},
		{
			name: "missing request url",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{Method: "GET"}}}},
				},
			},
			wantErr: `probe "1": request at index 0: url is required`,
		},
		{
			name: "missing ping uri",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &PingSpec{Targets: []Ping{{}}}},
				},
			},
			wantErr: `probe "1": ping at index 0: uri is required`,
		},
		{
			name: "invalid socket",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &SocketSpec{Targets: []Socket{{Host: "localhost"}}}},
				},
			},
			wantErr: `probe "1": socket at index 0: host, port, and data are required`,
		},
		{
			name: "missing probe alert assertion",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "http://a.com"}}}, Alerts: []Alert{{Message: "bad"}}},
				},
			},
			wantErr: `probe "1": alert at index 0: assertion is required`,
		},
		{
			name: "missing request alert assertion",
			config: Config{
				Probes: []Probe{
					{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "http://a.com", Alerts: []Alert{{Message: "bad"}}}}}},
				},
			},
			wantErr: `probe "1": request at index 0: alert at index 0: assertion is required`,
		},
		{
			name: "unknown notification type",
			config: Config{
				Probes:        []Probe{{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "http://a.com"}}}}},
				Notifications: []Notification{{ID: "n1", Type: "carrier_pigeon"}},
			},
			wantErr: `notification "n1": unknown type "carrier_pigeon"`,
		},
		{
			name: "notification with empty id",
			config: Config{
				Probes:        []Probe{{ID: "1", Spec: &HTTPSpec{Requests: []Request{{URL: "http://a.com"}}}}},
				Notifications: []Notification{{Type: "slack"}},
			},
			wantErr: "notification at index 0: id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&tt.config)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}
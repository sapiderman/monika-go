package config

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML decodes the flat YAML wire format into the ProbeSpec discriminated union.
func (p *Probe) UnmarshalYAML(unmarshal func(any) error) error {
	var wire struct {
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
		MySQL             *[]MariaDB  `yaml:"mysql"`
	}
	if err := unmarshal(&wire); err != nil {
		return err
	}

	p.ID = wire.ID
	p.Name = wire.Name
	p.Description = wire.Description
	p.Interval = wire.Interval
	p.IncidentThreshold = wire.IncidentThreshold
	p.RecoveryThreshold = wire.RecoveryThreshold
	p.Alerts = wire.Alerts

	var set int
	if wire.Requests != nil {
		p.Spec = &HTTPSpec{Requests: *wire.Requests}
		set++
	}
	if wire.Ping != nil {
		p.Spec = &PingSpec{Targets: *wire.Ping}
		set++
	}
	if wire.Socket != nil {
		p.Spec = &SocketSpec{Targets: *wire.Socket}
		set++
	}
	if wire.Mongo != nil {
		p.Spec = &MongoDBSpec{Targets: *wire.Mongo}
		set++
	}
	if wire.Redis != nil {
		p.Spec = &RedisSpec{Targets: *wire.Redis}
		set++
	}
	if wire.Postgres != nil {
		p.Spec = &PostgresSpec{Targets: *wire.Postgres}
		set++
	}
	if wire.MariaDB != nil {
		p.Spec = &MariaDBSpec{Targets: *wire.MariaDB}
		set++
	}
	if wire.MySQL != nil {
		p.Spec = &MySQLSpec{Targets: *wire.MySQL}
		set++
	}

	switch {
	case set == 0:
		return fmt.Errorf("probe %q: must specify exactly one probe type", wire.ID)
	case set > 1:
		return fmt.Errorf("probe %q: must specify exactly one probe type, found %d", wire.ID, set)
	}
	return nil
}

// UnmarshalYAML ensures HTTP request body is a string or mapping.
func (b *RequestBody) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		b.text = value.Value
	case yaml.MappingNode:
		return value.Decode(&b.form)
	default:
		return fmt.Errorf("body must be a string or mapping, got %s", value.Tag)
	}
	return nil
}

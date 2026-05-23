package config

import (
	"fmt"
)

var knownNotificationTypes = map[string]bool{
	"smtp":         true,
	"slack":        true,
	"webhook":      true,
	"telegram":     true,
	"discord":      true,
	"teams":        true,
	"lark":         true,
	"mailgun":      true,
	"sendgrid":     true,
	"instatus":     true,
	"opsgenie":     true,
	"pushover":     true,
	"workplace":    true,
	"dingtalk":     true,
	"monika-notif": true,
	"whatsapp":     true,
	"desktop":      true, // native OS notification
}

// Validate checks a Config for semantic errors.
// Returns the first error encountered, or nil if valid.
func Validate(cfg *Config) error {
	if len(cfg.Probes) == 0 {
		return fmt.Errorf("config must define at least one probe")
	}

	seenIDs := make(map[string]bool)

	for i, p := range cfg.Probes {
		if p.ID == "" {
			return fmt.Errorf("probe at index %d: id is required", i)
		}
		if seenIDs[p.ID] {
			return fmt.Errorf("duplicate probe id: %q", p.ID)
		}
		seenIDs[p.ID] = true

		typeCount := probeTypeCount(&p)
		if typeCount != 1 {
			return fmt.Errorf("probe %q: must have exactly one probe type, got %d", p.ID, typeCount)
		}

		for j, a := range p.Alerts {
			if a.Assertion == "" {
				return fmt.Errorf("probe %q: alert at index %d: assertion is required", p.ID, j)
			}
		}

		if p.Requests != nil {
			for j, req := range *p.Requests {
				if req.URL == "" {
					return fmt.Errorf("probe %q: request at index %d: url is required", p.ID, j)
				}
				for k, a := range req.Alerts {
					if a.Assertion == "" {
						return fmt.Errorf("probe %q: request at index %d: alert at index %d: assertion is required", p.ID, j, k)
					}
				}
			}
		}

		if p.Ping != nil {
			for j, ping := range *p.Ping {
				if ping.URI == "" {
					return fmt.Errorf("probe %q: ping at index %d: uri is required", p.ID, j)
				}
			}
		}

		if p.Socket != nil {
			for j, socket := range *p.Socket {
				if socket.Host == "" || socket.Port <= 0 || socket.Data == "" {
					return fmt.Errorf("probe %q: socket at index %d: host, port, and data are required", p.ID, j)
				}
			}
		}
	}

	for i, n := range cfg.Notifications {
		if n.ID == "" {
			return fmt.Errorf("notification at index %d: id is required", i)
		}
		if !knownNotificationTypes[n.Type] {
			return fmt.Errorf("notification %q: unknown type %q", n.ID, n.Type)
		}
	}

	return nil
}

// probeTypeCount counts how many probe type fields are set.
func probeTypeCount(p *Probe) int {
	count := 0
	if p.Requests != nil {
		count++
	}
	if p.Ping != nil {
		count++
	}
	if p.Socket != nil {
		count++
	}
	if p.Mongo != nil {
		count++
	}
	if p.Redis != nil {
		count++
	}
	if p.Postgres != nil {
		count++
	}
	if p.MariaDB != nil {
		count++
	}
	if p.MySQL != nil {
		count++
	}
	return count
}

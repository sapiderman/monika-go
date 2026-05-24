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

		if p.Spec == nil {
			return fmt.Errorf("probe %q: must specify exactly one probe type", p.ID)
		}

		for j, a := range p.Alerts {
			if a.Assertion == nil {
				return fmt.Errorf("probe %q: alert at index %d: assertion is required", p.ID, j)
			}
		}

		if err := p.Spec.Validate(); err != nil {
			return fmt.Errorf("probe %q: %w", p.ID, err)
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
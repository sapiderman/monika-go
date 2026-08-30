package config

import (
	"fmt"
)

// Only types implemented by notification.NewNotifier belong here; add a type
// when the factory implements it (reference Monika supports more channels).
var knownNotificationTypes = map[string]bool{
	"smtp":    true,
	"slack":   true,
	"webhook": true,
	"desktop": true, // native OS notification
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
		if err := validateNotificationData(n); err != nil {
			return err
		}
	}

	return nil
}

// validateNotificationData checks required Data fields per notification type.
// Presence/non-empty only — the notifier constructors do the deep parsing.
func validateNotificationData(n Notification) error {
	switch n.Type {
	case "smtp":
		switch recs := n.Data["recipients"].(type) {
		case []any:
			if len(recs) == 0 {
				return fmt.Errorf("notification %q: smtp requires a non-empty \"recipients\" list", n.ID)
			}
		case []string:
			if len(recs) == 0 {
				return fmt.Errorf("notification %q: smtp requires a non-empty \"recipients\" list", n.ID)
			}
		default:
			return fmt.Errorf("notification %q: smtp requires a non-empty \"recipients\" list", n.ID)
		}
		if host, ok := n.Data["hostname"].(string); !ok || host == "" {
			return fmt.Errorf("notification %q: smtp requires a non-empty \"hostname\" string", n.ID)
		}
	case "webhook":
		if url, ok := n.Data["url"].(string); !ok || url == "" {
			return fmt.Errorf("notification %q: webhook requires a non-empty \"url\" string", n.ID)
		}
	case "slack":
		if url, ok := n.Data["url"].(string); !ok || url == "" {
			return fmt.Errorf("notification %q: slack requires a non-empty \"url\" string", n.ID)
		}
	}
	return nil
}

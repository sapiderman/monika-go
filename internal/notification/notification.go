package notification

import (
	"context"
	"fmt"

	"monika-go/internal/alert"
	"monika-go/internal/config"
	"monika-go/internal/logger"
)

// Notifier represents a channel capable of dispatching state transition alerts.
type Notifier interface {
	// ID returns the unique identifier for this notifier instance.
	ID() string
	// Type returns the kind of notification channel (e.g. "desktop", "smtp", "webhook").
	Type() string
	// Notify dispatches the state transition alert to the concrete destination.
	Notify(ctx context.Context, event alert.TransitionEvent) error
}

// NewNotifier acts as a factory constructing concrete Notifier channels from configuration.
func NewNotifier(cfg config.Notification, log logger.Logger) (Notifier, error) {
	switch cfg.Type {
	case "desktop":
		return NewDesktopNotifier(cfg.ID, log), nil
	case "smtp":
		return NewSMTPNotifier(cfg.ID, cfg.Data, log)
	case "webhook":
		return NewWebhookNotifier(cfg.ID, cfg.Data, log)
	case "slack":
		return NewSlackNotifier(cfg.ID, cfg.Data, log)
	default:
		return nil, fmt.Errorf("unsupported notification type: %q", cfg.Type)
	}
}

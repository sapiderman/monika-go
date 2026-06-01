package notification

import (
	"context"
	"os/exec"
	"runtime"

	"monika-go/internal/alert"
	"monika-go/internal/logger"
)

type desktopNotifier struct {
	id  string
	log logger.Logger
}

// NewDesktopNotifier constructs a native OS desktop notifier instance.
func NewDesktopNotifier(id string, log logger.Logger) Notifier {
	return &desktopNotifier{
		id:  id,
		log: log.With(logger.Component("desktop-notifier")),
	}
}

func (d *desktopNotifier) ID() string   { return d.id }
func (d *desktopNotifier) Type() string { return "desktop" }

// Notify displays a native Linux banner utilizing notify-send. Thread-safe.
func (d *desktopNotifier) Notify(ctx context.Context, event alert.TransitionEvent) error {
	d.log.Info("dispatching desktop notification", logger.F("probe_id", event.ProbeID))

	if runtime.GOOS != "linux" {
		d.log.Warn("desktop notifications are only natively supported on Linux", logger.F("os", runtime.GOOS))
		return nil
	}

	urgency := "normal"
	if event.ToState == alert.StateIncident {
		urgency = "critical"
	}

	title := "Monika-Go Alert: " + event.ProbeID
	msg := event.Message

	cmd := exec.CommandContext(ctx, "notify-send", "-u", urgency, title, msg)
	if err := cmd.Run(); err != nil {
		d.log.Warn("failed to execute notify-send (make sure libnotify-bin is installed)", logger.Err(err))
		return err
	}

	return nil
}

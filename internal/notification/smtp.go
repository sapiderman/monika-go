package notification

import (
	"context"
	"fmt"
	"net/smtp"

	"monika-go/internal/alert"
	"monika-go/internal/logger"
)

type smtpNotifier struct {
	id         string
	recipients []string
	hostname   string
	port       int
	username   string
	password   string
	html       bool
	log        logger.Logger
}

// NewSMTPNotifier constructs an SMTP notifier instance parsing generic YAML maps.
func NewSMTPNotifier(id string, data map[string]any, log logger.Logger) (Notifier, error) {
	n := &smtpNotifier{
		id:  id,
		log: log.With(logger.Component("smtp-notifier")),
	}

	if recs, ok := data["recipients"]; ok {
		if recList, ok := recs.([]any); ok {
			for _, r := range recList {
				if rStr, ok := r.(string); ok {
					n.recipients = append(n.recipients, rStr)
				}
			}
		} else if recListStr, ok := recs.([]string); ok {
			n.recipients = recListStr
		}
	}

	if len(n.recipients) == 0 {
		return nil, fmt.Errorf("smtp notifier %q: recipients must not be empty", id)
	}

	if host, ok := data["hostname"].(string); ok {
		n.hostname = host
	} else {
		return nil, fmt.Errorf("smtp notifier %q: hostname must be a string", id)
	}

	if portVal, ok := data["port"]; ok {
		switch v := portVal.(type) {
		case int:
			n.port = v
		case float64:
			n.port = int(v)
		default:
			return nil, fmt.Errorf("smtp notifier %q: invalid port type", id)
		}
	} else {
		n.port = 587
	}

	if user, ok := data["username"].(string); ok {
		n.username = user
	}
	if pass, ok := data["password"].(string); ok {
		n.password = pass
	}
	if html, ok := data["html"].(bool); ok {
		n.html = html
	}

	return n, nil
}

func (s *smtpNotifier) ID() string   { return s.id }
func (s *smtpNotifier) Type() string { return "smtp" }

// Notify dispatches email messages using net/smtp. Safe with contexts and timeouts.
func (s *smtpNotifier) Notify(ctx context.Context, event alert.TransitionEvent) error {
	s.log.Info("dispatching SMTP email alert", logger.F("probe_id", event.ProbeID), logger.F("recipients", len(s.recipients)))

	subject := fmt.Sprintf("Subject: Monika-Go Alert: %s (%s)\r\n", event.ProbeID, event.ToState)
	contentType := "text/plain"
	if s.html {
		contentType = "text/html"
	}
	mime := fmt.Sprintf("MIME-version: 1.0;\r\nContent-Type: %s; charset=\"UTF-8\";\r\n\r\n", contentType)
	body := fmt.Sprintf("Probe Status Changed!\r\n\r\nProbe ID: %s\r\nFrom State: %s\r\nTo State: %s\r\nMessage: %s\r\n",
		event.ProbeID, event.FromState, event.ToState, event.Message)

	msg := []byte(subject + mime + body)
	addr := fmt.Sprintf("%s:%d", s.hostname, s.port)

	errChan := make(chan error, 1)
	go func() {
		var auth smtp.Auth
		if s.username != "" {
			auth = smtp.PlainAuth("", s.username, s.password, s.hostname)
		}
		errChan <- smtp.SendMail(addr, auth, s.username, s.recipients, msg)
	}()

	// ponytail: ctx timeout is best-effort — net/smtp ignores context; upgrade to net.Dialer + smtp.NewClient if SMTP hangs in practice.
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			s.log.Error("failed to send SMTP email", logger.Err(err))
			return err
		}
	}

	return nil
}

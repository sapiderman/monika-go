package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"monika-go/internal/alert"
	"monika-go/internal/logger"
)

type slackNotifier struct {
	id  string
	url string
	log logger.Logger
}

// NewSlackNotifier constructs a Slack incoming-webhook notifier parsing generic YAML maps.
func NewSlackNotifier(id string, data map[string]any, log logger.Logger) (Notifier, error) {
	n := &slackNotifier{
		id:  id,
		log: log.With(logger.Component("slack-notifier")),
	}

	if urlVal, ok := data["url"].(string); ok && urlVal != "" {
		n.url = urlVal
	} else {
		return nil, fmt.Errorf("slack notifier %q: url must be a non-empty string", id)
	}

	return n, nil
}

func (s *slackNotifier) ID() string   { return s.id }
func (s *slackNotifier) Type() string { return "slack" }

// Notify posts the transition message as JSON to the configured Slack incoming webhook. Safe with contexts.
func (s *slackNotifier) Notify(ctx context.Context, event alert.TransitionEvent) error {
	s.log.Info("dispatching slack notification", logger.F("probe_id", event.ProbeID))

	payload := struct {
		Text string `json:"text"`
	}{
		Text: fmt.Sprintf("monika: probe %s is %s: %s", event.ProbeID, string(event.ToState), event.Message),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.log.Error("slack delivery failed", logger.Err(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errStatus := fmt.Errorf("slack responded with status: %d", resp.StatusCode)
		s.log.Error("slack delivery failed", logger.Err(errStatus))
		return errStatus
	}

	return nil
}

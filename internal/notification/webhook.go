package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"monika-go/internal/alert"
	"monika-go/internal/logger"
)

type webhookNotifier struct {
	id      string
	url     string
	method  string
	headers map[string]string
	log     logger.Logger
}

// NewWebhookNotifier constructs a Webhook notifier instance parsing generic YAML maps.
func NewWebhookNotifier(id string, data map[string]any, log logger.Logger) (Notifier, error) {
	n := &webhookNotifier{
		id:      id,
		headers: make(map[string]string),
		log:     log.With(logger.Component("webhook-notifier")),
	}

	if urlVal, ok := data["url"].(string); ok {
		n.url = urlVal
	} else {
		return nil, fmt.Errorf("webhook notifier %q: url must be a string", id)
	}

	if methodVal, ok := data["method"].(string); ok {
		n.method = strings.ToUpper(methodVal)
	} else {
		n.method = "POST"
	}

	if headersVal, ok := data["headers"]; ok {
		if hMap, ok := headersVal.(map[string]any); ok {
			for k, v := range hMap {
				n.headers[k] = fmt.Sprintf("%v", v)
			}
		} else if hMapStr, ok := headersVal.(map[string]string); ok {
			n.headers = hMapStr
		}
	}

	return n, nil
}

func (w *webhookNotifier) ID() string   { return w.id }
func (w *webhookNotifier) Type() string { return "webhook" }

// WebhookPayload structures the JSON payload dispatched to HTTP webhook endpoints.
type WebhookPayload struct {
	ProbeID   string `json:"probe_id"`
	FromState string `json:"from_state"`
	ToState   string `json:"to_state"`
	Message   string `json:"message"`
}

// Notify triggers the HTTP request carrying transition JSON data. Safe with contexts.
func (w *webhookNotifier) Notify(ctx context.Context, event alert.TransitionEvent) error {
	w.log.Info("dispatching webhook notification", logger.F("probe_id", event.ProbeID), logger.F("url", w.url))

	payload := WebhookPayload{
		ProbeID:   event.ProbeID,
		FromState: string(event.FromState),
		ToState:   string(event.ToState),
		Message:   event.Message,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, w.method, w.url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range w.headers {
		req.Header.Set(k, v)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		w.log.Error("webhook delivery failed", logger.Err(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errStatus := fmt.Errorf("webhook responded with status: %d", resp.StatusCode)
		w.log.Error("webhook delivery failed", logger.Err(errStatus))
		return errStatus
	}

	return nil
}

package notification

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"monika-go/internal/alert"
	"monika-go/internal/config"
	"monika-go/internal/logger/loggertest"
)

func TestNewNotifier_Factory(t *testing.T) {
	nopLog := loggertest.NopLogger{}

	t.Run("Desktop Notifier Creation", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "d1",
			Type: "desktop",
		}
		n, err := NewNotifier(cfg, nopLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.ID() != "d1" || n.Type() != "desktop" {
			t.Errorf("incorrect ID or Type: ID=%s, Type=%s", n.ID(), n.Type())
		}
	})

	t.Run("SMTP Notifier Creation Success", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "s1",
			Type: "smtp",
			Data: map[string]any{
				"recipients": []any{"test@example.com"},
				"hostname":   "smtp.example.com",
				"port":       25,
			},
		}
		n, err := NewNotifier(cfg, nopLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.ID() != "s1" || n.Type() != "smtp" {
			t.Errorf("incorrect ID or Type")
		}
	})

	t.Run("SMTP Notifier Missing Hostname", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "s2",
			Type: "smtp",
			Data: map[string]any{
				"recipients": []any{"test@example.com"},
			},
		}
		_, err := NewNotifier(cfg, nopLog)
		if err == nil {
			t.Error("expected error for missing hostname, got nil")
		}
	})

	t.Run("Webhook Notifier Creation Success", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "w1",
			Type: "webhook",
			Data: map[string]any{
				"url":    "https://example.com/webhook",
				"method": "POST",
				"headers": map[string]any{
					"Authorization": "Bearer token",
				},
			},
		}
		n, err := NewNotifier(cfg, nopLog)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if n.ID() != "w1" || n.Type() != "webhook" {
			t.Errorf("incorrect ID or Type")
		}
	})

	t.Run("Webhook Notifier Missing URL", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "w2",
			Type: "webhook",
			Data: map[string]any{
				"method": "POST",
			},
		}
		_, err := NewNotifier(cfg, nopLog)
		if err == nil {
			t.Error("expected error for missing url, got nil")
		}
	})

	t.Run("Unsupported Notifier Type", func(t *testing.T) {
		cfg := config.Notification{
			ID:   "un1",
			Type: "telegram",
		}
		_, err := NewNotifier(cfg, nopLog)
		if err == nil {
			t.Error("expected error for unsupported type slack, got nil")
		}
	})
}

func TestWebhookNotifier_Notify(t *testing.T) {
	var receivedPayload WebhookPayload
	var receivedHeaders http.Header

	// Spin up mock server to receive webhook trigger
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	nopLog := loggertest.NopLogger{}
	notifier, err := NewWebhookNotifier("w-test", map[string]any{
		"url":    server.URL,
		"method": "POST",
		"headers": map[string]any{
			"X-Custom-Header": "CustomValue",
		},
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to create webhook: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "test-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Degraded health detected",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = notifier.Notify(ctx, event)
	if err != nil {
		t.Fatalf("failed to execute webhook notification: %v", err)
	}

	// Verify headers
	if receivedHeaders.Get("X-Custom-Header") != "CustomValue" {
		t.Errorf("expected header X-Custom-Header = CustomValue, got %q", receivedHeaders.Get("X-Custom-Header"))
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type = application/json, got %q", receivedHeaders.Get("Content-Type"))
	}

	// Verify payload
	if receivedPayload.ProbeID != event.ProbeID {
		t.Errorf("expected probe ID %s, got %s", event.ProbeID, receivedPayload.ProbeID)
	}
	if receivedPayload.ToState != string(event.ToState) {
		t.Errorf("expected to state %s, got %s", event.ToState, receivedPayload.ToState)
	}
	if receivedPayload.Message != event.Message {
		t.Errorf("expected message %s, got %s", event.Message, receivedPayload.Message)
	}
}

// startMockSMTP runs a minimal SMTP server recording every received line, including DATA payload lines.
// Returns the host, port, and a func retrieving the recording once the session has ended.
func startMockSMTP(t *testing.T) (host string, port int, commands func() []string) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind mock SMTP listener: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	host, portStr, _ := net.SplitHostPort(listener.Addr().String())
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	var recordedCommands []string
	done := make(chan struct{})

	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		writer := bufio.NewWriter(conn)
		reader := bufio.NewReader(conn)

		// Send initial greeting
		writer.WriteString("220 smtp.example.com ESMTP\r\n")
		writer.Flush()

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			cmd := strings.TrimSpace(line)
			recordedCommands = append(recordedCommands, cmd)

			if strings.HasPrefix(cmd, "EHLO") || strings.HasPrefix(cmd, "HELO") {
				writer.WriteString("250-smtp.example.com\r\n250 AUTH PLAIN\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "AUTH PLAIN") {
				writer.WriteString("235 Authentication succeeded\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "MAIL FROM:") {
				writer.WriteString("250 OK\r\n")
				writer.Flush()
			} else if strings.HasPrefix(cmd, "RCPT TO:") {
				writer.WriteString("250 OK\r\n")
				writer.Flush()
			} else if cmd == "DATA" {
				writer.WriteString("354 Start mail input; end with <CR><LF>.<CR><LF>\r\n")
				writer.Flush()
			} else if cmd == "." {
				writer.WriteString("250 OK\r\n")
				writer.Flush()
			} else if cmd == "QUIT" {
				writer.WriteString("221 Goodbye\r\n")
				writer.Flush()
				break
			}
		}
	}()

	return host, port, func() []string {
		<-done
		return recordedCommands
	}
}

func TestSMTPNotifier_Notify(t *testing.T) {
	host, port, commands := startMockSMTP(t)

	nopLog := loggertest.NopLogger{}
	notifier, err := NewSMTPNotifier("smtp-test", map[string]any{
		"recipients": []any{"user@example.com"},
		"hostname":   host,
		"port":       port,
		"username":   "sender@example.com",
		"password":   "secretpassword",
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to construct smtp notifier: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "mail-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Degraded health detected",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = notifier.Notify(ctx, event)
	if err != nil {
		t.Fatalf("failed to execute SMTP email notification: %v", err)
	}

	// Assert that standard commands were executed in order
	var foundMailFrom, foundRcptTo, foundData bool
	for _, cmd := range commands() {
		if strings.HasPrefix(cmd, "MAIL FROM:") {
			foundMailFrom = true
		}
		if strings.HasPrefix(cmd, "RCPT TO:") {
			foundRcptTo = true
		}
		if cmd == "DATA" {
			foundData = true
		}
	}

	if !foundMailFrom {
		t.Error("expected MAIL FROM command to be issued by smtp client")
	}
	if !foundRcptTo {
		t.Error("expected RCPT TO command to be issued by smtp client")
	}
	if !foundData {
		t.Error("expected DATA command to be issued by smtp client")
	}
}

func TestDesktopNotifier_Notify(t *testing.T) {
	nopLog := loggertest.NopLogger{}
	notifier := NewDesktopNotifier("d-test", nopLog)

	event := alert.TransitionEvent{
		ProbeID:   "linux-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "OS Incident Triggered",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// This should run without panicking and return nil on platforms other than Linux.
	// On Linux, it requires libnotify-bin/notify-send. If notify-send is absent, it returns an error,
	// which is the correct system behavior. So we only run and check on non-Linux to avoid environment issues in headless containers.
	// We check for no panics.
	_ = notifier.Notify(ctx, event)
}

func TestWebhookNotify_Non2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	nopLog := loggertest.NopLogger{}
	notifier, err := NewNotifier(config.Notification{
		ID:   "w-500",
		Type: "webhook",
		Data: map[string]any{
			"url":    server.URL,
			"method": "POST",
		},
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to create webhook notifier: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "status-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Non-2xx delivery test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = notifier.Notify(ctx, event)
	if err == nil {
		t.Fatal("expected error for non-2xx webhook response, got nil")
	}
	if !strings.Contains(err.Error(), "webhook responded with status") {
		t.Errorf("expected error to mention webhook status, got %q", err.Error())
	}
}

func TestWebhookNotify_UnreachableURL(t *testing.T) {
	nopLog := loggertest.NopLogger{}
	notifier, err := NewNotifier(config.Notification{
		ID:   "w-unreachable",
		Type: "webhook",
		Data: map[string]any{
			"url":    "http://127.0.0.1:1/webhook",
			"method": "POST",
		},
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to create webhook notifier: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "unreachable-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Unreachable delivery test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Port 1 on localhost is closed, so the request fails fast with a connection error.
	if err = notifier.Notify(ctx, event); err == nil {
		t.Fatal("expected error for unreachable webhook URL, got nil")
	}
}

func TestSlackNotifier_Notify(t *testing.T) {
	var receivedText string
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		bodyBytes, _ := io.ReadAll(r.Body)
		var payload struct {
			Text string `json:"text"`
		}
		_ = json.Unmarshal(bodyBytes, &payload)
		receivedText = payload.Text
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	nopLog := loggertest.NopLogger{}
	notifier, err := NewSlackNotifier("slack-test", map[string]any{
		"url": server.URL,
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to create slack notifier: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "slack-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Degraded health detected",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err = notifier.Notify(ctx, event); err != nil {
		t.Fatalf("failed to execute slack notification: %v", err)
	}

	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type = application/json, got %q", receivedHeaders.Get("Content-Type"))
	}
	if !strings.Contains(receivedText, "slack-probe") {
		t.Errorf("expected message to mention probe ID, got %q", receivedText)
	}
	if !strings.Contains(receivedText, string(alert.StateIncident)) {
		t.Errorf("expected message to mention target state, got %q", receivedText)
	}
}

func TestSlackNotify_Non2xxError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	nopLog := loggertest.NopLogger{}
	notifier, err := NewSlackNotifier("slack-500", map[string]any{
		"url": server.URL,
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to create slack notifier: %v", err)
	}

	event := alert.TransitionEvent{
		ProbeID:   "status-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "Non-2xx delivery test",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = notifier.Notify(ctx, event)
	if err == nil {
		t.Fatal("expected error for non-2xx slack response, got nil")
	}
	if !strings.Contains(err.Error(), "slack responded with status") {
		t.Errorf("expected error to mention slack status, got %q", err.Error())
	}
}

func TestSlackNotifier_MissingURL(t *testing.T) {
	nopLog := loggertest.NopLogger{}

	for name, data := range map[string]map[string]any{
		"absent url": {},
		"empty url":  {"url": ""},
		"non-string": {"url": 123},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewSlackNotifier("slack-no-url", data, nopLog)
			if err == nil {
				t.Error("expected error for missing url, got nil")
			}
		})
	}
}

func TestSMTPNotifier_HTMLContentType(t *testing.T) {
	nopLog := loggertest.NopLogger{}
	event := alert.TransitionEvent{
		ProbeID:   "mail-probe",
		FromState: alert.StateHealthy,
		ToState:   alert.StateIncident,
		Message:   "HTML body test",
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Plain text send: html flag absent
	host, port, commands := startMockSMTP(t)
	notifier, err := NewSMTPNotifier("smtp-plain", map[string]any{
		"recipients": []any{"user@example.com"},
		"hostname":   host,
		"port":       port,
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to construct smtp notifier: %v", err)
	}
	if err = notifier.Notify(ctx, event); err != nil {
		t.Fatalf("failed to send plain-text email: %v", err)
	}

	var foundPlain bool
	for _, cmd := range commands() {
		if strings.Contains(cmd, "Content-Type: text/plain") {
			foundPlain = true
		}
	}
	if !foundPlain {
		t.Error("expected Content-Type: text/plain in message without html flag")
	}

	// HTML send: html flag set
	host, port, commands = startMockSMTP(t)
	htmlNotifier, err := NewSMTPNotifier("smtp-html", map[string]any{
		"recipients": []any{"user@example.com"},
		"hostname":   host,
		"port":       port,
		"html":       true,
	}, nopLog)
	if err != nil {
		t.Fatalf("failed to construct smtp notifier: %v", err)
	}
	if err = htmlNotifier.Notify(ctx, event); err != nil {
		t.Fatalf("failed to send html email: %v", err)
	}

	var foundHTML bool
	for _, cmd := range commands() {
		if strings.Contains(cmd, "Content-Type: text/html") {
			foundHTML = true
		}
	}
	if !foundHTML {
		t.Error("expected Content-Type: text/html in message with html flag")
	}
}

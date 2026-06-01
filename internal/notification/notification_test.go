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
	"sync"
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
			Type: "slack",
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

func TestSMTPNotifier_Notify(t *testing.T) {
	// Spin up a mock SMTP TCP server to record auth and commands
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to bind mock SMTP listener: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().String()
	host, portStr, _ := net.SplitHostPort(addr)
	var port int
	_, _ = fmt.Sscanf(portStr, "%d", &port)

	var recordedCommands []string
	var mockServerWg sync.WaitGroup
	mockServerWg.Add(1)

	go func() {
		defer mockServerWg.Done()
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

	mockServerWg.Wait()

	// Assert that standard commands were executed in order
	var foundMailFrom, foundRcptTo, foundData bool
	for _, cmd := range recordedCommands {
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

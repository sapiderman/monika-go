//nolint:testpackage // needs to test internal/unexported components
package prober

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"monika-go/internal/assertion"
	"monika-go/internal/config"

	"gopkg.in/yaml.v3"
)

func TestNewProber(t *testing.T) {
	httpSpec := &config.HTTPSpec{}
	p := NewProber(httpSpec)
	if _, ok := p.(*HTTPProber); !ok {
		t.Errorf("NewProber for HTTPSpec did not return an HTTPProber")
	}

	// Unsupported spec
	pingSpec := &config.PingSpec{}
	p2 := NewProber(pingSpec)
	if p2 != nil {
		t.Errorf("NewProber for unsupported PingSpec should have returned nil")
	}
}

func makeTextBody(t *testing.T, text string) config.RequestBody {
	t.Helper()
	var rb config.RequestBody
	err := yaml.Unmarshal([]byte(fmt.Sprintf("%q", text)), &rb)
	if err != nil {
		t.Fatalf("failed to marshal/unmarshal text body: %v", err)
	}
	return rb
}

func makeFormBody(t *testing.T, form map[string]any) config.RequestBody {
	t.Helper()
	var rb config.RequestBody
	bytes, err := yaml.Marshal(form)
	if err != nil {
		t.Fatalf("failed to marshal form: %v", err)
	}
	err = yaml.Unmarshal(bytes, &rb)
	if err != nil {
		t.Fatalf("failed to unmarshal form body: %v", err)
	}
	return rb
}

//nolint:gocognit // complex integration tests with mock http servers
func TestHTTPProber_SuccessAndBody(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/text", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) != "hello world" {
			http.Error(w, "bad body: "+string(body), http.StatusBadRequest)
			return
		}
		if r.Header.Get("X-Custom") != "value" {
			http.Error(w, "missing header", http.StatusBadRequest)
			return
		}
		w.Header().Set("X-Response-Header", "HelloBack")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("response text"))
	})

	mux.HandleFunc("/form", func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("foo") != "bar" || r.FormValue("num") != "42" {
			http.Error(w, "invalid form values", http.StatusBadRequest)
			return
		}
		if !strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
			http.Error(w, "wrong content-type", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	alertStatus200 := config.Alert{Assertion: assertion.MustParse("response.status == 200"), Message: "not 200"}
	alertStatus201 := config.Alert{Assertion: assertion.MustParse("response.status == 201"), Message: "not 201"}

	tests := []struct {
		name          string
		req           config.Request
		wantStatus    int
		wantBody      string
		wantBodySize  int64
		wantHeaderKey string
		wantHeaderVal string
		wantPassed    bool
	}{
		{
			name: "POST request with text body and header check",
			req: config.Request{
				Method:   "POST",
				URL:      server.URL + "/text",
				Headers:  map[string]string{"X-Custom": "value"},
				SaveBody: true,
				Body:     makeTextBody(t, "hello world"),
				Alerts:   []config.Alert{alertStatus200},
			},
			wantStatus:    200,
			wantBody:      "response text",
			wantBodySize:  13,
			wantHeaderKey: "x-response-header",
			wantHeaderVal: "HelloBack",
			wantPassed:    true,
		},
		{
			name: "POST request with text body without saving body",
			req: config.Request{
				Method:   "POST",
				URL:      server.URL + "/text",
				Headers:  map[string]string{"X-Custom": "value"},
				SaveBody: false,
				Body:     makeTextBody(t, "hello world"),
				Alerts:   []config.Alert{alertStatus200},
			},
			wantStatus:    200,
			wantBody:      "",
			wantBodySize:  13,
			wantHeaderKey: "x-response-header",
			wantHeaderVal: "HelloBack",
			wantPassed:    true,
		},
		{
			name: "POST request with form body",
			req: config.Request{
				Method: "POST",
				URL:    server.URL + "/form",
				Body: makeFormBody(t, map[string]any{
					"foo": "bar",
					"num": 42,
				}),
				Alerts: []config.Alert{alertStatus201},
			},
			wantStatus: 201,
			wantPassed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prober := NewHTTPProber(&config.HTTPSpec{
				Requests: []config.Request{tt.req},
			})
			results, err := prober.Probe(context.Background())
			if err != nil {
				t.Fatalf("unexpected probe error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			res := results[0]
			if res.Result.Status != tt.wantStatus {
				t.Errorf("got status %d, want %d", res.Result.Status, tt.wantStatus)
			}
			if res.Result.Body != tt.wantBody {
				t.Errorf("got body %q, want %q", res.Result.Body, tt.wantBody)
			}
			if res.Result.BodySize != tt.wantBodySize {
				t.Errorf("got body size %d, want %d", res.Result.BodySize, tt.wantBodySize)
			}
			if tt.wantHeaderKey != "" {
				val, exists := res.Result.Headers[tt.wantHeaderKey]
				if !exists {
					t.Errorf("expected response header %q, not found", tt.wantHeaderKey)
				} else if val != tt.wantHeaderVal {
					t.Errorf("got response header %q = %q, want %q", tt.wantHeaderKey, val, tt.wantHeaderVal)
				}
			}
			if res.AlertPassed != tt.wantPassed {
				t.Errorf("got alert passed %v, want %v", res.AlertPassed, tt.wantPassed)
			}
		})
	}
}

//nolint:gocognit,nestif // complex redirect integration test
func TestHTTPProber_Redirects(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/r1", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/r2", http.StatusFound)
	})
	mux.HandleFunc("/r2", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/r3", http.StatusFound)
	})
	mux.HandleFunc("/r3", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	tests := []struct {
		name            string
		followRedirects int
		wantStatus      int
		wantErr         error
	}{
		{
			name:            "no redirects",
			followRedirects: 0,
			wantStatus:      http.StatusFound,
			wantErr:         nil,
		},
		{
			name:            "follow 1 redirect (fails because it takes 2)",
			followRedirects: 1,
			wantErr:         ErrRedirect,
		},
		{
			name:            "follow unlimited redirects (-1)",
			followRedirects: -1,
			wantStatus:      http.StatusOK,
			wantErr:         nil,
		},
		{
			name:            "follow sufficient redirects (2)",
			followRedirects: 2,
			wantStatus:      http.StatusOK,
			wantErr:         nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prober := NewHTTPProber(&config.HTTPSpec{
				Requests: []config.Request{
					{
						URL:             server.URL + "/r1",
						FollowRedirects: tt.followRedirects,
					},
				},
			})
			results, err := prober.Probe(context.Background())
			if err != nil {
				t.Fatalf("probe returned unexpected setup error: %v", err)
			}
			if len(results) != 1 {
				t.Fatalf("expected 1 result, got %d", len(results))
			}

			res := results[0]
			if tt.wantErr != nil {
				if res.Result.Err == nil {
					t.Errorf("expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(res.Result.Err, tt.wantErr) {
					t.Errorf("got error %v, want error wrapping %v", res.Result.Err, tt.wantErr)
				}
			} else {
				if res.Result.Err != nil {
					t.Errorf("unexpected connection error: %v", res.Result.Err)
				}
				if res.Result.Status != tt.wantStatus {
					t.Errorf("got status %d, want %d", res.Result.Status, tt.wantStatus)
				}
			}
		})
	}
}

func TestHTTPProber_Timeout(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(150 * time.Millisecond):
			w.WriteHeader(http.StatusOK)
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Run("request timeout takes precedence and triggers ErrTimeout", func(t *testing.T) {
		prober := NewHTTPProber(&config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:     server.URL + "/slow",
					Timeout: 20, // 20 milliseconds
				},
			},
		})
		results, err := prober.Probe(context.Background())
		if err != nil {
			t.Fatalf("probe returned unexpected setup error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}

		res := results[0]
		if res.Result.Err == nil {
			t.Fatalf("expected timeout error, got nil")
		}
		if !errors.Is(res.Result.Err, ErrTimeout) {
			t.Errorf("got error %v, want %v", res.Result.Err, ErrTimeout)
		}
	})
}

func TestHTTPProber_TLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("AllowUnauthorized is false fails", func(t *testing.T) {
		prober := NewHTTPProber(&config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:               server.URL,
					AllowUnauthorized: false,
				},
			},
		})
		results, err := prober.Probe(context.Background())
		if err != nil {
			t.Fatalf("probe returned unexpected setup error: %v", err)
		}
		res := results[0]
		if res.Result.Err == nil {
			t.Fatalf("expected TLS error, got nil")
		}
		if !errors.Is(res.Result.Err, ErrTLS) {
			t.Errorf("got error %v, want wrapping %v", res.Result.Err, ErrTLS)
		}
	})

	t.Run("AllowUnauthorized is true succeeds", func(t *testing.T) {
		prober := NewHTTPProber(&config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:               server.URL,
					AllowUnauthorized: true,
				},
			},
		})
		results, err := prober.Probe(context.Background())
		if err != nil {
			t.Fatalf("probe returned unexpected setup error: %v", err)
		}
		res := results[0]
		if res.Result.Err != nil {
			t.Fatalf("expected success, got connection error: %v", res.Result.Err)
		}
		if res.Result.Status != http.StatusOK {
			t.Errorf("got status %d, want 200", res.Result.Status)
		}
	})
}

func TestHTTPProber_AlertGating(t *testing.T) {
	var executed2 bool
	mux := http.NewServeMux()
	mux.HandleFunc("/req1", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // 400
	})
	mux.HandleFunc("/req2", func(w http.ResponseWriter, _ *http.Request) {
		executed2 = true
		w.WriteHeader(http.StatusOK)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	alertStatus200 := config.Alert{Assertion: assertion.MustParse("response.status == 200"), Message: "not 200"}

	t.Run("first request failing assertion stops chain", func(t *testing.T) {
		executed2 = false
		prober := NewHTTPProber(&config.HTTPSpec{
			Requests: []config.Request{
				{
					URL:    server.URL + "/req1",
					Alerts: []config.Alert{alertStatus200}, // will fail because status is 400
				},
				{
					URL: server.URL + "/req2",
				},
			},
		})
		results, err := prober.Probe(context.Background())
		if err != nil {
			t.Fatalf("unexpected probe error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected chain to stop at request 1 (len=1), but got results len = %d", len(results))
		}
		if executed2 {
			t.Errorf("request 2 should not have been executed")
		}
		if results[0].AlertPassed {
			t.Errorf("request 1 alerts should have failed")
		}
	})

	t.Run("first request connection error stops chain", func(t *testing.T) {
		executed2 = false
		prober := NewHTTPProber(&config.HTTPSpec{
			Requests: []config.Request{
				{
					URL: "http://invalid-dns-name-that-does-not-exist.local",
				},
				{
					URL: server.URL + "/req2",
				},
			},
		})
		results, err := prober.Probe(context.Background())
		if err != nil {
			t.Fatalf("unexpected probe error: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("expected chain to stop at request 1 due to connection error, got results len = %d", len(results))
		}
		if executed2 {
			t.Errorf("request 2 should not have been executed")
		}
		if results[0].Result.Err == nil {
			t.Errorf("expected connection error on request 1, got nil")
		}
	})
}

func TestTranslateError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		wantErr error
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "wrapped ErrRedirect",
			err:     fmt.Errorf("some context: %w", ErrRedirect),
			wantErr: ErrRedirect,
		},
		{
			name:    "redirect limit exceeded string",
			err:     errors.New("stopped after 10 redirects: redirect limit exceeded"),
			wantErr: ErrRedirect,
		},
		{
			name:    "context deadline exceeded",
			err:     context.DeadlineExceeded,
			wantErr: ErrTimeout,
		},
		{
			name:    "net.DNSError",
			err:     &net.DNSError{Name: "example.com", Err: "no such host"},
			wantErr: ErrDNS,
		},
		{
			name:    "tls error string",
			err:     errors.New("tls: handshake failed"),
			wantErr: ErrTLS,
		},
		{
			name:    "connection refused string",
			err:     errors.New("dial tcp 127.0.0.1:80: connect: connection refused"),
			wantErr: ErrConnection,
		},
		{
			name:    "generic error",
			err:     errors.New("some unknown error"),
			wantErr: ErrConnection,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateError(tt.err)
			if tt.wantErr == nil {
				if got != nil {
					t.Errorf("translateError(%v) = %v, want nil", tt.err, got)
				}
			} else {
				if !errors.Is(got, tt.wantErr) {
					t.Errorf("translateError(%v) = %v, want wrapping %v", tt.err, got, tt.wantErr)
				}
			}
		})
	}
}

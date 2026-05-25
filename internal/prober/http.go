package prober

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"monika-go/internal/assertion"
	"monika-go/internal/config"
)

// HTTPProber executes a chain of HTTP requests specified in config.HTTPSpec.
type HTTPProber struct {
	spec *config.HTTPSpec
}

// NewHTTPProber constructs an HTTPProber with the given HTTPSpec.
func NewHTTPProber(spec *config.HTTPSpec) *HTTPProber {
	return &HTTPProber{spec: spec}
}

// Probe executes the HTTP requests sequentially. It stops early on connection/network error,
// or if any request-level alert assertion fails.
//
//nolint:nilerr // res.Result.Err is returned inside RequestResult, not as a prober-level setup error
func (p *HTTPProber) Probe(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(p.spec.Requests))

	for _, req := range p.spec.Requests {
		res, err := p.executeRequest(ctx, req)
		if err != nil {
			return nil, err
		}

		results = append(results, res)

		// Chain stopping rules:
		// 1. Connection error - ProbeResult.Err != nil
		// 2. Failed request-level alert - AlertPassed == false
		hasError := res.Result.Err != nil
		if hasError || !res.AlertPassed {
			break
		}
	}

	return results, nil
}

func prepareBody(req config.Request) (io.Reader, string) {
	if req.Body.IsText() {
		return strings.NewReader(req.Body.Text()), ""
	}
	if req.Body.Form() != nil {
		values := url.Values{}
		for k, v := range req.Body.Form() {
			values.Set(k, fmt.Sprintf("%v", v))
		}
		return strings.NewReader(values.Encode()), "application/x-www-form-urlencoded"
	}
	return nil, ""
}

func createHTTPClient(req config.Request) (*http.Client, func()) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: req.AllowUnauthorized, //nolint:gosec // user configuration choice
		},
	}
	client := &http.Client{
		Transport: tr,
	}

	client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if req.FollowRedirects == 0 {
			return http.ErrUseLastResponse
		}
		if req.FollowRedirects > 0 && len(via) > req.FollowRedirects {
			return ErrRedirect
		}
		if req.FollowRedirects < 0 && len(via) > 100 {
			return ErrRedirect
		}
		return nil
	}

	return client, tr.CloseIdleConnections
}

func (p *HTTPProber) executeRequest(ctx context.Context, req config.Request) (RequestResult, error) {
	method := req.Method
	if method == "" {
		method = "GET"
	}

	bodyReader, contentType := prepareBody(req)

	// Handle tighter request timeout
	reqCtx := ctx
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Millisecond)
		defer cancel()
	}

	httpReq, err := http.NewRequestWithContext(reqCtx, method, req.URL, bodyReader)
	if err != nil {
		return RequestResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	if contentType != "" && httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", contentType)
	}

	client, cleanup := createHTTPClient(req)
	defer cleanup()

	start := time.Now()
	resp, err := client.Do(httpReq)
	duration := time.Since(start).Milliseconds()

	var probeResult assertion.ProbeResult

	if err != nil {
		probeResult = assertion.ProbeResult{
			Err: translateError(err),
		}
	} else {
		var bodyBytes []byte
		if resp.Body != nil {
			defer resp.Body.Close()
			bodyBytes, _ = io.ReadAll(resp.Body)
		}

		headers := make(map[string]string)
		for k, v := range resp.Header {
			headers[strings.ToLower(k)] = strings.Join(v, ", ")
		}

		var bodyStr string
		if req.SaveBody {
			bodyStr = string(bodyBytes)
		}

		probeResult = assertion.ProbeResult{
			Status:       resp.StatusCode,
			ResponseTime: duration,
			BodySize:     int64(len(bodyBytes)),
			Headers:      headers,
			Body:         bodyStr,
		}
	}

	// Evaluate request-level alerts
	failedAlerts := make([]FailedAlert, 0)
	alertPassed := true

	for _, alert := range req.Alerts {
		if alert.Assertion == nil {
			continue
		}
		if !alert.Assertion.Evaluate(probeResult) {
			alertPassed = false
			failedAlerts = append(failedAlerts, FailedAlert{
				Assertion: alert.Assertion.String(),
				Message:   alert.Message,
			})
		}
	}

	return RequestResult{
		Result:       probeResult,
		AlertPassed:  alertPassed,
		FailedAlerts: failedAlerts,
	}, nil
}

func translateError(err error) error {
	if err == nil {
		return nil
	}

	errStr := err.Error()

	// 1. Redirect
	if errors.Is(err, ErrRedirect) ||
		strings.Contains(errStr, "redirect limit exceeded") ||
		strings.Contains(errStr, "stopped after") {
		return fmt.Errorf("%s: %w", errStr, ErrRedirect)
	}

	// 2. Timeout
	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) ||
		(errors.As(err, &netErr) && netErr.Timeout()) ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return fmt.Errorf("%s: %w", errStr, ErrTimeout)
	}

	// 3. DNS
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) ||
		strings.Contains(errStr, "lookup") ||
		strings.Contains(errStr, "no such host") {
		return fmt.Errorf("%s: %w", errStr, ErrDNS)
	}

	// 4. TLS
	if strings.Contains(errStr, "tls:") ||
		strings.Contains(errStr, "handshake") ||
		strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "remote error:") {
		return fmt.Errorf("%s: %w", errStr, ErrTLS)
	}

	// 5. Connection / Network fallback
	var opErr *net.OpError
	if errors.As(err, &opErr) ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "dial") ||
		strings.Contains(errStr, "reset by peer") {
		return fmt.Errorf("%s: %w", errStr, ErrConnection)
	}

	return fmt.Errorf("%s: %w", errStr, ErrConnection)
}

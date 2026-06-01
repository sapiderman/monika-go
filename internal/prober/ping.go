package prober

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"monika-go/internal/assertion"
	"monika-go/internal/config"
)

// PingProber sends ICMP echo requests using the system's native ping utility.
type PingProber struct {
	spec *config.PingSpec
}

// NewPingProber constructs a PingProber with the given PingSpec.
func NewPingProber(spec *config.PingSpec) *PingProber {
	return &PingProber{spec: spec}
}

// Probe executes ping checks for each target.
func (p *PingProber) Probe(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(p.spec.Targets))

	for _, target := range p.spec.Targets {
		res, err := p.executePing(ctx, target)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	return results, nil
}

func (p *PingProber) executePing(ctx context.Context, target config.Ping) (RequestResult, error) {
	host := parseHost(target.URI)
	if host == "" {
		return RequestResult{}, fmt.Errorf("empty host for target %q: %w", target.URI, ErrConnection)
	}

	// Default 10 seconds timeout per Monika specification
	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var cmdName = "ping"
	var cmdArgs []string
	if runtime.GOOS == "windows" {
		cmdArgs = []string{"-n", "1", host}
	} else {
		cmdArgs = []string{"-c", "1", host}
	}

	start := time.Now()
	cmd := exec.CommandContext(reqCtx, cmdName, cmdArgs...)
	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	var probeResult assertion.ProbeResult

	if err != nil {
		// Wrap with context deadline error if context expired
		if reqCtx.Err() != nil {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("ping timeout for %s: %w", host, ErrTimeout),
			}
		} else {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("ping failed for %s (%s): %w", host, strings.TrimSpace(string(output)), ErrConnection),
			}
		}
	} else {
		probeResult = assertion.ProbeResult{
			Status:       0, // 0 for successful ping status
			ResponseTime: duration,
			Body:         string(output),
		}
	}

	return RequestResult{
		Result:      probeResult,
		AlertPassed: true,
	}, nil
}

func parseHost(uri string) string {
	if uri == "" {
		return ""
	}
	parsedURI := uri
	if !strings.Contains(uri, "://") {
		parsedURI = "http://" + uri
	}
	u, err := url.Parse(parsedURI)
	if err != nil {
		return uri
	}
	host := u.Hostname()
	if host == "" {
		return uri
	}
	return host
}

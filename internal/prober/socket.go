package prober

import (
	"context"
	"fmt"
	"net"
	"time"

	"monika-go/internal/assertion"
	"monika-go/internal/config"
)

// SocketProber checks TCP socket connectivity by attempting to establish a TCP connection.
type SocketProber struct {
	spec *config.SocketSpec
}

// NewSocketProber constructs a SocketProber with the given SocketSpec.
func NewSocketProber(spec *config.SocketSpec) *SocketProber {
	return &SocketProber{spec: spec}
}

// Probe executes socket connectivity checks for each target.
func (p *SocketProber) Probe(ctx context.Context) ([]RequestResult, error) {
	results := make([]RequestResult, 0, len(p.spec.Targets))

	for _, target := range p.spec.Targets {
		res, err := p.executeSocket(ctx, target)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}

	return results, nil
}

func (p *SocketProber) executeSocket(ctx context.Context, target config.Socket) (RequestResult, error) {
	address := fmt.Sprintf("%s:%d", target.Host, target.Port)

	dialCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	var dialer net.Dialer
	start := time.Now()
	conn, err := dialer.DialContext(dialCtx, "tcp", address)
	duration := max(time.Since(start).Milliseconds(), 1) // floor 1ms: sub-ms success must not report 0

	var probeResult assertion.ProbeResult

	if err != nil {
		if dialCtx.Err() != nil {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("socket connect timeout to %s: %w", address, ErrTimeout),
			}
		} else {
			probeResult = assertion.ProbeResult{
				Err: fmt.Errorf("socket connect failed to %s: %w", address, ErrConnection),
			}
		}
	} else {
		defer conn.Close()

		// If data payload is provided, write and read response back
		if target.Data != "" {
			err = conn.SetDeadline(time.Now().Add(5 * time.Second))
			if err != nil {
				probeResult = assertion.ProbeResult{
					Err: fmt.Errorf("socket failed to set deadline: %w", ErrConnection),
				}
			} else {
				_, err = conn.Write([]byte(target.Data))
				if err != nil {
					probeResult = assertion.ProbeResult{
						Err: fmt.Errorf("socket failed to write data: %w", ErrConnection),
					}
				} else {
					buf := make([]byte, 1024)
					n, errRead := conn.Read(buf)
					if errRead != nil {
						// Simple socket check may close connection, or might return EOF. Let's record what we got.
						probeResult = assertion.ProbeResult{
							Status:       0,
							ResponseTime: duration,
							Body:         "",
						}
					} else {
						probeResult = assertion.ProbeResult{
							Status:       0,
							ResponseTime: duration,
							Body:         string(buf[:n]),
							BodySize:     int64(n),
						}
					}
				}
			}
		} else {
			probeResult = assertion.ProbeResult{
				Status:       0,
				ResponseTime: duration,
			}
		}
	}

	return RequestResult{
		Result:      probeResult,
		AlertPassed: true,
	}, nil
}

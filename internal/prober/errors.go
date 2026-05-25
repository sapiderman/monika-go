package prober

import "errors"

var (
	ErrTimeout    = errors.New("probe: request timed out")
	ErrDNS        = errors.New("probe: dns resolution failed")
	ErrConnection = errors.New("probe: connection failed")
	ErrTLS        = errors.New("probe: tls handshake failed")
	ErrRedirect   = errors.New("probe: redirect limit exceeded")
)

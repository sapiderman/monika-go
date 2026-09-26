# AGENTS.md — monika-go

Synthetic monitoring tool: reads `monika.yaml`, probes targets on a schedule, alerts on failures.

## Stack

Go 1.27 · cobra (CLI) · yaml.v3 · logrus (only `internal/logger` imports it) · testify.

## Layout

- `cmd/` — cobra commands (`root.go` is the entrypoint wiring)
- `internal/config/` — YAML parsing, validation, probe specs
- `internal/prober/` — probers: http, ping (ICMP), socket (TCP), db; `Prober` interface in `prober.go`
- `internal/scheduler/` — concurrent probe scheduling
- `internal/alert/` + `internal/notification/` — probes emit events; managers consume (slack, smtp, webhook, desktop)
- `internal/assertion/` — response assertion expressions
- `internal/logger/` — injectable Logger interface; use `logger.Component/TraceID/DurationMS` field helpers

Domain glossary: see `CONTEXT.md`. Config samples: `monika.yaml`.

## Rules

- Wrap errors: `fmt.Errorf("probe failed: %w", err)`. No panic in prod code.
- Every prober respects `context.Context` for timeout/cancellation.
- Table-driven tests; run `make test` (race, coverage) — lint is `make lint` (staticcheck, gofmt).
- Godoc on all exported symbols.

## Build

`make build` (static binary) · `make run` · `make docker`.

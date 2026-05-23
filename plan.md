# Implementation Plan: Config Parsing Rebuild

## Summary

Rebuild `internal/config/` to parse `monika.yaml` according to the official [Monika config schema](https://github.com/hyperjumptech/monika/blob/main/src/monika-config-schema.json). Replace Viper with `gopkg.in/yaml.v3`. Strict validation, fail-fast on errors.

## Design Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Schema source | Match reference Monika JSON schema | Compatibility with existing `monika.yaml` files |
| Notification data | `map[string]any` | Opaque to prober engine, extensible without code changes |
| Probe types | Flat struct with optional pointers + post-parse validation | yaml.v3-friendly, no custom unmarshaling, validation catches errors |
| YAML library | `gopkg.in/yaml.v3` (drop Viper) | ADR-0001: only need read + unmarshal, Viper is over-engineering |
| File structure | `types.go` + `parse.go` + `validate.go` | Separate concerns: data, I/O, validation |
| Body field | `any` type | YAML natively handles map or string, type-assert when needed |
| Alert levels | Both probe-scoped and request-scoped | Prevents data loss on parse |
| Alert format | `assertion` only | `query` is deprecated, silently dropped on parse |
| `followRedirects` | `int` (max redirect count) | Matches reference implementation behavior |
| Default values | Zero-value structs, prober decides defaults | Config struct is faithful mirror of YAML, ADR-0002 |
| Unknown fields | Reject (strict parsing) | Fail fast on typos like `timout` instead of `timeout` |

## Scope

**In scope:** Read YAML file → unmarshal into Go structs → validate → return `Config`.  
**Out of scope:** Prober engine, alert evaluation, notification sending, CLI flags, config watching, remote config URLs.

## File Layout

```
internal/config/
├── types.go      # All struct definitions matching the schema
├── parse.go      # Read YAML file, unmarshal with strict mode
└── validate.go   # Post-parse validation, return errors on invalid config
```

---

## Step 1: Remove Viper, add yaml.v3

**What:** Swap dependencies in `go.mod`.

**Actions:**
1. Remove `github.com/spf13/viper` from `go.mod` (run `go mod tidy` after)
2. Add `gopkg.in/yaml.v3` (run `go get gopkg.in/yaml.v3`)
3. Remove Viper references from `cmd/root.go` and `cmd/version.go`
4. Fix compilation errors in `cmd/` — replace `viper.GetString("app.version")` with a const or flag in `cmd/version.go`

**Verify:** `go build ./...` passes with zero errors.

---

## Step 2: Write `types.go`

**What:** All Go structs matching the Monika config schema. Zero business logic.

**Struct definitions:**

```go
package config

// Config represents the top-level monika.yaml configuration.
type Config struct {
    Probes        []Probe        `yaml:"probes"`
    Notifications []Notification `yaml:"notifications"`
}

// Probe represents a single monitoring target.
// Exactly one of Requests, Ping, Socket, Mongo,Redis, Postgres, MariaDB must be set (validated later).
type Probe struct {
    ID                  string   `yaml:"id"`
    Name                string   `yaml:"name"`
    Description         string   `yaml:"description"`
    Interval            int      `yaml:"interval"`
    IncidentThreshold   int      `yaml:"incidentThreshold"`
    RecoveryThreshold   int      `yaml:"recoveryThreshold"`
    Alerts              []Alert  `yaml:"alerts"`
    Requests            *[]Request  `yaml:"requests"`
    Ping                *[]Ping     `yaml:"ping"`
    Socket              *[]Socket   `yaml:"socket"`
    Mongo               *[]MongoDB  `yaml:"mongo"`
    Redis               *[]Redis    `yaml:"redis"`
    Postgres            *[]Postgres `yaml:"postgres"`
    MariaDB             *[]MariaDB  `yaml:"mariadb"`
    MySQL               *[]MariaDB  `yaml:"mysql"`    // MySQL shares schema with MariaDB
}

// Request represents a single HTTP request within a probe.
type Request struct {
    Method            string            `yaml:"method"`
    URL               string            `yaml:"url"`
    Timeout           int               `yaml:"timeout"`
    SaveBody          bool              `yaml:"saveBody"`
    AllowUnauthorized bool              `yaml:"allowUnauthorized"`
    FollowRedirects   int               `yaml:"followRedirects"`
    Headers           map[string]string `yaml:"headers"`
    Body              any               `yaml:"body"`
    Alerts             []Alert          `yaml:"alerts"`
    Interval          int               `yaml:"interval"`
}

// Ping represents an ICMP ping probe target.
type Ping struct {
    URI string `yaml:"uri"`
}

// Socket represents a TCP socket probe target.
type Socket struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
    Data string `yaml:"data"`
}

// MongoDB represents a MongoDB health check target.
type MongoDB struct {
    URI      string `yaml:"uri"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

// Redis represents a Redis health check target.
type Redis struct {
    URI      string `yaml:"uri"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

// Postgres represents a PostgreSQL health check target.
type Postgres struct {
    URI      string `yaml:"uri"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Database string `yaml:"database"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

// MariaDB represents a MariaDB/MySQL health check target.
type MariaDB struct {
    URI      string `yaml:"uri"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Database string `yaml:"database"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
}

// Alert defines an assertion that triggers a notification.
type Alert struct {
    ID         string `yaml:"id"`
    Assertion  string `yaml:"assertion"`
    Message    string `yaml:"message"`
}

// Notification represents a notification channel configuration.
type Notification struct {
    ID   string         `yaml:"id"`
    Type string         `yaml:"type"`
    Data map[string]any `yaml:"data"`
}
```

**Notes:**
- All pointer fields for probe sub-types use `*[]T` so nil means "not present in YAML"
- `Body any` accepts both `map[string]any` (form data) and `string` (raw text/XML)
- `MySQL` reuses `MariaDB` struct — identical schema
- No `Query` field on `Alert` — deprecated, silently dropped on parse
- No defaults in struct tags — config is a faithful mirror of YAML
- Typo fix: `RecovveryThreshold` → `RecoveryThreshold`

**Verify:** `go build ./internal/config/` compiles without errors.

---

## Step 3: Write `parse.go`

**What:** Read a YAML file and unmarshal into `Config` with strict mode (reject unknown fields).

**Functions:**

```go
package config

import (
    "fmt"
    "os"

    "gopkg.in/yaml.v3"
)

// Parse reads a YAML file and returns a Config struct.
// Returns an error if the file cannot be read or contains unknown fields.
func Parse(filename string) (*Config, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    return ParseBytes(data)
}

// ParseBytes unmarshals YAML bytes into a Config struct.
// Uses strict decoding to reject unknown fields.
func ParseBytes(data []byte) (*Config, error) {
    var cfg Config
    decoder := yaml.NewDecoder(bytes.NewReader(data))
    decoder.KnownFields(true) // strict mode: reject unknown fields

    if err := decoder.Decode(&cfg); err != nil {
        return nil, fmt.Errorf("parse config: %w", err)
    }

    return &cfg, nil
}
```

**Key behaviors:**
- `KnownFields(true)` makes the decoder reject any YAML key that doesn't map to a struct field — this catches typos like `timout`
- `ParseBytes` is separate for testability — tests can pass raw YAML strings without touching the filesystem
- No defaults applied — what's in the file is what you get

**Verify:** Unit test that parses the existing `monika.yaml` sample successfully.

---

## Step 4: Write `validate.go`

**What:** Post-parse validation. Every rule returns a clear error message.

**Validation rules:**

| # | Rule | Error Message |
|---|---|---|
| 1 | At least one probe must exist | `"config must define at least one probe"` |
| 2 | Each probe must have a non-empty `id` | `"probe at index %d: id is required"` |
| 3 | Each probe must have exactly one probe type | `"probe %q: must have exactly one probe type, got %d"` |
| 4 | Each HTTP request must have a non-empty `url` | `"probe %q: request at index %d: url is required"` |
| 5 | Each ping must have a non-empty `uri` | `"probe %q: ping at index %d: uri is required"` |
| 6 | Each socket must have non-empty `host`, `port > 0`, non-empty `data` | `"probe %q: socket at index %d: host, port, and data are required"` |
| 7 | Each alert must have a non-empty `assertion` | `"probe %q: alert at index %d: assertion is required"` |
| 8 | Probe IDs must be unique across the config | `"duplicate probe id: %q"` |
| 9 | Notification `id` must be non-empty | `"notification at index %d: id is required"` |
| 10 | Notification `type` must be one of the known types | `"notification %q: unknown type %q"` |
| 11 | Each request alert must have a non-empty `assertion` | `"probe %q: request at index %d: alert at index %d: assertion is required"` |

**Known notification types:** smtp, slack, webhook, telegram, discord, teams, lark, mailgun, sendgrid, instatus, opsgenie, pushover, workplace, dingtalk, monika-notif, whatsapp

**Function signature:**

```go
package config

import (
    "fmt"
    "slices"
    "strings"
)

var knownNotificationTypes = []string{
    "smtp", "slack", "webhook", "telegram", "discord", "teams",
    "lark", "mailgun", "sendgrid", "instatus", "opsgenie",
    "pushover", "workplace", "dingtalk", "monika-notif", "whatsapp",
}

// Validate checks a Config for semantic errors.
// Returns the first error encountered, or nil if valid.
func Validate(cfg *Config) error { ... }

// probeTypeCount counts how many probe type fields are set.
func probeTypeCount(p *Probe) int { ... }
```

**Notes:**
- Validation is separate from parsing — `Parse` returns the raw struct, `Validate` checks business rules
- Returns first error only — fail fast, don't collect all errors (matches "fail in the beginning" principle)
- Database probe types (mongo, redis, postgres, mariadb) with `uri` field: validate URI format? Or just check non-empty?
  - **Decision: just check non-empty.** URI format validation belongs to the prober, not the config parser.

**Verify:** Unit tests for each validation rule — one valid config, one invalid config per rule.

---

## Step 5: Wire into `cmd/root.go`

**What:** Replace Viper-based config loading with `Parse` + `Validate`.

**Changes to `cmd/root.go`:**
1. Remove `config.LoadDefaultConfig()` from `init()`
2. Remove Viper import
3. Store loaded config in a package-level var accessible by other commands
4. Load config in a `PersistentPreRunE` so it's available to all subcommands
5. Add `--config` / `-c` flag to root command (default: `monika.yaml`)

**Changes to `cmd/version.go`:**
1. Replace `viper.GetString("app.version")` with a `var Version = "0.0.1"` set at build time via `-ldflags`

**Changes to `cmd/config.go`:**
1. Remove stub — config path is now a root flag, not a subcommand

**Changes to `cmd/create_config.go`:**
1. Leave as stub for now (out of scope)

**Structure:**

```go
var (
    cfgFile string
    cfg     *config.Config
)

func init() {
    rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "monika.yaml", "config file path")
}

// PersistentPreRunE loads and validates config before any subcommand runs.
rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []error) error {
    parsed, err := config.Parse(cfgFile)
    if err != nil {
        return fmt.Errorf("loading config: %w", err)
    }
    if err := config.Validate(parsed); err != nil {
        return fmt.Errorf("validating config: %w", err)
    }
    cfg = parsed
    return nil
}
```

**Verify:** `go run . -c monika.yaml` loads, parses, validates the config, and exits cleanly. A typo in `monika.yaml` produces a clear error.

---

## Step 6: Fix `monika.yaml` sample

**What:** Update the sample config to match the new strict parsing.

**Current issues in `monika.yaml`:**
- Contains `query` fields in alerts (deprecated, will be silently dropped — OK)
- Verify it parses cleanly after changes

**Verify:** `go run . -c monika.yaml` succeeds without errors.

---

## Step 7: Write table-driven tests

**What:** Comprehensive tests for `parse.go` and `validate.go`.

### `parse_test.go`

```go
// Test cases:
// - valid minimal config (only required fields)
// - valid full config (all probe types)
// - unknown field in probe → error
// - unknown field in request → error
// - unknown field at top level → error
// - empty file → error (no probes)
```

### `validate_test.go`

```go
// Test cases (table-driven):
// - valid config with single HTTP probe
// - valid config with ping probe
// - valid config with socket probe
// - valid config with database probes (mongo, redis, postgres, mariadb)
// - multiple probes, multiple requests
// - probe with no type → error
// - probe with multiple types → error
// - empty probe id → error
// - duplicate probe id → error
// - request missing url → error
// - ping missing uri → error
// - socket missing required fields → error
// - alert missing assertion → error
// - notification with unknown type → error
// - notification with empty id → error
```

**Verify:** `go test ./internal/config/ -v` — all tests pass.

---

## Execution Order

```
Step 1 → verify: go build ./...
Step 2 → verify: go build ./internal/config/
Step 3 → verify: unit test Parse with monika.yaml
Step 4 → verify: unit tests for each validation rule
Step 5 → verify: go run . -c monika.yaml
Step 6 → verify: go run . -c monika.yaml (clean exit)
Step 7 → verify: go test ./internal/config/ -v
```

Each step must pass its verify gate before proceeding to the next.

---

## Open Questions (deferred)

1. **Remote config URLs** — the reference Monika supports `monika -c https://...`. Out of scope for now.
2. **Config file watching** — the reference Monika reloads on file change. Out of scope.
3. **HAR/Postman/Insomnia/Sitemap import** — supported by reference Monika CLI. Out of scope.
4. **Assertion expression parsing** — `response.status != 200` needs an expression evaluator. Out of scope (belongs to alert engine).
5. **Notification channel data validation** — `Data map[string]any` is opaque. Per-type validation deferred to notification sender implementation.
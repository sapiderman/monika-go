# monika-go vs Monika (JS/TS) Feature Comparison

Comparison of monika-go against the reference Monika documentation:

- https://monika.hyperjump.tech/overview
- https://monika.hyperjump.tech/guides/probes
- https://monika.hyperjump.tech/guides/alerts
- https://monika.hyperjump.tech/guides/notifications

1:1 parity is **not** a goal; this documents what is covered, partial, and deliberately absent.

Legend: ✅ covered · 🟡 partial/divergent · ❌ not implemented

---

## 1. Overview page

### Core loop ("How it works")

| Doc claim | Status | Where |
|---|---|---|
| Reads everything from a config file | ✅ | `internal/config/` — YAML parse + fail-fast validation |
| Builds & sends HTTP requests | ✅ | `internal/prober/http.go` |
| Builds & sends TCP requests | ✅ | `internal/prober/socket.go` |
| Notifies when response is unexpected | ✅ | assertion → alert tracker → `AlertManager` → notifiers |

### Feature list

| Feature | Status | Notes |
|---|---|---|
| Multiple probes, multiple HTTP requests each | ✅ | Scheduler runs probes concurrently, requests sequentially |
| Complex alert assertions (status/time/size/headers/body) | ✅ | `internal/assertion` — all 5 LHS fields, operators, type-checking |
| TCP socket probes | ✅ | `socket.go` |
| Notification channels | 🟡 4 of ~20 | smtp, slack, webhook, desktop |
| Incident/recovery thresholds | ✅ | `incidentThreshold`/`recoveryThreshold` + state tracker |
| Request chaining (response data → next request) | ❌ | No vars/env substitution |
| HAR / Postman / Insomnia / sitemap import | ❌ | Only native YAML |
| TLS certificate checkers (expiry alerts) | ❌ | `allowUnauthorized` exists, but no cert-expiry probe |
| Run Monika from a URL | ❌ | `--config` file only |
| Periodic status-notification (summary reports) | ❌ | — |
| Prometheus metrics endpoint | ❌ | Nothing in the repo |
| Multiple config files | ❌ | Single `-c` flag |

### In monika-go but not on the overview page

- **ICMP ping** and **DB probes** (mongo/redis/postgres/mariadb/mysql) — exist in reference Monika's probes guide too, just not listed on the overview.
- **Immediate first probe** (no initial interval wait) — deliberate improvement.
- **Probe-level alerts** evaluated against every request — matches Monika semantics.

---

## 2. Probes guide

### HTTP request fields

| Field | Doc | monika-go | Notes |
|---|---|---|---|
| `method`, `url`, `headers` | ✅ | ✅ | |
| `timeout` (ms, default 10000) | ✅ | ✅ | Honored in ms; unset/0 falls back to the 10s default (`prober.defaultTimeout`), shared with socket/ping/db |
| `body` (text/form) | ✅ | ✅ | Raw string + `x-www-form-urlencoded` map serialization; JSON works as raw text + manual Content-Type header |
| `alerts` (request-level) | ✅ | ✅ | |
| `saveBody` | ✅ | 🟡 | Parsed but stored nowhere — go has no internal results DB, so it's a no-op |
| `allowUnauthorized` | ✅ | ✅ | |
| `followRedirects` | ✅ | ✅ | Integer count; **missing the fallback to the `--follow-redirects` CLI flag** (flag doesn't exist) |
| `interval` (request-level) | ✅ | 🟡 | Field parsed in `types.go` but **never read** by scheduler/prober — dead field |
| `ping` (boolean per-request) | ✅ | ❌ | Go only supports ping as a separate probe type, not a request flag |
| `incidentThreshold`/`recoveryThreshold`, default 5 | ✅ | ✅ | Defaults applied in `tracker.go`; doc's "effective retries = max(incident, recovery)" quirk is **not** replicated — go uses each threshold independently (arguably more correct) |

### Probe types

| Type | Status |
|---|---|
| `requests` (HTTP) | ✅ |
| `ping` (ICMP) | ✅ |
| `socket` (TCP, incl. `data`) | ✅ |
| `mongo`, `redis`, `postgres`, `mariadb`/`mysql` | ✅ all, incl. `uri` form |
| MariaDB ≡ MySQL interchangeable | ✅ |

Minor: doc's postgres example uses `user:`; monika-go's YAML tag is `username` — verify against the reference schema, not just the doc example.

### Probe response anatomy & behavior

| Feature | Status | Notes |
|---|---|---|
| Response `status`, `headers`, `data`, `size`, `time` | ✅ | `ProbeResult` has all; `statusText` is **missing** |
| Custom HTTP status codes 0–99 for connection errors | 🟡 | Go has typed errors (`ErrTimeout`, `ErrDNS`, `ErrTLS`, …) that fail the probe run, but they're **not exposed as numeric statuses usable in assertions** (`response.status == 6`). Cleaner design, different contract |
| Response cache (TTL 5) | ❌ | Not implemented — irrelevant for a Go rewrite |
| Fake data (`{{ uuid }}`, `{{ timestamp }}`, …) | ❌ | |
| Postman / Insomnia / HAR import | ❌ | |
| `--id` / `--repeat` execution control flags | 🟡 | `-i/--id` filters probes by comma-separated IDs (config order kept, unknown IDs fail fast); `-r/--repeat N` runs each probe N times then exits. Divergence: reference runs IDs sequentially in flag order, go keeps concurrent loops |
| Sequential probe execution | 🟡 | Go runs probes **concurrently** (goroutine per probe); doc describes sequential looping. Deliberate improvement, but ordering-sensitive configs will behave differently |
| Requests within a probe run in order | ✅ | |

---

## 3. Alerts guide

| Feature | Status | Notes |
|---|---|---|
| Request alerts vs probe alerts | ✅ | Probe-level evaluated against every request result |
| Alert timing via thresholds | ✅ | |
| Built-in alerts on inaccessible probes | ✅ | Prober error → `RecordResult(false)` → incident/recovery transitions |
| `response.status/time/size/headers["k"]/body` as LHS | ✅ | All five supported |
| Comparison ops `== != < <= > >=` | ✅ | |
| Regex `~=` | ❌ | |
| `in` / `not in` | ❌ | |
| Boolean logic `and/or/not`, ternary `? :`, parentheses | ❌ | Go = exactly **one** comparison per assertion |
| Arithmetic `+ - * / % ^` | ❌ | |
| Helper functions (`has`, `lowerCase`, `includes`, `size`, …) | ❌ | |
| Deep access: `response.body.data.todos[0].title` | ❌ | Body kept as string; no JSON path access. `headers["k"]` works, body sub-fields don't |
| Message templating `{{ response.status }}` | ❌ | Message sent verbatim |

The assertion engine gap is the big one: reference Monika has a full expression language; monika-go has single comparisons. Fine until someone needs `response.status != 200 and response.time > 500` — today that's unexpressible in one alert.

---

## 4. Notifications guide

| Channel | Status | Notes |
|---|---|---|
| smtp | ✅ | All doc fields (`recipients`, `hostname`, `port`, `username`, `password`) + go-only `html` flag |
| slack | ✅ | Same `url` field |
| webhook | 🟡 | Same `url` field, but **payload shape differs**: go sends `{probeID, fromState, toState, message, ...}` transition event; doc documents `{url, time, alert}` |
| desktop | ✅ | |
| Remaining ~16 (telegram, discord, teams, mailgun, sendgrid, pagerduty, opsgenie, pushover, whatsapp, dingtalk, lark, google-chat, workplace, gotify, pushbullet, instatus/statuspage/monika-notif) | ❌ | Factory pattern makes each ~70 lines like `slack.go` |
| "Every alert goes to all configured channels" | ✅ | |

---

## Priority view (if parity is the goal)

1. **Assertion expression language** (boolean ops + `in` + body paths) — the only gap that changes what users can *express*.
2. **Webhook payload shape** — breaks drop-in consumers expecting the documented `{url, time, alert}`.
3. **Message templating `{{ }}`** — cheap, high user-visible value.
4. Delete or wire up dead `interval` (request-level) and `saveBody` fields — parsed-but-ignored config is worse than unsupported config.

Everything else (fake data, import formats, cache, remaining ~16 channels, TLS checkers, Prometheus, status notifications) is ecosystem breadth — add on demand.

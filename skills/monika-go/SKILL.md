---
name: monika-go
description: Operate monika-go, a synthetic monitoring CLI that probes HTTP endpoints, TCP sockets, ICMP ping, and databases (redis, mongo, postgres, mysql, mariadb) on a schedule, evaluates alert assertions, and sends notifications (Slack, email/SMTP, webhook, desktop). Use when the user wants to monitor a URL or service, write a monika.yaml probe config, run health checks on an interval, or set up uptime alerting.
---

# monika-go

Go port of [Monika](https://github.com/hyperjumptech/monika) (HyperJump). Runs probes on an interval, evaluates assertions, fires notifications on status transitions (incident/recovery), logs results as structured JSON.

## Build & Run

```bash
make build                  # or: go build -o monika-go .
./monika-go                 # run all probes in ./monika.yaml, forever
./monika-go -c my.yaml      # custom config path
./monika-go -i github,httpbin   # run only probes with these IDs
./monika-go -r 3            # exit after 3 rounds (default: run forever)
./monika-go createConfig    # write an example config file
./monika-go version
```

Runs until Ctrl-C (graceful shutdown) or `--repeat` is exhausted. Validation errors are reported at startup — fix the config, don't retry blindly.

## Config format (YAML)

Default file: `monika.yaml`. Top-level keys: `probes`, `notifications`.

### Probes

```yaml
probes:
  - id: my-api            # required, unique — used with -i
    name: My API
    description: Health check
    interval: 10          # seconds, default 10
    incidentThreshold: 5  # consecutive failures before INCIDENT (defaults apply)
    recoveryThreshold: 5  # consecutive successes before RECOVERY
    requests:
      - url: https://api.example.com/health
        method: GET               # GET/POST/...
        timeout: 7000             # ms
        allowUnauthorized: true   # skip TLS verify
        headers:
          Content-Type: application/json
        body: |
          {"key": "value"}
        alerts:
          - assertion: response.status == 500
            message: API returned 500
          - assertion: response.time > 1500
            message: API too slow
```

- **HTTP**: `requests:` list (checked in order; a failed request short-circuits the chain). Assertions: `response.status`, `response.time` (ms), `response.size`, `response.body`, `response.headers["key"]`; comparisons `== != > < >= <=`.
- **Ping (ICMP)**: `ping: [{uri: https://example.com}]`
- **TCP socket**: `socket: [{host: example.com, port: 443, data: ping}]` — `host`, `port`, and `data` are all required.
- **Databases**: `redis:` / `mongo:` / `postgres:` / `mysql:` / `mariadb:` — each `{host, port, username, password}`, all also accept `uri`; postgres/mysql/mariadb also `database`.

Multiple request/alert blocks per probe are allowed. Request-level and probe-level `alerts` are both evaluated for every request; a failure in either fails the run.

### Notifications

```yaml
notifications:
  - id: my-slack
    type: slack
    data:
      url: https://hooks.slack.com/services/XXX/YYY/ZZZ

  - id: my-mail
    type: smtp
    data:
      recipients: [ops@example.com]
      hostname: smtp.gmail.com
      port: 587
      username: user@gmail.com
      password: app-password

  - id: my-webhook
    type: webhook
    data:
      url: https://example.com/hook
```

Supported types: `slack`, `smtp`, `webhook`, `desktop`. Secrets belong in the config file the user controls — never invent credentials; ask or use env placeholders if the user hasn't supplied them.

## Typical tasks

1. **"Monitor this URL"** → write/extend a `monika.yaml` probe (status + latency assertions are a good default), run `./monika-go -r 1` once to validate, then leave it running or tell the user how.
2. **"Why did my probe fail?"** → read the structured JSON log output (component, error, duration); reproduce with `./monika-go -i <probe-id> -r 1`.
3. **Validate only** → `./monika-go -i <id> -r 1` exits nonzero with a config error if YAML is invalid; no need to install anything else.

## Reference config

`monika.yaml` in the repo root documents every supported field with commented examples — consult it before inventing keys. Unknown keys are a hard parse error at startup (strict decoding); when unsure of a field's existence, check this file or `internal/config/types.go`, not the original Node.js Monika docs.

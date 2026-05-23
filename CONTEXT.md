# Monika-Go Configuration

A cloud-native synthetic monitoring tool that reads a YAML configuration file and probes targets at intervals, alerting on failures.

## Language

**Configuration**: The YAML file (typically `monika.yaml`) that defines all probes, alerts, and notifications. Not to be confused with runtime config or CLI flags.
_Avoid_: config file, config, settings

**Probe**: A monitoring target definition. Contains exactly one probe type (HTTP requests, ping, socket, or database check) plus optional alerts and thresholds.
_Avoid_: check, monitor, test

**Probe Type**: The kind of check a probe performs. One of: `requests` (HTTP), `ping` (ICMP), `socket` (TCP), `mongo`, `redis`, `postgres`, `mariadb`/`mysql`.
_Avoid_: probe kind, probe category

**Request**: An HTTP request step within a probe. A probe can have multiple requests that execute sequentially (request chaining).
_Avoid_: HTTP probe, HTTP check

**Assertion**: An expression evaluated against a probe response to trigger an alert. Uses `response.status`, `response.time`, `response.size`, `response.headers`, `response.body`.
_Avoid_: query (deprecated in reference Monika), condition, rule

**Alert**: A configuration that defines an assertion and a message. Can appear at probe level (applies to all requests) or request level (applies to one request).
_Avoid_: trigger, alarm

**Notification**: A channel configuration for sending alerts. Has `id`, `type` (e.g. smtp, slack, webhook), and opaque `data`.
_Avoid_: channel, receiver

**Incident Threshold**: Number of consecutive assertion failures before an incident notification is sent. Default: 5.
_Avoid_: failure count, retry count

**Recovery Threshold**: Number of consecutive assertion passes before a recovery notification is sent. Default: 5.
_Avoid_: success count

**Config Parser**: The system that reads a YAML file and produces a validated `Config` struct. Fails fast on any error: unknown fields, missing required fields, invalid structure.

## Relationships

- A **Configuration** contains one or more **Probes** and zero or more **Notifications**
- A **Probe** contains exactly one **Probe Type** (enforced by validation)
- A **Probe** may have probe-level **Alerts** that apply to all its requests
- A **Request** may have request-level **Alerts** that apply only to that request
- An **Alert** contains an **Assertion** and a message
- A **Notification** has a `type` that determines the shape of its `data` field

## Example dialogue

> **Dev:** "When a **Probe** has both `requests` and `ping`, what happens?"
> **Domain expert:** "That's a validation error. A **Probe** must have exactly one **Probe Type**."

> **Dev:** "Should the config parser apply defaults like `interval: 10`?"
> **Domain expert:** "No — the config struct is a faithful mirror of the YAML. If `interval` is omitted, it's `0`. The prober decides what `0` means."

## Flagged ambiguities

- "query" appears in the reference Monika schema as a deprecated alias for "assertion" — resolved: we only support `assertion`. `query` is silently dropped on parse.
- `followRedirects` appears as boolean in schema description but as integer in examples — resolved: it's an `int` representing max redirect count. `0` means no redirects.
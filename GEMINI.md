# GEMINI.md

## 🚀 Project Overview

**Project Name:** `monika-go`

**Description:** A high-performance, cloud-native synthetic monitoring tool written in Go. It is designed to perform distributed probing (HTTP, TCP, ICMP, gRPC) at scale, with built-in alerting, structured logging, and seamless integration with observability stacks like OpenSearch.

## 🛠 Core Tech Stack

* **Language:** Go (Latest Stable)
* **Concurrency:** Goroutines & Channels (Worker Pool Pattern)
* **Configuration:** YAML (Static) / Dynamic (via API or ETCD)
* **Storage/Logs:** OpenSearch (Primary backend for results and snapshots)
* **Logging:** Structured JSON logging (Optimized for machine ingestion)

## 🧠 AI Specialist Skills Required

To assist in this project, the AI should apply expertise in:

1. **Idiomatic Go Development:** Focus on "The Go Way"—simplicity, composition over inheritance, and explicit error handling.
2. **High-Concurrency Architecture:** Designing non-blocking probe engines using `select` statements, `context` cancellation, and thread-safe state management.
3. **Network Engineering:** Low-level knowledge of HTTP/2, TLS handshakes, ICMP (Raw Sockets), and gRPC health checks.
4. **DevOps & Observability:** Understanding of Prometheus metrics exporting and Grafana dashboarding for monitoring the monitor.
5. **Testing & Benchmarking:** Expertise in writing comprehensive unit tests, integration tests, and performance benchmarks to ensure reliability and efficiency.
6. **Security Best Practices:** Implementing secure coding practices, especially for network interactions and configuration management.
7. **API Design:** Crafting intuitive APIs for probe configuration, result retrieval, and alert management that align with Go's design philosophies.
8. **OpenSearch Integration:** Proficiency in designing efficient data models for storing probe results and snapshots in OpenSearch, as well as optimizing query performance.

## 🏗 Architectural Principles (Go-First)

* **Standard Project Layout:** Strictly adhere to the standard Go project layout (`cmd/` for entrypoints, `internal/` for private application code, `pkg/` for public libraries).
* **Context-Driven:** Every probe must respect a `context.Context` for timeouts and graceful cancellation.
* **Interface-Based Probing:** Define a `Prober` interface. Whether it's HTTP, TCP, or Custom, the core engine should treat them as a single type.
* **Zero-Dependency Core:** Keep the core probing logic as close to the Go Standard Library as possible. Use external packages only when they provide significant value (e.g., YAML parsing, OpenSearch clients).
* **Event-Sourced Alerts:** Probes emit events; the Alert Manager consumes them. This decoupling allows for multiple notification types (Slack, Email, Webhook) without bloating the probe logic.

## 📂 Project Structure

* `cmd/`: Contains the CLI application entrypoints (e.g., `root.go`, `config.go`).
* `internal/`: Contains private application and business logic (e.g., `config/`, `logger/`, `prober/`).
* `pkg/`: (Optional) Library code that is safe to be imported by external projects.

## 📝 Engineering Standards

* **Errors as Values:** No `panic` in production code. Use wrapped errors: `fmt.Errorf("probe failed: %w", err)`.
* **Table-Driven Tests:** All probing logic must be validated using Go's table-driven testing pattern.
* **Structured Metadata:** Every log entry must include `component`, `trace_id`, and `duration_ms`.
* **Linting & Formatting:** Code must pass `golangci-lint` and be formatted with `gofmt` or `goimports`.
* **Documentation:** All exported functions, types, and interfaces must have descriptive Godoc comments.

## 🎯 Current Objectives

1. Finalize the configuration parsing logic in `internal/config/config_parser.go`.
2. Define the `Prober` interface and implement the initial HTTP prober in `internal/prober/`.
3. Create a concurrent worker pool to manage probe execution efficiently.
4. Set up structured logging with a focus on performance and clarity.

---
This document serves as the guiding star for the `monika-go` project, ensuring that all development efforts align with the core principles of Go and the project's goals. By adhering to these guidelines, we can create a robust, efficient, and maintainable synthetic monitoring tool that stands out in the ecosystem.
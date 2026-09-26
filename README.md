# Monika Go

A PoC of [monika](https://github.com/hyperjumptech/monika) written in Go.

Reads a `monika.yaml` config, probes targets (HTTP, ping, socket, database) on a schedule, and alerts on failures (Slack, SMTP, webhook, desktop).

## Development

- Go 1.27
- [cobra](https://github.com/spf13/cobra) for the CLI
- [yaml.v3](https://gopkg.in/yaml.v3) for configuration
- [logrus](https://github.com/sirupsen/logrus) for structured logging
- standard library `testing` for table-driven tests

## Build

```console
make build
```

## Usage

```console
monika-go                       # run all probes in monika.yaml, forever
monika-go -c myconfig.yaml      # use a different config file
monika-go -i 1,3                # run only probes with these IDs
monika-go -r 5                  # run each probe 5 times, then exit
monika-go createConfig          # write an example monika.yaml
monika-go version
```

# AGENTS.md

Go Prometheus exporter for the parsedmarc PostgreSQL schema. See `README.md` for metrics and
configuration and `CONTRIBUTING.md` for the workflow.

- Layout: `cmd/exporter` (HTTP, logging, `-healthcheck`), `internal/config` (env), `internal/collector`
  (metrics; depends only on the `Store` interface), `internal/pgstore` (pgx queries).
- Metrics follow the Prometheus naming practices: counters end in `_total`, base units, one label name
  per concept (`domain`), no unbounded labels, zero-initialized known series. Update the README
  catalog and `CHANGELOG.md` whenever a metric changes.
- The database is accessed read-only; sessions force `default_transaction_read_only=on`. Never add
  write queries.
- A Go toolchain may be absent: run `gofmt`, `go vet` and `go test` in the `golang` image (see
  `CONTRIBUTING.md`).
- Logging uses `rs/zerolog` to stderr (plus `PARSEDMARC_EXPORTER_LOG_FILE` when set).

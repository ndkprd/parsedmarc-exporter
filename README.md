# parsedmarc-exporter

A [Prometheus](https://prometheus.io/) exporter for the PostgreSQL database that
[parsedmarc](https://github.com/domainaware/parsedmarc) fills with DMARC aggregate reports, DMARC
failure reports and SMTP TLS-RPT reports. It turns that data into metrics you can graph and alert
on: DMARC pass/fail volume per domain, sending sources, TLS failures and "no report received" freshness.

> **Disclaimer:** parsedmarc-exporter is an independent, community project. It is **not affiliated
> with, endorsed by, or related to** parsedmarc or its maintainers. "parsedmarc" is used only to
> describe compatibility: the exporter reads the PostgreSQL schema that parsedmarc writes and does
> not use or include parsedmarc code. Because it depends on that schema, a future parsedmarc release
> may change the tables and break this exporter. All trademarks belong to their respective owners.

## Features

- Cumulative counters for aggregate messages (by domain, disposition, DMARC/SPF/DKIM result),
  sender type, failure reports and SMTP TLS sessions; choose windows in PromQL with `increase()`.
- Freshness timestamp per report type and domain for "no reports received" alerts.
- Top sending sources per domain for a configurable window (bounded cardinality).
- Read-only database sessions (`default_transaction_read_only=on`); collection happens at scrape time.
- Exporter health metrics, a landing page at `/`, `/healthz`, and a `-healthcheck` flag.
- Single static binary, small container image, structured JSON logs.

## Quick start

The container image is published at `registry.gitlab.com/endekasoft/parsedmarc-exporter:latest`.

A complete PostgreSQL + parsedmarc + exporter stack is in
[`examples/docker-compose`](examples/docker-compose):

```bash
cd examples/docker-compose
cp .env.example .env        # set the IMAP settings and passwords
docker compose up -d          # pulls the published exporter image
# docker compose up -d --build  # or build it from this checkout instead
curl http://127.0.0.1:9190/metrics
```

Already running parsedmarc with PostgreSQL? Run only the exporter against your database:

```bash
docker run --rm -p 9190:9190 \
  -e PARSEDMARC_EXPORTER_DB_HOST=your-postgres-host \
  -e POSTGRES_USER=... -e POSTGRES_PASSWORD=... -e POSTGRES_DB=... \
  registry.gitlab.com/endekasoft/parsedmarc-exporter:latest
```

For Kubernetes with the Prometheus Operator see [`examples/kubernetes`](examples/kubernetes)
(Deployment, Service, ServiceMonitor and an optional PrometheusRule).

A ready-made Grafana dashboard is in [`examples/grafana`](examples/grafana).

The database account only needs `SELECT` on the parsedmarc tables; the exporter forces read-only
sessions either way.

## Endpoints

| Path | Purpose |
|---|---|
| `/` | landing page linking to `/metrics` |
| `/metrics` | Prometheus metrics |
| `/healthz` | liveness (`ok`); also used by `exporter -healthcheck` |

## Configuration

All settings are environment variables.

| Variable | Default | Meaning |
|---|---|---|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | required | database account and name |
| `PARSEDMARC_EXPORTER_DB_HOST` / `PARSEDMARC_EXPORTER_DB_PORT` | `postgres` / `5432` | database address |
| `PARSEDMARC_EXPORTER_LISTEN` | `:9190` | HTTP listen address |
| `PARSEDMARC_EXPORTER_SOURCES_WINDOW` | `168h` | window of `parsedmarc_top_source_messages` |
| `PARSEDMARC_EXPORTER_TOP_N` | `20` | sources kept per domain before folding the rest into `other` |
| `PARSEDMARC_EXPORTER_QUERY_TIMEOUT` | `10s` | per-group query timeout |
| `PARSEDMARC_EXPORTER_LOG_LEVEL` / `PARSEDMARC_EXPORTER_LOG_FILE` | `info` / unset | logs go to stderr, and also to the file if set |

## Metrics

Values are cumulative over all stored reports, so choose the window in PromQL
(`increase(...[7d])`). Counters move when parsedmarc **stores** a report, not when the mail was
sent: `increase(...[7d])` means "stored in the last 7 days". Domains are lowercased. Deleting stored
rows shows up as a counter reset.

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `parsedmarc_aggregate_messages_total` | counter | `domain`, `disposition`, `dmarc_passed`, `spf_aligned`, `dkim_aligned` | messages in aggregate reports; the 24 `none/quarantine/reject` x `true/false` combinations are emitted at 0 for every known domain |
| `parsedmarc_source_messages_total` | counter | `domain`, `source_type` | messages by sender type |
| `parsedmarc_failure_reports_total` | counter | `domain` | DMARC failure reports |
| `parsedmarc_smtp_tls_sessions_total` | counter | `domain`, `result` (`success`/`failure`) | TLS-RPT sessions |
| `parsedmarc_smtp_tls_failed_sessions_total` | counter | `domain`, `result_type` | failed TLS-RPT sessions by failure type |
| `parsedmarc_reports_total` | counter | `report_type` (`aggregate`/`failure`/`smtp_tls`) | stored reports |
| `parsedmarc_last_report_end_timestamp_seconds` | gauge | `report_type`, `domain` | newest report end (arrival time for failure reports) |
| `parsedmarc_top_source_messages` | gauge | `window`, `domain`, `source_name`, `source_country` | messages from the top-N sources in the last window, the rest as `other` |
| `parsedmarc_exporter_up` | gauge | | 1 if every collection group reached PostgreSQL |
| `parsedmarc_exporter_scrape_duration_seconds` | gauge | | duration of the last collection |
| `parsedmarc_exporter_scrape_errors_total` | counter | `group` | failed groups (`aggregate`, `sources`, `failure`, `smtp_tls`, `freshness`) |
| `parsedmarc_exporter_build_info` | gauge | `version`, `commit`, `go_version` | always 1 |

`parsedmarc_top_source_messages` is a deliberate exception to "export counters, not windowed
gauges": a top-N ranking changes membership over time and cannot be a counter. For per-source
detail beyond top-N, query PostgreSQL directly (for example with the Grafana PostgreSQL data source).

If a collection group fails (for example the database is down), its metrics are omitted, its
`scrape_errors_total` increments and `parsedmarc_exporter_up` is 0; the scrape itself still returns HTTP 200.

## Prometheus

```yaml
scrape_configs:
  - job_name: parsedmarc
    scrape_interval: 60s
    static_configs:
      - targets: ['parsedmarc-exporter:9190']
```

Example alerts (rate windows should be at least 4x the scrape interval):

```yaml
groups:
  - name: parsedmarc
    rules:
      - alert: ParsedmarcExporterDown
        expr: parsedmarc_exporter_up == 0 or up{job="parsedmarc"} == 0
        for: 5m
      - alert: ParsedmarcNoRecentReports
        expr: time() - max by (domain) (parsedmarc_last_report_end_timestamp_seconds{report_type="aggregate"}) > 72 * 3600
        for: 15m
      - alert: ParsedmarcDMARCFailureRatioHigh
        expr: |
          sum by (domain) (increase(parsedmarc_aggregate_messages_total{dmarc_passed="false"}[7d]))
            / sum by (domain) (increase(parsedmarc_aggregate_messages_total[7d])) > 0.05
        for: 1h
```

## Building

The Dockerfile uses public base images and the public Go module proxy by default. Override them
with build arguments, for example to build behind mirrors:

```bash
docker build \
  --build-arg GO_IMAGE=registry.example.com/golang:1.27.1 \
  --build-arg RUNTIME_IMAGE=registry.example.com/debian:trixie-20260918-slim \
  --build-arg GOPROXY=https://goproxy.example.com,direct \
  --build-arg VERSION=0.1.1 -t registry.gitlab.com/endekasoft/parsedmarc-exporter:0.1.1 .
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md). Lint the live output with `promtool`:

```bash
curl -s http://127.0.0.1:9190/metrics | promtool check metrics
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## License

[MIT](LICENSE)

# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-10-03

Changes since 0.1.0.

### Changed
- **Breaking:** the exporter's own environment variables are now prefixed with
  `PARSEDMARC_EXPORTER_`: `EXPORTER_DB_HOST`, `EXPORTER_DB_PORT`, `EXPORTER_LISTEN`,
  `EXPORTER_SOURCES_WINDOW`, `EXPORTER_TOP_N`, `EXPORTER_QUERY_TIMEOUT`, `EXPORTER_LOG_LEVEL` and
  `EXPORTER_LOG_FILE` become `PARSEDMARC_EXPORTER_<same suffix>`. The old names are no longer read.
  `POSTGRES_USER`, `POSTGRES_PASSWORD` and `POSTGRES_DB` are unchanged.
- The Go module path is now `gitlab.com/endekasoft/parsedmarc-exporter`.
- The container image is published as `registry.gitlab.com/endekasoft/parsedmarc-exporter`.
- The Dockerfile defaults to public base images and the public Go module proxy; `GO_IMAGE`,
  `RUNTIME_IMAGE` and `GOPROXY` build arguments allow mirrors.

### Added
- MIT `LICENSE`, `CONTRIBUTING.md`, `AGENTS.md`, and a README disclaimer that the project is not
  affiliated with parsedmarc.
- `examples/docker-compose` (PostgreSQL, parsedmarc and the exporter) and `examples/kubernetes`
  (Deployment, Service, ServiceMonitor, optional PrometheusRule).

### Upgrading from 0.1.0
Rename the `EXPORTER_*` variables as listed above before deploying 0.1.1; with the old names the
exporter silently falls back to the defaults.

## [0.1.0] - 2026-10-03

### Added
- Prometheus exporter for the [parsedmarc](https://github.com/domainaware/parsedmarc) PostgreSQL
  schema: DMARC aggregate and failure reports and SMTP TLS-RPT reports.
- Cumulative counters `parsedmarc_aggregate_messages_total`, `parsedmarc_source_messages_total`,
  `parsedmarc_failure_reports_total`, `parsedmarc_smtp_tls_sessions_total`,
  `parsedmarc_smtp_tls_failed_sessions_total` and `parsedmarc_reports_total`.
- `parsedmarc_last_report_end_timestamp_seconds` for freshness alerts and
  `parsedmarc_top_source_messages` for the top sending sources in a window.
- Exporter health metrics: `parsedmarc_exporter_up`, `parsedmarc_exporter_scrape_duration_seconds`,
  `parsedmarc_exporter_scrape_errors_total` and `parsedmarc_exporter_build_info`.
- Read-only database sessions (`default_transaction_read_only=on`), scrape-time collection with
  per-group timeouts, and zero-initialized label combinations.
- Landing page at `/`, `/healthz`, and a `-healthcheck` flag for container healthchecks.
- Multi-stage Dockerfile, Docker Compose and Kubernetes (ServiceMonitor) examples.

[Unreleased]: https://gitlab.com/endekasoft/parsedmarc-exporter/-/compare/v0.1.1...HEAD
[0.1.1]: https://gitlab.com/endekasoft/parsedmarc-exporter/-/compare/v0.1.0...v0.1.1
[0.1.0]: https://gitlab.com/endekasoft/parsedmarc-exporter/-/tags/v0.1.0

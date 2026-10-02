# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project
adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://gitlab.com/endekasoft/parsedmarc-exporter/-/compare/v0.1.0...HEAD
[0.1.0]: https://gitlab.com/endekasoft/parsedmarc-exporter/-/tags/v0.1.0

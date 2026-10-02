# Contributing

Thanks for helping improve parsedmarc-exporter. Bug reports, metric ideas and pull requests are
welcome.

## Reporting issues

Include the exporter version (`parsedmarc_exporter_build_info`), the parsedmarc version that wrote
the database, the relevant `/metrics` output and the exporter logs. Never paste real report
contents, credentials or customer domains you do not want public.

## Development setup

You need Docker; a local Go toolchain is optional. Without one, run Go in the builder image:

```bash
docker run --rm -v "$PWD":/src -w /src -e HOME=/tmp -e GOCACHE=/tmp/gocache -e GOMODCACHE=/tmp/gomod \
  --user "$(id -u):$(id -g)" golang:1.27.1 sh -c 'gofmt -l . && go vet ./... && go test ./...'
```

With Go installed: `gofmt -l . && go vet ./... && go test ./...`.

Layout:

- `cmd/exporter`: HTTP server, logging, `-healthcheck`
- `internal/config`: environment configuration
- `internal/collector`: metrics; depends only on the `Store` interface, so it is tested with fakes
- `internal/pgstore`: PostgreSQL queries (read-only)

## Pull requests

1. Open an issue first for new metrics or behavior changes.
2. Keep changes focused; add or update tests. The collector tests use a fake `Store`.
3. `gofmt`, `go vet` and `go test ./...` must pass.
4. Update `README.md` (metric catalog) and `CHANGELOG.md` (under `[Unreleased]`).
5. Use [Conventional Commits](https://www.conventionalcommits.org/) for commit messages
   (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`).

## Metric guidelines

Metrics follow the [Prometheus naming and instrumentation practices](https://prometheus.io/docs/practices/naming/):

- Counters end in `_total`; durations use `_seconds`, sizes `_bytes`; use base units.
- Export cumulative counters and let PromQL compute windows. A windowed gauge needs a written
  justification in the PR and in the metric's HELP text.
- One label name per concept across all metrics (`domain`), no unbounded label values (no IP
  addresses, report IDs, mailbox addresses or sample content), and a documented series budget.
- Emit known label combinations at 0 so `rate()` and alerts see them.
- Check the output with `promtool check metrics`.

## Database access

The exporter must stay read-only. Do not add queries that write, and keep the
`default_transaction_read_only=on` session parameter.

## License

By contributing you agree that your contributions are licensed under the [MIT License](LICENSE).

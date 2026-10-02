# Docker Compose quick start

A minimal stack: PostgreSQL, parsedmarc and parsedmarc-exporter.

```bash
cd examples/docker-compose
cp .env.example .env        # set the IMAP settings and passwords
docker compose up -d          # pulls the published exporter image
# docker compose up -d --build  # or build it from this checkout instead
curl http://127.0.0.1:9190/metrics
```

Open <http://127.0.0.1:9190/> for the landing page. Metrics for report types you have not received
yet are absent or zero until parsedmarc stores the first report.

Point Prometheus at the exporter, for example:

```yaml
scrape_configs:
  - job_name: parsedmarc
    scrape_interval: 60s
    static_configs:
      - targets: ['<docker-host>:9190']
```

To scrape over a Docker network instead of a host port, put Prometheus and the exporter on the same
network and use `exporter:9190`.

Already running parsedmarc with PostgreSQL? Add only the `exporter` service and point
`EXPORTER_DB_HOST` and the `POSTGRES_*` variables at your existing database.

The image tag `ghcr.io/domainaware/parsedmarc` and the PostgreSQL version are pinned for
reproducibility; check upstream for newer releases. For other mailbox types (Microsoft Graph,
Gmail API, Maildir) see the parsedmarc documentation; its settings map to
`PARSEDMARC_<SECTION>_<KEY>` environment variables.

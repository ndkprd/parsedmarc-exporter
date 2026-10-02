# Grafana dashboard example

`parsedmarc-overview.json` is a ready-to-import Grafana dashboard for the metrics of
parsedmarc-exporter. It answers one question: *are our domains passing DMARC, is anything being
quarantined or rejected, and are reports still arriving?*

Colours: **blue** = good or neutral, **yellow** = warning, **orange** = bad.

## Import

1. Grafana: **Dashboards > New > Import**, upload `parsedmarc-overview.json`.
2. Pick your Prometheus data source (variable `PROMETHEUS_DS`).
3. Choose the scrape job of the exporter in the **Job** variable (default `parsedmarc`; see the
   Prometheus scrape config in the main README). Use **Domain** to filter.

Provisioning also works: copy the file into a provisioned dashboards folder. Tested with
Grafana 13.

## Layout

```
┌ DMARC pass rate ┬ Messages ┬ Quarantined/rejected ┬ Stalest domain (h) ┬ Exporter ┐  summary
├─────────────────┴──┬───────┴──────────┬───────────┴────────────────────┴──────────┤
│ Messages per day   │ Pass rate by     │ Disposition over time                      │  trends
│ pass vs fail       │ domain           │ (none / quarantine / reject)               │
├────────────────────┴──────────────────┴────────────────────────────────────────────┤
│ ▸ Senders          SPF/DKIM alignment · sender type · failure reports · top sources │  details
│ ▸ Reports and TLS  TLS success/failure · failure types · time since newest report   │  (collapsed)
│ ▸ Exporter health  scrape duration · scrape errors                                  │
└──────────────────────────────────────────────────────────────────────────────────────┘
```

## Notes

- Metrics are cumulative counters that change only when parsedmarc stores a batch of reports, so
  range statistics use `increase(...[$__range])` and trends use a rolling `1d` window instead of
  `$__rate_interval`. The default time range is 30 days and the refresh is 5 minutes because the
  data changes at most daily.
- Counters need history: with less than a few days of data in Prometheus the range panels
  show small or empty values.
- The top-sources table uses the exporter's own window (default 7 days) and ignores the dashboard
  time range.
- The `job` filter is on every query and `PROMETHEUS_DS` is a variable, so the file can be imported
  into any Grafana without editing UIDs.
- Thresholds: pass rate yellow below 98% and orange below 90%; quarantined or rejected share
  yellow above 0 and orange above 1%; report age yellow above 48 h and orange above 72 h. Edit them
  in the panel field settings to match your policy.

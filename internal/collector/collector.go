// Package collector exposes the parsedmarc database as Prometheus metrics.
package collector

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

// Report types used as the report_type label value.
const (
	reportAggregate = "aggregate"
	reportFailure   = "failure"
	reportSMTPTLS   = "smtp_tls"
)

// Collection groups; also the values of the scrape_errors_total group label.
const (
	groupAggregate = "aggregate"
	groupSources   = "sources"
	groupFailure   = "failure"
	groupSMTPTLS   = "smtp_tls"
	groupFreshness = "freshness"
)

var groupNames = []string{groupAggregate, groupSources, groupFailure, groupSMTPTLS, groupFreshness}

func desc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc("parsedmarc_"+name, help, labels, nil)
}

var (
	aggregateMessages = desc("aggregate_messages_total",
		"Messages in stored DMARC aggregate reports (dmarc_aggregate_record), cumulative over all stored reports.",
		"domain", "disposition", "dmarc_passed", "spf_aligned", "dkim_aligned")
	sourceMessages = desc("source_messages_total",
		"Messages in stored DMARC aggregate reports by sender type, cumulative over all stored reports.",
		"domain", "source_type")
	failureReports = desc("failure_reports_total",
		"Stored DMARC failure reports (dmarc_failure_report), cumulative.", "domain")
	tlsSessions = desc("smtp_tls_sessions_total",
		"SMTP TLS-RPT sessions in stored reports (smtp_tls_policy), cumulative.", "domain", "result")
	tlsFailedSessions = desc("smtp_tls_failed_sessions_total",
		"Failed SMTP TLS-RPT sessions by failure type (smtp_tls_failure_detail), cumulative.", "domain", "result_type")
	reportsTotal = desc("reports_total",
		"Stored reports by type (aggregate, failure, smtp_tls), cumulative.", "report_type")
	lastReportEnd = desc("last_report_end_timestamp_seconds",
		"Unix time of the newest stored report end (arrival time for failure reports).", "report_type", "domain")
	topSourceMessages = desc("top_source_messages",
		"Messages from the top sources per domain within the last window, by report end date. "+
			"A ranking cannot be a counter; this gauge is a documented exception to the cumulative-counter rule.",
		"window", "domain", "source_name", "source_country")

	exporterUp = desc("exporter_up",
		"1 if every collection group reached PostgreSQL on the last scrape.")
	exporterDuration = desc("exporter_scrape_duration_seconds",
		"Duration of the last collection in seconds.")
	exporterErrors = desc("exporter_scrape_errors_total",
		"Failed collection groups since the exporter started.", "group")
	exporterBuild = desc("exporter_build_info",
		"Exporter build information; the value is always 1.", "version", "commit", "go_version")
)

// Options configures a Collector.
type Options struct {
	Timeout       time.Duration
	SourcesWindow time.Duration
	TopN          int
	Version       string
	Commit        string
}

// Collector implements prometheus.Collector. Every scrape queries the store
// and builds fresh const metrics; nothing runs in the background.
type Collector struct {
	store Store
	opts  Options
	log   zerolog.Logger

	mu     sync.Mutex
	errors map[string]float64
}

// New returns a Collector reading from store.
func New(store Store, opts Options, log zerolog.Logger) *Collector {
	errs := make(map[string]float64, len(groupNames))
	for _, g := range groupNames {
		errs[g] = 0
	}
	return &Collector{store: store, opts: opts, log: log, errors: errs}
}

// Describe sends nothing, which registers the collector as unchecked: the
// label sets depend on the data, so descriptors are not fixed up front.
func (c *Collector) Describe(chan<- *prometheus.Desc) {}

// Collect runs every group and emits the exporter health metrics.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	groups := []struct {
		name string
		fn   func(context.Context) ([]prometheus.Metric, error)
	}{
		{groupAggregate, c.aggregate},
		{groupSources, c.sources},
		{groupFailure, c.failure},
		{groupSMTPTLS, c.smtpTLS},
		{groupFreshness, c.freshness},
	}

	up := 1.0
	for _, g := range groups {
		ctx, cancel := context.WithTimeout(context.Background(), c.opts.Timeout)
		metrics, err := g.fn(ctx)
		cancel()
		if err != nil {
			up = 0
			c.countError(g.name)
			c.log.Error().Err(err).Str("group", g.name).Msg("collection failed")
			continue
		}
		for _, m := range metrics {
			ch <- m
		}
	}

	ch <- prometheus.MustNewConstMetric(exporterUp, prometheus.GaugeValue, up)
	ch <- prometheus.MustNewConstMetric(exporterDuration, prometheus.GaugeValue, time.Since(start).Seconds())
	ch <- prometheus.MustNewConstMetric(exporterBuild, prometheus.GaugeValue, 1,
		c.opts.Version, c.opts.Commit, runtime.Version())
	c.mu.Lock()
	for _, g := range groupNames {
		ch <- prometheus.MustNewConstMetric(exporterErrors, prometheus.CounterValue, c.errors[g], g)
	}
	c.mu.Unlock()
}

func (c *Collector) countError(group string) {
	c.mu.Lock()
	c.errors[group]++
	c.mu.Unlock()
}

func counter(d *prometheus.Desc, v int64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d, prometheus.CounterValue, float64(v), labels...)
}

func gauge(d *prometheus.Desc, v float64, labels ...string) prometheus.Metric {
	return prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
}

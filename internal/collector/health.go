package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

// freshness emits stored report counts and the newest report time per type
// and domain. Report types with no rows still get a zero count.
func (c *Collector) freshness(ctx context.Context) ([]prometheus.Metric, error) {
	counts, err := c.store.ReportCounts(ctx)
	if err != nil {
		return nil, err
	}
	last, err := c.store.LastReports(ctx)
	if err != nil {
		return nil, err
	}
	metrics := make([]prometheus.Metric, 0, 3+len(last))
	for _, t := range []string{reportAggregate, reportFailure, reportSMTPTLS} {
		metrics = append(metrics, counter(reportsTotal, counts[t], t))
	}
	for _, l := range last {
		metrics = append(metrics, gauge(lastReportEnd, float64(l.At.Unix()), l.ReportType, l.Domain))
	}
	return metrics, nil
}

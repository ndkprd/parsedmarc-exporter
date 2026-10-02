package collector

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
)

func (c *Collector) failure(ctx context.Context) ([]prometheus.Metric, error) {
	rows, err := c.store.FailureTotals(ctx)
	if err != nil {
		return nil, err
	}
	metrics := make([]prometheus.Metric, 0, len(rows))
	for _, r := range rows {
		metrics = append(metrics, counter(failureReports, r.Count, r.Domain))
	}
	return metrics, nil
}

// smtpTLS emits session totals (both result values per known domain) and
// failed sessions by failure type.
func (c *Collector) smtpTLS(ctx context.Context) ([]prometheus.Metric, error) {
	sessions, err := c.store.TLSSessionTotals(ctx)
	if err != nil {
		return nil, err
	}
	failures, err := c.store.TLSFailureTotals(ctx)
	if err != nil {
		return nil, err
	}
	metrics := make([]prometheus.Metric, 0, 2*len(sessions)+len(failures))
	for _, s := range sessions {
		metrics = append(metrics,
			counter(tlsSessions, s.Successful, s.Domain, "success"),
			counter(tlsSessions, s.Failed, s.Domain, "failure"))
	}
	for _, f := range failures {
		metrics = append(metrics, counter(tlsFailedSessions, f.Sessions, f.Domain, f.ResultType))
	}
	return metrics, nil
}

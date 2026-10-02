package collector

import (
	"context"
	"sort"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	dispositions = []string{"none", "quarantine", "reject"}
	bools        = []string{"true", "false"}
)

// aggregate emits message totals per label combination. For every known
// domain the full disposition x dmarc_passed x spf x dkim cross-product is
// emitted, with 0 where no rows exist, so rate() and alerts see the series.
func (c *Collector) aggregate(ctx context.Context) ([]prometheus.Metric, error) {
	rows, err := c.store.AggregateTotals(ctx)
	if err != nil {
		return nil, err
	}
	sources, err := c.store.SourceTypeTotals(ctx)
	if err != nil {
		return nil, err
	}

	type key struct{ domain, disposition, dmarc, spf, dkim string }
	totals := make(map[key]int64, len(rows))
	domains := map[string]bool{}
	for _, r := range rows {
		totals[key{r.Domain, r.Disposition, r.DMARCPassed, r.SPFAligned, r.DKIMAligned}] += r.Messages
		domains[r.Domain] = true
	}
	for _, s := range sources {
		domains[s.Domain] = true
	}
	for d := range domains {
		for _, disp := range dispositions {
			for _, dmarc := range bools {
				for _, spf := range bools {
					for _, dkim := range bools {
						k := key{d, disp, dmarc, spf, dkim}
						totals[k] += 0 // creates the series if absent
					}
				}
			}
		}
	}

	metrics := make([]prometheus.Metric, 0, len(totals)+len(sources))
	for k, n := range totals {
		metrics = append(metrics, counter(aggregateMessages, n, k.domain, k.disposition, k.dmarc, k.spf, k.dkim))
	}
	for _, s := range sources {
		metrics = append(metrics, counter(sourceMessages, s.Messages, s.Domain, s.SourceType))
	}
	return metrics, nil
}

// sortedKeys returns the keys of m in order, for deterministic output.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

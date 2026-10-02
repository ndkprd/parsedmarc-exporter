package collector

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const otherSource = "other"

// sources emits the top-N sources per domain for the configured window and
// folds the rest into source_name="other".
func (c *Collector) sources(ctx context.Context) ([]prometheus.Metric, error) {
	rows, err := c.store.SourceWindowTotals(ctx, c.opts.SourcesWindow)
	if err != nil {
		return nil, err
	}
	window := formatWindow(c.opts.SourcesWindow)

	byDomain := map[string][]SourceRow{}
	for _, r := range rows {
		byDomain[r.Domain] = append(byDomain[r.Domain], r)
	}

	var metrics []prometheus.Metric
	for _, domain := range sortedKeys(byDomain) {
		list := byDomain[domain]
		sort.Slice(list, func(i, j int) bool {
			if list[i].Messages != list[j].Messages {
				return list[i].Messages > list[j].Messages
			}
			return list[i].Name < list[j].Name
		})
		var rest int64
		for i, r := range list {
			if i >= c.opts.TopN {
				rest += r.Messages
				continue
			}
			metrics = append(metrics, gauge(topSourceMessages, float64(r.Messages), window, domain, r.Name, r.Country))
		}
		if rest > 0 {
			metrics = append(metrics, gauge(topSourceMessages, float64(rest), window, domain, otherSource, otherSource))
		}
	}
	return metrics, nil
}

// formatWindow renders a duration as a Prometheus-style window label (7d, 36h).
func formatWindow(d time.Duration) string {
	if d%(24*time.Hour) == 0 {
		return strconv.FormatInt(int64(d/(24*time.Hour)), 10) + "d"
	}
	return d.String()
}

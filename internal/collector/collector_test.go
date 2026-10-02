package collector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
)

type fakeStore struct {
	agg      []AggregateRow
	types    []SourceTypeRow
	sources  []SourceRow
	failures []DomainCount
	sessions []TLSSessionRow
	tlsFail  []TLSFailureRow
	counts   map[string]int64
	last     []LastReport
	err      map[string]error // keyed by group name
}

func (f *fakeStore) AggregateTotals(context.Context) ([]AggregateRow, error) {
	return f.agg, f.err[groupAggregate]
}
func (f *fakeStore) SourceTypeTotals(context.Context) ([]SourceTypeRow, error) { return f.types, nil }
func (f *fakeStore) SourceWindowTotals(context.Context, time.Duration) ([]SourceRow, error) {
	return f.sources, f.err[groupSources]
}
func (f *fakeStore) FailureTotals(context.Context) ([]DomainCount, error) {
	return f.failures, f.err[groupFailure]
}
func (f *fakeStore) TLSSessionTotals(context.Context) ([]TLSSessionRow, error) {
	return f.sessions, f.err[groupSMTPTLS]
}
func (f *fakeStore) TLSFailureTotals(context.Context) ([]TLSFailureRow, error) { return f.tlsFail, nil }
func (f *fakeStore) ReportCounts(context.Context) (map[string]int64, error) {
	return f.counts, f.err[groupFreshness]
}
func (f *fakeStore) LastReports(context.Context) ([]LastReport, error) { return f.last, nil }

func gather(t *testing.T, s Store) map[string]*dto.MetricFamily {
	t.Helper()
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(New(s, Options{Timeout: time.Second, SourcesWindow: 7 * 24 * time.Hour, TopN: 2, Version: "test", Commit: "abc"}, zerolog.Nop()))
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*dto.MetricFamily{}
	for _, f := range families {
		out[f.GetName()] = f
	}
	return out
}

// find returns the sample value whose labels contain every wanted pair.
func find(f *dto.MetricFamily, want map[string]string) (float64, bool) {
	for _, m := range f.GetMetric() {
		got := map[string]string{}
		for _, l := range m.GetLabel() {
			got[l.GetName()] = l.GetValue()
		}
		match := true
		for k, v := range want {
			match = match && got[k] == v
		}
		if match {
			if m.Counter != nil {
				return m.GetCounter().GetValue(), true
			}
			return m.GetGauge().GetValue(), true
		}
	}
	return 0, false
}

func mustValue(t *testing.T, fam map[string]*dto.MetricFamily, name string, labels map[string]string, want float64) {
	t.Helper()
	f, ok := fam[name]
	if !ok {
		t.Fatalf("metric %s missing", name)
	}
	got, ok := find(f, labels)
	if !ok || got != want {
		t.Errorf("%s%v = %v (found=%v), want %v", name, labels, got, ok, want)
	}
}

func TestAggregateCountersAndZeroInit(t *testing.T) {
	fam := gather(t, &fakeStore{
		agg: []AggregateRow{
			{"example.com", "none", "true", "true", "false", 171},
			{"example.com", "none", "true", "true", "false", 9}, // same labels from another source row
			{"example.com", "quarantine", "false", "false", "false", 12},
		},
		types: []SourceTypeRow{{"example.com", "Email Provider", 183}, {"example.com", "unknown", 9}},
	})
	mustValue(t, fam, "parsedmarc_aggregate_messages_total",
		map[string]string{"domain": "example.com", "disposition": "none", "dmarc_passed": "true", "spf_aligned": "true", "dkim_aligned": "false"}, 180)
	// A combination without rows is emitted as 0.
	mustValue(t, fam, "parsedmarc_aggregate_messages_total",
		map[string]string{"domain": "example.com", "disposition": "reject", "dmarc_passed": "false", "spf_aligned": "false", "dkim_aligned": "false"}, 0)
	if n := len(fam["parsedmarc_aggregate_messages_total"].GetMetric()); n != 24 {
		t.Errorf("series = %d, want the 24-combination cross-product", n)
	}
	mustValue(t, fam, "parsedmarc_source_messages_total", map[string]string{"source_type": "Email Provider"}, 183)
}

func TestTopSourcesFoldsRemainderIntoOther(t *testing.T) {
	fam := gather(t, &fakeStore{sources: []SourceRow{
		{"example.com", "google.com", "US", 100},
		{"example.com", "microsoft.com", "US", 50},
		{"example.com", "a.example", "ID", 5},
		{"example.com", "b.example", "ID", 3},
	}})
	f := fam["parsedmarc_top_source_messages"]
	if n := len(f.GetMetric()); n != 3 { // TopN=2 plus "other"
		t.Fatalf("series = %d, want 3", n)
	}
	mustValue(t, fam, "parsedmarc_top_source_messages", map[string]string{"source_name": "google.com", "window": "7d"}, 100)
	mustValue(t, fam, "parsedmarc_top_source_messages", map[string]string{"source_name": "other"}, 8)
}

func TestSMTPTLSAndFailureCounters(t *testing.T) {
	fam := gather(t, &fakeStore{
		failures: []DomainCount{{"example.com", 1}},
		sessions: []TLSSessionRow{{"example.com", 90, 10}},
		tlsFail:  []TLSFailureRow{{"example.com", "certificate-expired", 4}},
	})
	mustValue(t, fam, "parsedmarc_failure_reports_total", map[string]string{"domain": "example.com"}, 1)
	mustValue(t, fam, "parsedmarc_smtp_tls_sessions_total", map[string]string{"result": "success"}, 90)
	mustValue(t, fam, "parsedmarc_smtp_tls_sessions_total", map[string]string{"result": "failure"}, 10)
	mustValue(t, fam, "parsedmarc_smtp_tls_failed_sessions_total", map[string]string{"result_type": "certificate-expired"}, 4)
}

func TestFreshnessAndHealth(t *testing.T) {
	at := time.Date(2026, 10, 1, 23, 59, 59, 0, time.UTC)
	fam := gather(t, &fakeStore{
		counts: map[string]int64{"aggregate": 18},
		last:   []LastReport{{"aggregate", "example.com", at}},
	})
	mustValue(t, fam, "parsedmarc_reports_total", map[string]string{"report_type": "aggregate"}, 18)
	mustValue(t, fam, "parsedmarc_reports_total", map[string]string{"report_type": "smtp_tls"}, 0)
	mustValue(t, fam, "parsedmarc_last_report_end_timestamp_seconds", map[string]string{"domain": "example.com"}, float64(at.Unix()))
	mustValue(t, fam, "parsedmarc_exporter_up", nil, 1)
	mustValue(t, fam, "parsedmarc_exporter_build_info", map[string]string{"version": "test", "commit": "abc"}, 1)
	if n := len(fam["parsedmarc_exporter_scrape_errors_total"].GetMetric()); n != len(groupNames) {
		t.Errorf("error counters = %d, want one per group (%d)", n, len(groupNames))
	}
}

func TestDatabaseFailureReportsDownAndKeepsScraping(t *testing.T) {
	store := &fakeStore{
		failures: []DomainCount{{"example.com", 1}},
		err:      map[string]error{groupAggregate: errors.New("connection refused")},
	}
	fam := gather(t, store)
	mustValue(t, fam, "parsedmarc_exporter_up", nil, 0)
	mustValue(t, fam, "parsedmarc_exporter_scrape_errors_total", map[string]string{"group": "aggregate"}, 1)
	mustValue(t, fam, "parsedmarc_exporter_scrape_errors_total", map[string]string{"group": "failure"}, 0)
	if _, ok := fam["parsedmarc_aggregate_messages_total"]; ok {
		t.Error("failed group must not emit partial metrics")
	}
	mustValue(t, fam, "parsedmarc_failure_reports_total", map[string]string{"domain": "example.com"}, 1)
}

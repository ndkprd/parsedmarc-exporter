package collector

import (
	"context"
	"time"
)

// Store is the read-only view of the parsedmarc database the collector needs.
// Implementations normalize NULL labels to "unknown".
type Store interface {
	AggregateTotals(ctx context.Context) ([]AggregateRow, error)
	SourceTypeTotals(ctx context.Context) ([]SourceTypeRow, error)
	// SourceWindowTotals returns messages per source for reports ending within window.
	SourceWindowTotals(ctx context.Context, window time.Duration) ([]SourceRow, error)
	FailureTotals(ctx context.Context) ([]DomainCount, error)
	TLSSessionTotals(ctx context.Context) ([]TLSSessionRow, error)
	TLSFailureTotals(ctx context.Context) ([]TLSFailureRow, error)
	ReportCounts(ctx context.Context) (map[string]int64, error)
	LastReports(ctx context.Context) ([]LastReport, error)
}

// AggregateRow is the message total for one label combination.
type AggregateRow struct {
	Domain, Disposition, DMARCPassed, SPFAligned, DKIMAligned string
	Messages                                                  int64
}

// SourceTypeRow is the message total for one domain and sender type.
type SourceTypeRow struct {
	Domain, SourceType string
	Messages           int64
}

// SourceRow is the message total for one domain and sending source.
type SourceRow struct {
	Domain, Name, Country string
	Messages              int64
}

// DomainCount is a count for one domain.
type DomainCount struct {
	Domain string
	Count  int64
}

// TLSSessionRow holds the SMTP-TLS session totals for one domain.
type TLSSessionRow struct {
	Domain             string
	Successful, Failed int64
}

// TLSFailureRow holds failed SMTP-TLS sessions by failure type.
type TLSFailureRow struct {
	Domain, ResultType string
	Sessions           int64
}

// LastReport is the newest report time for one report type and domain.
type LastReport struct {
	ReportType, Domain string
	At                 time.Time
}

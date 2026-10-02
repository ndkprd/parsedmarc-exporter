// Package pgstore reads the parsedmarc PostgreSQL schema.
package pgstore

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gitlab.com/endekasoft/parsedmarc-exporter/internal/collector"
	"gitlab.com/endekasoft/parsedmarc-exporter/internal/config"
)

// Store implements collector.Store on a pgx pool. Domains are lowercased so
// the same domain never splits across label values.
type Store struct{ pool *pgxpool.Pool }

var _ collector.Store = (*Store)(nil)

// PoolConfig builds the pool settings. Every session is read-only, so the
// exporter cannot write even though it reuses the parsedmarc account.
func PoolConfig(c config.Config) (*pgxpool.Config, error) {
	pc, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}
	pc.ConnConfig.Host = c.DBHost
	pc.ConnConfig.Port = c.DBPort
	pc.ConnConfig.User = c.DBUser
	pc.ConnConfig.Password = c.DBPassword
	pc.ConnConfig.Database = c.DBName
	pc.ConnConfig.RuntimeParams["default_transaction_read_only"] = "on"
	pc.ConnConfig.RuntimeParams["application_name"] = "parsedmarc-exporter"
	pc.MaxConns = 2
	return pc, nil
}

// New connects lazily; a database that is down at startup only affects
// scrapes (parsedmarc_exporter_up 0), not the exporter process.
func New(ctx context.Context, c config.Config) (*Store, error) {
	pc, err := PoolConfig(c)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() { s.pool.Close() }

func query[T any](ctx context.Context, s *Store, sql string, scan func(pgx.Rows) (T, error), args ...any) ([]T, error) {
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []T
	for rows.Next() {
		v, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) AggregateTotals(ctx context.Context) ([]collector.AggregateRow, error) {
	return query(ctx, s, `
		SELECT lower(r.domain), coalesce(rec.disposition, 'unknown'),
		       coalesce(rec.dmarc_passed::text, 'unknown'),
		       coalesce(rec.spf_aligned::text, 'unknown'),
		       coalesce(rec.dkim_aligned::text, 'unknown'),
		       sum(rec.message_count)::bigint
		FROM dmarc_aggregate_record rec
		JOIN dmarc_aggregate_report r ON r.id = rec.report_id
		GROUP BY 1, 2, 3, 4, 5`,
		func(r pgx.Rows) (v collector.AggregateRow, err error) {
			err = r.Scan(&v.Domain, &v.Disposition, &v.DMARCPassed, &v.SPFAligned, &v.DKIMAligned, &v.Messages)
			return
		})
}

func (s *Store) SourceTypeTotals(ctx context.Context) ([]collector.SourceTypeRow, error) {
	return query(ctx, s, `
		SELECT lower(r.domain), coalesce(rec.source_type, 'unknown'), sum(rec.message_count)::bigint
		FROM dmarc_aggregate_record rec
		JOIN dmarc_aggregate_report r ON r.id = rec.report_id
		GROUP BY 1, 2`,
		func(r pgx.Rows) (v collector.SourceTypeRow, err error) {
			err = r.Scan(&v.Domain, &v.SourceType, &v.Messages)
			return
		})
}

func (s *Store) SourceWindowTotals(ctx context.Context, window time.Duration) ([]collector.SourceRow, error) {
	return query(ctx, s, `
		SELECT lower(r.domain), coalesce(rec.source_name, 'unknown'),
		       coalesce(rec.source_country, 'unknown'), sum(rec.message_count)::bigint
		FROM dmarc_aggregate_record rec
		JOIN dmarc_aggregate_report r ON r.id = rec.report_id
		WHERE r.end_date >= now() - make_interval(secs => $1)
		GROUP BY 1, 2, 3`,
		func(r pgx.Rows) (v collector.SourceRow, err error) {
			err = r.Scan(&v.Domain, &v.Name, &v.Country, &v.Messages)
			return
		}, window.Seconds())
}

func (s *Store) FailureTotals(ctx context.Context) ([]collector.DomainCount, error) {
	return query(ctx, s, `
		SELECT lower(coalesce(reported_domain, 'unknown')), count(*)::bigint
		FROM dmarc_failure_report GROUP BY 1`,
		func(r pgx.Rows) (v collector.DomainCount, err error) {
			err = r.Scan(&v.Domain, &v.Count)
			return
		})
}

func (s *Store) TLSSessionTotals(ctx context.Context) ([]collector.TLSSessionRow, error) {
	return query(ctx, s, `
		SELECT lower(coalesce(policy_domain, 'unknown')),
		       coalesce(sum(successful_session_count), 0)::bigint,
		       coalesce(sum(failed_session_count), 0)::bigint
		FROM smtp_tls_policy GROUP BY 1`,
		func(r pgx.Rows) (v collector.TLSSessionRow, err error) {
			err = r.Scan(&v.Domain, &v.Successful, &v.Failed)
			return
		})
}

func (s *Store) TLSFailureTotals(ctx context.Context) ([]collector.TLSFailureRow, error) {
	return query(ctx, s, `
		SELECT lower(coalesce(p.policy_domain, 'unknown')), coalesce(d.result_type, 'unknown'),
		       coalesce(sum(d.failed_session_count), 0)::bigint
		FROM smtp_tls_failure_detail d
		JOIN smtp_tls_policy p ON p.id = d.policy_id
		GROUP BY 1, 2`,
		func(r pgx.Rows) (v collector.TLSFailureRow, err error) {
			err = r.Scan(&v.Domain, &v.ResultType, &v.Sessions)
			return
		})
}

func (s *Store) ReportCounts(ctx context.Context) (map[string]int64, error) {
	type kv struct {
		k string
		n int64
	}
	rows, err := query(ctx, s, `
		SELECT 'aggregate', count(*)::bigint FROM dmarc_aggregate_report
		UNION ALL SELECT 'failure', count(*)::bigint FROM dmarc_failure_report
		UNION ALL SELECT 'smtp_tls', count(*)::bigint FROM smtp_tls_report`,
		func(r pgx.Rows) (v kv, err error) {
			err = r.Scan(&v.k, &v.n)
			return
		})
	counts := make(map[string]int64, len(rows))
	for _, r := range rows {
		counts[r.k] = r.n
	}
	return counts, err
}

func (s *Store) LastReports(ctx context.Context) ([]collector.LastReport, error) {
	return query(ctx, s, `
		SELECT 'aggregate', lower(domain), max(end_date) FROM dmarc_aggregate_report GROUP BY 2
		UNION ALL
		SELECT 'failure', lower(coalesce(reported_domain, 'unknown')), max(arrival_date_utc)
		FROM dmarc_failure_report WHERE arrival_date_utc IS NOT NULL GROUP BY 2
		UNION ALL
		SELECT 'smtp_tls', lower(coalesce(p.policy_domain, 'unknown')), max(r.end_date)
		FROM smtp_tls_report r JOIN smtp_tls_policy p ON p.report_id = r.id GROUP BY 2`,
		func(r pgx.Rows) (v collector.LastReport, err error) {
			err = r.Scan(&v.ReportType, &v.Domain, &v.At)
			return
		})
}

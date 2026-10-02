package pgstore

import (
	"testing"

	"gitlab.com/endekasoft/parsedmarc-exporter/internal/config"
)

func TestPoolConfigIsReadOnly(t *testing.T) {
	pc, err := PoolConfig(config.Config{DBHost: "postgres", DBPort: 5432, DBUser: "u", DBPassword: "p w'd", DBName: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if got := pc.ConnConfig.RuntimeParams["default_transaction_read_only"]; got != "on" {
		t.Errorf("default_transaction_read_only = %q, want on", got)
	}
	if pc.ConnConfig.Password != "p w'd" || pc.ConnConfig.Host != "postgres" {
		t.Errorf("connection fields not applied: %+v", pc.ConnConfig)
	}
}

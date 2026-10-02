package config

import (
	"strings"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

var base = map[string]string{"POSTGRES_USER": "u", "POSTGRES_PASSWORD": "p", "POSTGRES_DB": "d"}

func with(extra map[string]string) map[string]string {
	m := map[string]string{}
	for k, v := range base {
		m[k] = v
	}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func TestLoadDefaults(t *testing.T) {
	c, err := Load(env(base))
	if err != nil {
		t.Fatal(err)
	}
	if c.DBHost != "postgres" || c.DBPort != 5432 || c.Listen != ":9190" ||
		c.SourcesWindow != 7*24*time.Hour || c.TopN != 20 || c.QueryTimeout != 10*time.Second {
		t.Errorf("unexpected defaults: %+v", c)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := map[string]map[string]string{
		"PARSEDMARC_EXPORTER_SOURCES_WINDOW": {"PARSEDMARC_EXPORTER_SOURCES_WINDOW": "soon"},
		"PARSEDMARC_EXPORTER_QUERY_TIMEOUT":  {"PARSEDMARC_EXPORTER_QUERY_TIMEOUT": "-1s"},
		"PARSEDMARC_EXPORTER_TOP_N":          {"PARSEDMARC_EXPORTER_TOP_N": "0"},
		"PARSEDMARC_EXPORTER_DB_PORT":        {"PARSEDMARC_EXPORTER_DB_PORT": "99999"},
	}
	for want, extra := range tests {
		if _, err := Load(env(with(extra))); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("want error mentioning %s, got %v", want, err)
		}
	}
}

func TestLoadRequiresDatabaseCredentials(t *testing.T) {
	_, err := Load(env(nil))
	for _, key := range []string{"POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB"} {
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Errorf("want error mentioning %s, got %v", key, err)
		}
	}
}

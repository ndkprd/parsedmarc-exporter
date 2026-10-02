// Package config loads the exporter settings from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

// Config holds every setting the exporter needs.
type Config struct {
	DBHost        string
	DBPort        uint16
	DBUser        string
	DBPassword    string
	DBName        string
	Listen        string
	SourcesWindow time.Duration
	TopN          int
	QueryTimeout  time.Duration
	LogLevel      string
	LogFile       string
}

// Load reads the configuration through getenv (os.Getenv in production).
// POSTGRES_USER, POSTGRES_PASSWORD and POSTGRES_DB are reused from the
// parsedmarc compose environment.
func Load(getenv func(string) string) (Config, error) {
	get := func(key, def string) string {
		if v := getenv(key); v != "" {
			return v
		}
		return def
	}
	c := Config{
		DBHost:     get("EXPORTER_DB_HOST", "postgres"),
		DBUser:     getenv("POSTGRES_USER"),
		DBPassword: getenv("POSTGRES_PASSWORD"),
		DBName:     getenv("POSTGRES_DB"),
		Listen:     get("EXPORTER_LISTEN", ":9190"),
		LogLevel:   get("EXPORTER_LOG_LEVEL", "info"),
		LogFile:    getenv("EXPORTER_LOG_FILE"),
	}

	var errs []error
	for key, v := range map[string]string{"POSTGRES_USER": c.DBUser, "POSTGRES_PASSWORD": c.DBPassword, "POSTGRES_DB": c.DBName} {
		if v == "" {
			errs = append(errs, fmt.Errorf("%s is required", key))
		}
	}

	port, err := strconv.ParseUint(get("EXPORTER_DB_PORT", "5432"), 10, 16)
	if err != nil {
		errs = append(errs, fmt.Errorf("EXPORTER_DB_PORT: %w", err))
	}
	c.DBPort = uint16(port)

	if c.SourcesWindow, err = positiveDuration(get("EXPORTER_SOURCES_WINDOW", "168h")); err != nil {
		errs = append(errs, fmt.Errorf("EXPORTER_SOURCES_WINDOW: %w", err))
	}
	if c.QueryTimeout, err = positiveDuration(get("EXPORTER_QUERY_TIMEOUT", "10s")); err != nil {
		errs = append(errs, fmt.Errorf("EXPORTER_QUERY_TIMEOUT: %w", err))
	}
	if c.TopN, err = strconv.Atoi(get("EXPORTER_TOP_N", "20")); err != nil || c.TopN < 1 {
		errs = append(errs, errors.New("EXPORTER_TOP_N must be a positive integer"))
	}
	return c, errors.Join(errs...)
}

func positiveDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err == nil && d <= 0 {
		err = errors.New("must be positive")
	}
	return d, err
}

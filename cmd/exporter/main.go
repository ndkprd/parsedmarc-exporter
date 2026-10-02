// Command exporter serves Prometheus metrics from the parsedmarc database.
package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"

	"gitlab.com/endekasoft/parsedmarc-exporter/internal/collector"
	"gitlab.com/endekasoft/parsedmarc-exporter/internal/config"
	"gitlab.com/endekasoft/parsedmarc-exporter/internal/pgstore"
)

// Set at build time with -ldflags "-X main.version=... -X main.commit=...".
var (
	version = "dev"
	commit  = "unknown"
)

//go:embed index.html
var indexHTML string

var indexTemplate = template.Must(template.New("index").Parse(indexHTML))

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe /healthz on the local listener and exit")
	flag.Parse()

	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "invalid configuration:", err)
		os.Exit(2)
	}
	if *healthcheck {
		os.Exit(probe(cfg.Listen))
	}

	log, closeLog, err := newLogger(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "logging:", err)
		os.Exit(2)
	}
	defer closeLog()

	if err := run(cfg, log); err != nil {
		log.Error().Err(err).Msg("exporter stopped")
		closeLog()
		os.Exit(1)
	}
}

func run(cfg config.Config, log zerolog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := pgstore.New(ctx, cfg)
	if err != nil {
		return err
	}
	defer store.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(collector.New(store, collector.Options{
		Timeout:       cfg.QueryTimeout,
		SourcesWindow: cfg.SourcesWindow,
		TopN:          cfg.TopN,
		Version:       version,
		Commit:        commit,
	}, log))

	srv := &http.Server{
		Addr:              cfg.Listen,
		Handler:           newMux(reg),
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Info().Str("listen", cfg.Listen).Str("version", version).Msg("exporter started")

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info().Msg("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func newMux(gatherer prometheus.Gatherer) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{}))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok\n")
	})
	// "GET /{$}" matches the root path only; any other path is a 404.
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_ = indexTemplate.Execute(w, struct{ Version string }{version})
	})
	return mux
}

func newLogger(cfg config.Config) (zerolog.Logger, func(), error) {
	level, err := zerolog.ParseLevel(cfg.LogLevel)
	if err != nil {
		return zerolog.Logger{}, nil, fmt.Errorf("EXPORTER_LOG_LEVEL: %w", err)
	}
	var out io.Writer = os.Stderr
	closeLog := func() {}
	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
		if err != nil {
			return zerolog.Logger{}, nil, fmt.Errorf("open log file: %w", err)
		}
		out = io.MultiWriter(os.Stderr, f)
		closeLog = func() { _ = f.Close() }
	}
	return zerolog.New(out).Level(level).With().Timestamp().Logger(), closeLog, nil
}

// probe implements -healthcheck for the container healthcheck.
func probe(listen string) int {
	addr := listen
	if len(addr) > 0 && addr[0] == ':' {
		addr = "127.0.0.1" + addr
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

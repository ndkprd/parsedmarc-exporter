package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestRoutes(t *testing.T) {
	mux := newMux(prometheus.NewRegistry())
	tests := []struct {
		method, path string
		status       int
		contentType  string
		body         string
	}{
		{http.MethodGet, "/", http.StatusOK, "text/html", `href="/metrics"`},
		{http.MethodGet, "/healthz", http.StatusOK, "", "ok"},
		{http.MethodGet, "/metrics", http.StatusOK, "", ""},
		{http.MethodGet, "/nope", http.StatusNotFound, "", ""},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(tc.method, tc.path, nil))
		if rec.Code != tc.status {
			t.Errorf("%s %s status = %d, want %d", tc.method, tc.path, rec.Code, tc.status)
		}
		if !strings.HasPrefix(rec.Header().Get("Content-Type"), tc.contentType) {
			t.Errorf("%s content type = %q, want prefix %q", tc.path, rec.Header().Get("Content-Type"), tc.contentType)
		}
		if !strings.Contains(rec.Body.String(), tc.body) {
			t.Errorf("%s body missing %q", tc.path, tc.body)
		}
	}
}

package httpserver

import (
	"net/http"
	"time"
)

const (
	// PublicMaxRequestBodyBytes bounds JSON request bodies at the HTTP boundary.
	// Chapter edit/review payloads are the largest expected client-controlled
	// inputs; generation and audio work remain job-based rather than unbounded
	// HTTP uploads.
	PublicMaxRequestBodyBytes = 2 << 20 // 2 MiB

	publicMaxHeaderBytes  = 64 << 10 // 64 KiB
	metricsMaxHeaderBytes = 32 << 10 // 32 KiB
)

// ApplyPublicAPIPolicy configures explicit connection and request bounds for the
// public product API listener. WriteTimeout must remain at or above the
// longest synchronous TextAI handler budget (~10 minutes per provider contract)
// so legitimate admin generation/review calls are not truncated mid-response.
func ApplyPublicAPIPolicy(srv *http.Server) {
	srv.ReadHeaderTimeout = 5 * time.Second
	srv.ReadTimeout = 60 * time.Second
	srv.WriteTimeout = 11 * time.Minute
	srv.IdleTimeout = 120 * time.Second
	srv.MaxHeaderBytes = publicMaxHeaderBytes
}

// ApplyMetricsPolicy configures bounded scrape-oriented semantics for the
// private metrics listener. Values are intentionally tighter than the public API
// because scrapes are small GET requests on loopback/private interfaces.
func ApplyMetricsPolicy(srv *http.Server) {
	srv.ReadHeaderTimeout = 5 * time.Second
	srv.ReadTimeout = 15 * time.Second
	srv.WriteTimeout = 30 * time.Second
	srv.IdleTimeout = 60 * time.Second
	srv.MaxHeaderBytes = metricsMaxHeaderBytes
}

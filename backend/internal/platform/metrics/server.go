package metrics

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// NewPrivateServer builds a metrics-only HTTP server and rejects wildcard or
// public bind addresses. Production scrape exposure therefore requires an
// explicit loopback/private address rather than relying on ingress filtering.
func NewPrivateServer(addr string, handler http.Handler) (*http.Server, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, nil
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid metrics address: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || (!ip.IsLoopback() && !ip.IsPrivate()) {
		return nil, fmt.Errorf("metrics address must use an explicit loopback/private IP")
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}

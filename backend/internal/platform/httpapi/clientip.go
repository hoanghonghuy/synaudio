package httpapi

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type clientIPContextKey struct{}

// ClientIP returns the request client IP chosen by trusted-proxy middleware.
// When no trusted client IP was stored, RemoteAddr is used and forwarded
// headers are ignored.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(clientIPContextKey{}).(string); ok && ip != "" {
		return ip
	}
	return directRemoteIP(r)
}

// TrustedProxyConfig describes which immediate peers may supply forwarded headers.
type TrustedProxyConfig struct {
	TrustedNetworks []*net.IPNet
}

// ParseTrustedProxyConfig parses a comma-separated list of CIDR blocks.
func ParseTrustedProxyConfig(raw string) (TrustedProxyConfig, error) {
	cfg := TrustedProxyConfig{}
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.Contains(part, "/") {
			part += "/32"
		}
		_, network, err := net.ParseCIDR(part)
		if err != nil {
			return TrustedProxyConfig{}, err
		}
		cfg.TrustedNetworks = append(cfg.TrustedNetworks, network)
	}
	return cfg, nil
}

func (c TrustedProxyConfig) Trusted(peer string) bool {
	if len(c.TrustedNetworks) == 0 {
		return false
	}
	ip := net.ParseIP(strings.TrimSpace(peer))
	if ip == nil {
		return false
	}
	for _, network := range c.TrustedNetworks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// WithTrustedClientIP stores a trustworthy client IP on the request context.
// Forwarded headers are honored only when RemoteAddr is a configured trusted peer.
func WithTrustedClientIP(cfg TrustedProxyConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := directRemoteIP(r)
			peerIP := directRemoteIP(r)
			if cfg.Trusted(peerIP) {
				if forwarded := forwardedClientIP(r); forwarded != "" {
					clientIP = forwarded
					r.RemoteAddr = net.JoinHostPort(forwarded, "0")
				}
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPContextKey{}, clientIP)))
		})
	}
}

func directRemoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func forwardedClientIP(r *http.Request) string {
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		if ip := net.ParseIP(realIP); ip != nil {
			return ip.String()
		}
	}
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if ip := net.ParseIP(first); ip != nil {
			return ip.String()
		}
	}
	return ""
}

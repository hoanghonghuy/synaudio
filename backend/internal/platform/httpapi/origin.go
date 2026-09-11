package httpapi

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/synaudio/synaudio/backend/internal/identity"
)

func cookieMutationOriginError(r *http.Request) string {
	if !cookieProtectedAuthMutation(r) {
		return ""
	}
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return "CSRF_ORIGIN_REQUIRED"
	}
	if !sameOriginRequest(r, origin) {
		return "CSRF_ORIGIN_MISMATCH"
	}
	return ""
}

func cookieProtectedAuthMutation(r *http.Request) bool {
	if r.Method != http.MethodPost || !hasRefreshCookie(r) {
		return false
	}
	path := strings.TrimSuffix(r.URL.Path, "/")
	switch path {
	case "/api/v1/auth/refresh":
		return true
	case "/api/v1/auth/logout":
		return !hasBearerAuthorization(r)
	default:
		return false
	}
}

func hasRefreshCookie(r *http.Request) bool {
	for _, name := range []string{identity.RefreshCookieName, identity.DevelopmentRefreshCookieName} {
		if cookie, err := r.Cookie(name); err == nil && cookie.Value != "" {
			return true
		}
	}
	return false
}

func hasBearerAuthorization(r *http.Request) bool {
	value := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(value) < len("Bearer ") {
		return false
	}
	return strings.EqualFold(value[:len("Bearer ")], "Bearer ") && strings.TrimSpace(value[len("Bearer "):]) != ""
}

func sameOriginRequest(r *http.Request, origin string) bool {
	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" || originURL.User != nil {
		return false
	}
	if originURL.Path != "" || originURL.RawQuery != "" || originURL.Fragment != "" {
		return false
	}

	scheme := "http"
	if r.TLS != nil || strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")), "https") {
		scheme = "https"
	}
	targetURL, err := url.Parse(scheme + "://" + r.Host)
	if err != nil || targetURL.Host == "" {
		return false
	}

	return strings.EqualFold(originURL.Scheme, targetURL.Scheme) &&
		strings.EqualFold(originURL.Hostname(), targetURL.Hostname()) &&
		effectivePort(originURL) == effectivePort(targetURL)
}

func effectivePort(u *url.URL) string {
	if port := u.Port(); port != "" {
		return port
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

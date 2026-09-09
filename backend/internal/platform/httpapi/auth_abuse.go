package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/synaudio/synaudio/backend/internal/platform/metrics"
)

const (
	AbuseDimensionClient  = "client"
	AbuseDimensionAccount = "account"
	AbuseDimensionSession = "session"
)

// AbuseLimit defines one bounded counter dimension for an auth route.
type AbuseLimit struct {
	Dimension AbuseDimension
	Limit     int
	Window    time.Duration
}

type AbuseDimension string

// AuthAbusePolicy configures abuse controls for one auth route.
type AuthAbusePolicy struct {
	Limits            []AbuseLimit
	EnumerationSafe   bool
	EnumerationStatus int
}

// AuthAbuse wires production auth abuse controls into the router.
type AuthAbuse struct {
	Policies AuthAbusePolicies
	Limiter  AbuseLimiter
	Metrics  *metrics.Registry
	Logger   *slog.Logger
}

type AuthAbusePolicies map[string]AuthAbusePolicy

// DefaultAuthAbusePolicies returns the canonical sensitive-auth route policies.
// Issue #14 can mount additional MFA challenge routes under the same paths to
// inherit these limits without changing the middleware contract.
func DefaultAuthAbusePolicies() AuthAbusePolicies {
	return AuthAbusePolicies{
		"POST /login": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 30, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /register": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /refresh": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 60, Window: 15 * time.Minute},
			},
		},
		"POST /email/verify": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 20, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /email/resend": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 5, Window: time.Hour},
			},
			EnumerationSafe:   true,
			EnumerationStatus: http.StatusAccepted,
		},
		"POST /password/forgot": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 5, Window: time.Hour},
			},
			EnumerationSafe:   true,
			EnumerationStatus: http.StatusAccepted,
		},
		"POST /password/reset": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 5, Window: 15 * time.Minute},
			},
		},
		"POST /mfa/totp/setup": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionSession, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /mfa/totp/confirm": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /mfa/totp/disable": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 5, Window: 15 * time.Minute},
			},
		},
		"POST /mfa/challenge": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /mfa/verify": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionAccount, Limit: 10, Window: 15 * time.Minute},
			},
		},
		"POST /re-auth": {
			Limits: []AbuseLimit{
				{Dimension: AbuseDimensionClient, Limit: 10, Window: 15 * time.Minute},
				{Dimension: AbuseDimensionSession, Limit: 10, Window: 15 * time.Minute},
			},
		},
	}
}

func (a *AuthAbuse) Middleware() func(http.Handler) http.Handler {
	if a == nil || a.Limiter == nil || len(a.Policies) == 0 {
		return func(next http.Handler) http.Handler { return next }
	}
	return NewAuthAbuseMiddleware(a.Policies, a.Limiter, a.Metrics, a.Logger)
}

func NewAuthAbuseMiddleware(
	policies AuthAbusePolicies,
	limiter AbuseLimiter,
	registry *metrics.Registry,
	logger *slog.Logger,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			routeKey := r.Method + " " + AuthRoutePath(r.URL.Path)
			policy, ok := policies[routeKey]
			if !ok || len(policy.Limits) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			body, err := readAndRestoreBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
				return
			}

			email := normalizedEmailFromBody(body)
			clientKey := abuseKey("client", ClientIP(r))
			accountKey := ""
			if email != "" {
				accountKey = abuseKey("account", email)
			}
			sessionKey := abuseSessionKeyFromBearer(r)

			for _, limit := range policy.Limits {
				key := clientKey
				switch limit.Dimension {
				case AbuseDimensionClient:
					key = clientKey
				case AbuseDimensionAccount:
					if accountKey == "" {
						continue
					}
					key = accountKey
				case AbuseDimensionSession:
					if sessionKey == "" {
						continue
					}
					key = sessionKey
				default:
					continue
				}

				scope := routeKey + ":" + string(limit.Dimension)
				allowed, retryAfter, allowErr := limiter.Allow(r.Context(), scope, key, limit.Limit, limit.Window)
				if allowErr != nil {
					writeError(w, http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "service temporarily unavailable")
					return
				}
				if !allowed {
					observeThrottle(registry, routeKey, string(limit.Dimension))
					logThrottle(logger, r, routeKey, string(limit.Dimension))
					if policy.EnumerationSafe {
						writeJSON(w, policy.EnumerationStatus, map[string]string{"status": "accepted"})
						return
					}
					writeRateLimited(w, retryAfter)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func readAndRestoreBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

type bearerAccessClaims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
}

func abuseSessionKeyFromBearer(r *http.Request) string {
	token, err := bearerTokenFromRequest(r)
	if err != nil {
		return ""
	}
	claims, err := decodeUnverifiedBearerClaims(token)
	if err != nil || claims.SessionID == "" {
		return ""
	}
	return abuseKey("session", claims.SessionID)
}

func bearerTokenFromRequest(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", errors.New("missing authorization")
	}
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("invalid authorization")
	}
	return parts[1], nil
}

func decodeUnverifiedBearerClaims(token string) (bearerAccessClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[1] == "" {
		return bearerAccessClaims{}, errors.New("invalid token")
	}
	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return bearerAccessClaims{}, err
	}
	var claims bearerAccessClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return bearerAccessClaims{}, err
	}
	return claims, nil
}

func normalizedEmailFromBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Email))
}

func abuseKey(prefix, value string) string {
	sum := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(sum[:])
}

func writeRateLimited(w http.ResponseWriter, retryAfter time.Duration) {
	if retryAfter > 0 {
		seconds := int(retryAfter.Round(time.Second) / time.Second)
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
	}
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error": map[string]string{
			"code":    "RATE_LIMITED",
			"message": "too many requests",
		},
	})
}

func observeThrottle(registry *metrics.Registry, route, dimension string) {
	if registry == nil {
		return
	}
	registry.ObserveAuthThrottled(route, dimension)
}

func logThrottle(logger *slog.Logger, r *http.Request, route, dimension string) {
	if logger == nil {
		return
	}
	logger.Warn("auth request throttled",
		"route", route,
		"dimension", dimension,
		"client_ip_prefix", maskedIPPrefix(ClientIP(r)),
		"request_id", middleware.GetReqID(r.Context()),
	)
}

// AuthRoutePath normalizes auth route paths for abuse policy lookup.
func AuthRoutePath(path string) string {
	path = strings.TrimSuffix(strings.TrimSpace(path), "/")
	const prefix = "/api/v1/auth"
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimSuffix(path[len(prefix):], "/")
	}
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func maskedIPPrefix(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ""
	}
	if v4 := parsed.To4(); v4 != nil {
		return net.IPv4(v4[0], v4[1], v4[2], 0).String() + "/24"
	}
	masked := parsed.Mask(net.CIDRMask(48, 128))
	if masked == nil {
		return ""
	}
	return masked.String() + "/48"
}

package httpapi_test

import (
	"testing"

	"github.com/synaudio/synaudio/backend/internal/platform/httpapi"
)

func TestAuthRoutePathNormalizesMountedPaths(t *testing.T) {
	cases := map[string]string{
		"/login":                "/login",
		"/api/v1/auth/login":    "/login",
		"/api/v1/auth/login/":   "/login",
		"/email/resend":         "/email/resend",
		"/api/v1/auth/email/resend": "/email/resend",
	}
	for input, want := range cases {
		if got := httpapi.AuthRoutePath(input); got != want {
			t.Fatalf("%q: expected %q, got %q", input, want, got)
		}
	}
}

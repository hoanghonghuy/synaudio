package httpapi

import (
	"net/http"
	"testing"
)

func TestAdminDirectoryRoutesUseExplicitPermissionWithoutRecentAuth(t *testing.T) {
	for _, route := range []string{"/admin/users", "/admin/users/{userID}"} {
		policy := adminPolicyFor(http.MethodGet, route)
		if policy.Permission != "USER_STATUS_MANAGE" {
			t.Fatalf("route %s permission = %q", route, policy.Permission)
		}
		if policy.RecentAuth {
			t.Fatalf("route %s should not require recent-auth for read-only directory access", route)
		}
	}
}

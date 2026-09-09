package httpapi

import "net/http"

// authSecurityPolicyFor classifies mutating /auth routes that require the frozen
// recent-auth window before executing consequential security lifecycle changes.
func authSecurityPolicyFor(method, route string) adminRoutePolicy {
	switch method + " " + route {
	case http.MethodPost + " /mfa/totp/disable":
		return adminRoutePolicy{RecentAuth: true}
	case http.MethodPost + " /account/deletion/request":
		return adminRoutePolicy{RecentAuth: true}
	default:
		return adminRoutePolicy{}
	}
}

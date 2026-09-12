package httpapi

import (
	"net/http"
	"testing"
)

func TestCreativeDecisionMutationPoliciesRequireOperationPermissionAndRecentAuth(t *testing.T) {
	tests := []struct {
		name       string
		route      string
		permission string
	}{
		{name: "select", route: "/admin/creative-decisions/{decisionID}/select", permission: "CREATIVE_DECISION_RESOLVE"},
		{name: "reject", route: "/admin/creative-decisions/{decisionID}/reject", permission: "CREATIVE_DECISION_REJECT"},
		{name: "postpone", route: "/admin/creative-decisions/{decisionID}/postpone", permission: "CREATIVE_DECISION_POSTPONE"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := adminPolicyFor(http.MethodPost, tt.route)
			if policy.Permission != tt.permission {
				t.Fatalf("expected permission %q, got %q", tt.permission, policy.Permission)
			}
			if !policy.RecentAuth {
				t.Fatal("expected creative decision mutation to require recent auth")
			}
		})
	}
}

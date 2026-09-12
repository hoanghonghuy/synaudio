package audit

import (
	"net/http"
	"testing"
)

func TestCreativeDecisionPostponeUsesSemanticAuditDescriptor(t *testing.T) {
	desc, ok := auditDescriptor(http.MethodPost, "/admin/creative-decisions/{decisionID}/postpone")
	if !ok {
		t.Fatal("expected postpone route to be audited")
	}
	if desc.Action != "CREATIVE_DECISION_POSTPONED" {
		t.Fatalf("expected CREATIVE_DECISION_POSTPONED, got %q", desc.Action)
	}
	if desc.ResourceType != "CREATIVE_DECISION" || desc.ResourceParam != "decisionID" {
		t.Fatalf("unexpected postpone audit descriptor: %#v", desc)
	}
}

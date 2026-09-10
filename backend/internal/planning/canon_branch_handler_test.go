package planning

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWorkspaceHandlerGetsActiveOfficialCanonBranch(t *testing.T) {
	store := newFakeStore()
	store.branches["s1"] = []CanonBranch{
		{ID: "retcon-active", StoryID: "s1", Type: "RETCON", Status: "ACTIVE"},
		{ID: "official-active", StoryID: "s1", Type: "OFFICIAL", Status: "ACTIVE"},
	}
	h := NewWorkspaceHandler(NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/stories/s1/canon-branches/active-official", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var got CanonBranch
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "official-active" || got.Type != "OFFICIAL" || got.Status != "ACTIVE" {
		t.Fatalf("unexpected branch response: %+v", got)
	}
}

func TestWorkspaceHandlerActiveOfficialCanonBranchFailsClosedWhenMissing(t *testing.T) {
	store := newFakeStore()
	store.branches["s1"] = []CanonBranch{{ID: "retcon-active", StoryID: "s1", Type: "RETCON", Status: "ACTIVE"}}
	h := NewWorkspaceHandler(NewService(store))

	req := httptest.NewRequest(http.MethodGet, "/admin/stories/s1/canon-branches/active-official", nil)
	res := httptest.NewRecorder()
	h.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", res.Code, res.Body.String())
	}
}

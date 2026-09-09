package planning

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetPublishReadinessEndpointReturnsMissingDependencies(t *testing.T) {
	store := newPublishFakeStore()
	ch, _ := store.CreateChapter(t.Context(), Chapter{ID: "ch-1", StoryID: "s1", Status: "READY"})
	svc := NewService(store, WithPublishChecker(NewCompositePublishChecker(
		store,
		&fakePublishContentAuthority{hasApproved: false},
		&fakePublishAudioAuthority{missing: []string{"active_audio"}},
		&fakePublishStoryAuthority{},
	)))
	handler := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/admin/chapters/"+ch.ID+"/publish-readiness", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["ready"] != false {
		t.Fatalf("expected ready=false, got %#v", body["ready"])
	}
}

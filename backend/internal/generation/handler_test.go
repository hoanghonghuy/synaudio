package generation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEditContentHandlerUsesResolvedActor(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)
	original, err := svc.CreateContentRevision(context.Background(), "chapter-1", "original text", "AI_GENERATED", "writer")
	if err != nil {
		t.Fatalf("create original revision: %v", err)
	}

	handler := NewHandler(svc, func(context.Context, *http.Request) (string, error) {
		return "session-admin", nil
	})
	body := `{"based_on_revision_id":"` + original.ID + `","text":"edited text","edited_by":"attacker"}`
	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/edit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var revision ContentRevision
	if err := json.Unmarshal(rec.Body.Bytes(), &revision); err != nil {
		t.Fatalf("decode revision: %v", err)
	}
	if revision.CreatedBy != "session-admin" {
		t.Fatalf("expected session actor, got %q", revision.CreatedBy)
	}
}

func TestGenerationAdminHandlerRequiresActorResolver(t *testing.T) {
	handler := NewHandler(NewService(newFakeStore()), nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/chapters/chapter-1/edit", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

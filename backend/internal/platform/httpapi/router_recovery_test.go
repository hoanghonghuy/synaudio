package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestNewRouterRecoversPanicWithoutLogger(t *testing.T) {
	storyRoutes := chi.NewRouter()
	storyRoutes.Get("/stories/panic", func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})

	router := NewRouter(Dependencies{StoryHandler: storyRoutes})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/stories/panic", nil)

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("panic escaped router without logger: %v", recovered)
		}
	}()

	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 from recovered panic, got %d", recorder.Code)
	}
}

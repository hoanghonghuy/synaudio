package audit

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBufferedResponseFlushDefaultsStatusToOK(t *testing.T) {
	buffer := newBufferedResponse()
	recorder := httptest.NewRecorder()

	buffer.flushTo(recorder)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected default status %d, got %d", http.StatusOK, recorder.Code)
	}
}

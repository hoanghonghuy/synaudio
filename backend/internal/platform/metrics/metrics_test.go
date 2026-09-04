package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRegistryPrometheusOutputUsesBoundedLabels(t *testing.T) {
	r := NewRegistry()
	r.ObserveAPI(http.MethodGet, "/api/v1/stories/{storyID}", http.StatusOK, 250*time.Millisecond)
	r.ObserveAPI("TRACE", "/api/v1/stories/secret-user-id?token=secret", http.StatusInternalServerError, time.Second)
	r.ObserveWorkerLoop("audit_delivery", nil)
	r.AddWorkerItems("audit_delivery", "dead_letter", 2)
	r.ObserveGenerationJob("WRITER", "failure", "TRANSIENT")
	r.ObserveGenerationJob("user-controlled-job-type", "failure", "raw provider error with content")
	r.WorkerHeartbeat(time.Unix(123, 0))

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	res := httptest.NewRecorder()
	r.Handler().ServeHTTP(res, req)
	body := res.Body.String()

	checks := []string{
		`synaudio_api_requests_total{method="GET",route="/api/v1/stories/{storyID}",status_class="2xx"} 1`,
		`synaudio_api_requests_total{method="OTHER",route="unmatched",status_class="5xx"} 1`,
		`synaudio_worker_heartbeat_unixtime 123`,
		`synaudio_worker_loop_items_total{loop="audit_delivery",result="dead_letter"} 2`,
		`synaudio_generation_jobs_total{job_type="WRITER",outcome="failure",error_class="TRANSIENT"} 1`,
		`synaudio_generation_jobs_total{job_type="UNKNOWN",outcome="failure",error_class="other"} 1`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "secret-user-id") || strings.Contains(body, "provider error with content") {
		t.Fatalf("metrics output leaked unbounded label content: %s", body)
	}
}

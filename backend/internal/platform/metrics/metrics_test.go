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
	r.SetBacklog("generation", 3, 42*time.Second, 0)
	r.SetBacklog("user-controlled-queue", 4, time.Minute, 2)
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
		`synaudio_backlog_depth{queue="generation"} 3`,
		`synaudio_backlog_oldest_age_seconds{queue="generation"} 42`,
		`synaudio_backlog_dead_letter{queue="generation"} 0`,
		`synaudio_backlog_depth{queue="other"} 4`,
	}
	for _, want := range checks {
		if !strings.Contains(body, want) {
			t.Fatalf("metrics output missing %q\n%s", want, body)
		}
	}
	if strings.Contains(body, "secret-user-id") || strings.Contains(body, "provider error with content") || strings.Contains(body, "user-controlled-queue") {
		t.Fatalf("metrics output leaked unbounded label content: %s", body)
	}
}

func TestRegistryBacklogGaugeReplacesPreviousSnapshot(t *testing.T) {
	r := NewRegistry()
	r.SetBacklog("audit_outbox", 5, 90*time.Second, 2)
	r.SetBacklog("audit_outbox", 0, 0, 0)

	res := httptest.NewRecorder()
	r.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := res.Body.String()

	for _, want := range []string{
		`synaudio_backlog_depth{queue="audit_outbox"} 0`,
		`synaudio_backlog_oldest_age_seconds{queue="audit_outbox"} 0`,
		`synaudio_backlog_dead_letter{queue="audit_outbox"} 0`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("backlog zero transition missing %q\n%s", want, body)
		}
	}
}

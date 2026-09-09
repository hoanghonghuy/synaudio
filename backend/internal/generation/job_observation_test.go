package generation

import "testing"

func TestObserveJobQueuedPending(t *testing.T) {
	view := ObserveJob(GenerationJob{Status: "PENDING", AttemptCount: 0, MaxAttempts: 3})
	if view.Observation != "queued" || view.Retryable {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestObserveJobRunning(t *testing.T) {
	view := ObserveJob(GenerationJob{Status: "RUNNING", AttemptCount: 1, MaxAttempts: 3})
	if view.Observation != "running" || view.Retryable {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestObserveJobSucceeded(t *testing.T) {
	view := ObserveJob(GenerationJob{Status: "SUCCEEDED", AttemptCount: 1, MaxAttempts: 3})
	if view.Observation != "succeeded" || view.Retryable {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestObserveJobRetryableFailedTransient(t *testing.T) {
	view := ObserveJob(GenerationJob{
		Status:         "FAILED",
		AttemptCount:   1,
		MaxAttempts:    3,
		LastErrorClass: "TRANSIENT",
		LastErrorCode:  "PROVIDER_TIMEOUT",
	})
	if view.Observation != "retryable" || !view.Retryable || view.AttemptsExhausted {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestObserveJobExhaustedFailed(t *testing.T) {
	view := ObserveJob(GenerationJob{
		Status:         "FAILED",
		AttemptCount:   3,
		MaxAttempts:    3,
		LastErrorClass: "RETRY_EXHAUSTED",
		LastErrorCode:  "MAX_ATTEMPTS_EXHAUSTED",
	})
	if view.Observation != "exhausted" || view.Retryable || !view.AttemptsExhausted {
		t.Fatalf("unexpected view: %#v", view)
	}
}

func TestObserveJobPermanentFailedIsNotRetryable(t *testing.T) {
	view := ObserveJob(GenerationJob{
		Status:         "FAILED",
		AttemptCount:   1,
		MaxAttempts:    3,
		LastErrorClass: "PERMANENT",
		LastErrorCode:  "POLICY_BLOCK",
	})
	if view.Observation != "failed" || view.Retryable {
		t.Fatalf("unexpected view: %#v", view)
	}
}

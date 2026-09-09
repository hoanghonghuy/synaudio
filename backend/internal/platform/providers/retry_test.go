package providers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

func TestClassifyHTTPStatus(t *testing.T) {
	tests := []struct {
		status int
		class  string
		code   string
	}{
		{http.StatusTooManyRequests, "TRANSIENT", "PROVIDER_RATE_LIMITED"},
		{http.StatusServiceUnavailable, "TRANSIENT", "PROVIDER_UNAVAILABLE"},
		{http.StatusInternalServerError, "TRANSIENT", "PROVIDER_UNAVAILABLE"},
		{http.StatusUnauthorized, "PERMANENT", "PROVIDER_AUTH"},
		{http.StatusForbidden, "PERMANENT", "PROVIDER_AUTH"},
		{http.StatusBadRequest, "PERMANENT", "PROVIDER_CONFIG"},
		{http.StatusNotFound, "PERMANENT", "PROVIDER_CONFIG"},
		{http.StatusTeapot, "PERMANENT", "PROVIDER_HTTP_ERROR"},
	}
	for _, tc := range tests {
		class, code := classifyHTTPStatus(tc.status)
		if class != tc.class || code != tc.code {
			t.Fatalf("status %d: expected %s/%s, got %s/%s", tc.status, tc.class, tc.code, class, code)
		}
	}
}

func TestParseRetryAfter(t *testing.T) {
	header := make(http.Header)
	header.Set("Retry-After", "2")
	if got := parseRetryAfter(header); got != 2*time.Second {
		t.Fatalf("expected 2s, got %v", got)
	}
}

func TestGenerateRetries429WithRetryAfterThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	var slept time.Duration
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
		ResetCallObserver()
	})
	SetProviderSleepForTests(func(_ context.Context, delay time.Duration) error {
		slept += delay
		return nil
	})

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			n := calls.Add(1)
			if n == 1 {
				header := make(http.Header)
				header.Set("Retry-After", "1")
				return &http.Response{
					StatusCode: http.StatusTooManyRequests,
					Header:     header,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"rate limited"}}`)),
					Request:    req,
				}, nil
			}
			return okTextResponse(req, "hello")
		})},
	}

	resp, err := client.generate(context.Background(), "prompt", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	text, err := responseText(resp)
	if err != nil || text != "hello" {
		t.Fatalf("expected hello, got %q (%v)", text, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 provider calls, got %d", calls.Load())
	}
	if slept < time.Second {
		t.Fatalf("expected Retry-After delay >= 1s, got %v", slept)
	}
}

func TestGenerateRetries503ThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})
	SetProviderSleepForTests(func(context.Context, time.Duration) error { return nil })

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unavailable"}}`)),
					Request:    req,
				}, nil
			}
			return okTextResponse(req, "done")
		})},
	}

	if _, err := client.generate(context.Background(), "prompt", nil); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

func TestGenerateRetriesNonJSON503ThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})
	SetProviderSleepForTests(func(context.Context, time.Duration) error { return nil })

	const upstreamBody = "<html><body>503 Service Unavailable: upstream unavailable secret-detail</body></html>"
	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader(upstreamBody)),
					Request:    req,
				}, nil
			}
			return okTextResponse(req, "recovered")
		})},
	}

	resp, err := client.generate(context.Background(), "prompt", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	text, err := responseText(resp)
	if err != nil || text != "recovered" {
		t.Fatalf("expected recovered, got %q (%v)", text, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls after non-JSON 503 retry, got %d", calls.Load())
	}
}

func TestGenerateDoesNotRetryPermanentAuthFailure(t *testing.T) {
	var calls atomic.Int32
	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusUnauthorized,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"invalid api key secret-token"}}`)),
				Request:    req,
			}, nil
		})},
	}

	_, err := client.generate(context.Background(), "prompt", nil)
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if calls.Load() != 1 {
		t.Fatalf("expected single provider call, got %d", calls.Load())
	}
	class, code := generation.ClassifyError(err)
	if class != "PERMANENT" || code != "PROVIDER_AUTH" {
		t.Fatalf("expected PERMANENT/PROVIDER_AUTH, got %s/%s", class, code)
	}
	if strings.Contains(err.Error(), "secret-token") {
		t.Fatalf("raw provider body leaked into error: %v", err)
	}
}

func TestGenerateExhaustsInCallRetriesForTransientFailure(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})
	SetProviderSleepForTests(func(context.Context, time.Duration) error { return nil })

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad gateway"}}`)),
				Request:    req,
			}, nil
		})},
	}

	_, err := client.generate(context.Background(), "prompt", nil)
	if err == nil {
		t.Fatal("expected retry exhaustion")
	}
	wantCalls := int32(1 + MaxInCallRetries)
	if calls.Load() != wantCalls {
		t.Fatalf("expected %d calls, got %d", wantCalls, calls.Load())
	}
	class, code := generation.ClassifyError(err)
	if class != "TRANSIENT" || code != "PROVIDER_RETRY_EXHAUSTED" {
		t.Fatalf("expected TRANSIENT/PROVIDER_RETRY_EXHAUSTED, got %s/%s", class, code)
	}
}

func TestGenerateStopsOnContextCancelDuringBackoff(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})

	block := make(chan struct{})
	SetProviderSleepForTests(func(ctx context.Context, delay time.Duration) error {
		close(block)
		return sleepWithContext(ctx, delay)
	})

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"slow down"}}`)),
				Request:    req,
			}, nil
		})},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := client.generate(ctx, "prompt", nil)
		done <- err
	}()

	<-block
	cancel()

	err := <-done
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one call before cancel, got %d", calls.Load())
	}
}

func TestClassifyAttemptFailure2xxBodyReadResetIsTransient(t *testing.T) {
	result := attemptResult{
		statusCode: http.StatusOK,
		err:        &responseReadError{cause: errors.New("connection reset by peer")},
	}
	class, code, retryable, retryAfter := classifyAttemptFailure(result)
	if class != "TRANSIENT" || code != "PROVIDER_NETWORK" || !retryable || retryAfter != 0 {
		t.Fatalf("expected TRANSIENT/PROVIDER_NETWORK retryable, got %s/%s retryable=%v retryAfter=%v", class, code, retryable, retryAfter)
	}
}

func TestClassifyAttemptFailure2xxMalformedJSONStaysPermanent(t *testing.T) {
	result := attemptResult{
		statusCode: http.StatusOK,
		err:        classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("invalid character")),
	}
	class, code, retryable, _ := classifyAttemptFailure(result)
	if class != "PERMANENT" || code != "PROVIDER_MALFORMED" || retryable {
		t.Fatalf("expected PERMANENT/PROVIDER_MALFORMED non-retryable, got %s/%s retryable=%v", class, code, retryable)
	}
}

func TestGenerateRetries2xxBodyReadResetThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})
	SetProviderSleepForTests(func(context.Context, time.Duration) error { return nil })

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     make(http.Header),
					Body:       io.NopCloser(errReader{err: errors.New("connection reset by peer")}),
					Request:    req,
				}, nil
			}
			return okTextResponse(req, "recovered")
		})},
	}

	resp, err := client.generate(context.Background(), "prompt", nil)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	text, err := responseText(resp)
	if err != nil || text != "recovered" {
		t.Fatalf("expected recovered, got %q (%v)", text, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls after 2xx body-read reset retry, got %d", calls.Load())
	}
}

func TestGenerateStopsOnContextCancelDuring2xxBodyRead(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(cancelOnReadReader{cancel: cancel}),
				Request:    req,
			}, nil
		})},
	}

	_, err := client.generate(ctx, "prompt", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("expected one call before cancel, got %d", calls.Load())
	}
}

func TestGenerateNetworkFailureThenSucceeds(t *testing.T) {
	var calls atomic.Int32
	t.Cleanup(func() {
		SetProviderSleepForTests(nil)
	})
	SetProviderSleepForTests(func(context.Context, time.Duration) error { return nil })

	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				return nil, errors.New("connection reset")
			}
			return okTextResponse(req, "recovered")
		})},
	}

	if _, err := client.generate(context.Background(), "prompt", nil); err != nil {
		t.Fatalf("generate: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("expected 2 calls, got %d", calls.Load())
	}
}

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

type cancelOnReadReader struct {
	cancel func()
}

func (r cancelOnReadReader) Read([]byte) (int, error) {
	r.cancel()
	return 0, context.Canceled
}

func okTextResponse(req *http.Request, text string) (*http.Response, error) {
	body := `{"candidates":[{"content":{"parts":[{"text":"` + text + `"}]}}]}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

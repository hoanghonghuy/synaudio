// Package providers implements production AI/TTS adapters and their outbound
// retry contract.
//
// Retry authority is split across two bounded layers:
//  1. In-call provider retries (this package): up to 1+MaxInCallRetries HTTP
//     attempts for a single logical provider invocation, with exponential backoff,
//     jitter, and Retry-After honoring on throttling responses.
//  2. Durable GenerationJob retries (#44): the worker requeues TRANSIENT job
//     failures until attempt_count reaches max_attempts. This layer owns the
//     durable attempt budget; it must not be bypassed or duplicated.
//
// Worst-case external call amplification per durable job attempt is therefore
// 1+MaxInCallRetries provider HTTP requests. With the default max_attempts=3
// from #44 and MaxInCallRetries=2, a single logical WRITER job performs at
// most 9 provider HTTP calls before terminal failure.
//
// TTS uses the same in-call retry contract at the HTTP layer. Segment-level
// orchestration remains synchronous today; durable TTS job retries are out of
// scope for this contract and must not multiply in-call retries silently.
package providers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

const (
	// MaxInCallRetries is the number of retries after the initial HTTP attempt
	// for one logical provider call (generateContent for text or audio).
	MaxInCallRetries = 2

	providerName = "gemini"
)

// Operation identifies a bounded provider operation class for observability.
type Operation string

const (
	OpGenerateContent Operation = "generate_content"
)

// CallOutcome is a bounded result label for provider observability.
type CallOutcome string

const (
	CallSucceeded CallOutcome = "success"
	CallFailed    CallOutcome = "failure"
)

// RetryDecision is a bounded label describing whether an in-call retry ran.
type RetryDecision string

const (
	RetryNone       RetryDecision = "none"
	RetryScheduled  RetryDecision = "scheduled"
	RetryExhausted  RetryDecision = "exhausted"
	RetryNotAllowed RetryDecision = "not_allowed"
)

// CallEvent is emitted after each provider HTTP attempt for metrics/logging.
// It intentionally excludes prompts, tokens, and raw provider bodies.
type CallEvent struct {
	Provider      string
	Operation     Operation
	Outcome       CallOutcome
	FailureClass  string
	FailureCode   string
	HTTPStatus    int
	RetryDecision RetryDecision
	Attempt       int
	Latency       time.Duration
}

// CallObserver receives bounded provider call events. It is optional.
type CallObserver func(CallEvent)

var callObserver CallObserver

// SetCallObserver registers a metrics/logging hook for provider attempts.
func SetCallObserver(observer CallObserver) {
	callObserver = observer
}

func observeCall(event CallEvent) {
	if callObserver == nil {
		return
	}
	callObserver(event)
}

func classifyHTTPStatus(status int) (class, code string) {
	switch {
	case status == http.StatusTooManyRequests:
		return "TRANSIENT", "PROVIDER_RATE_LIMITED"
	case status == http.StatusRequestTimeout, status == http.StatusGatewayTimeout:
		return "TRANSIENT", "PROVIDER_TIMEOUT"
	case status >= http.StatusInternalServerError:
		return "TRANSIENT", "PROVIDER_UNAVAILABLE"
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return "PERMANENT", "PROVIDER_AUTH"
	case status == http.StatusBadRequest, status == http.StatusNotFound,
		status == http.StatusUnprocessableEntity, status == http.StatusNotImplemented:
		return "PERMANENT", "PROVIDER_CONFIG"
	default:
		return "PERMANENT", "PROVIDER_HTTP_ERROR"
	}
}

func parseRetryAfter(header http.Header) time.Duration {
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second
	}
	if when, err := http.ParseTime(value); err == nil {
		delay := time.Until(when)
		if delay > 0 {
			return delay
		}
	}
	return 0
}

func retryDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter + jitterDuration(retryAfter / 10)
	}
	var base time.Duration
	switch attempt {
	case 1:
		base = 2 * time.Second
	case 2:
		base = 5 * time.Second
	default:
		base = 15 * time.Second
	}
	return base + jitterDuration(base / 4)
}

func jitterDuration(base time.Duration) time.Duration {
	if base <= 0 {
		return 0
	}
	// Deterministic pseudo-jitter keeps tests stable while avoiding tight loops.
	return time.Duration((base.Milliseconds()%500)+100) * time.Millisecond
}

func classifiedProviderError(class, code string, cause error) error {
	return &generation.ClassifiedError{
		Class: class,
		Code:  code,
		Err:   safeProviderError(code, cause),
	}
}

func safeProviderError(code string, cause error) error {
	_ = cause
	switch code {
	case "PROVIDER_RATE_LIMITED":
		return errors.New("provider rate limited")
	case "PROVIDER_TIMEOUT":
		return errors.New("provider request timed out")
	case "PROVIDER_UNAVAILABLE":
		return errors.New("provider temporarily unavailable")
	case "PROVIDER_NETWORK":
		return errors.New("provider network failure")
	case "PROVIDER_AUTH":
		return errors.New("provider authentication failed")
	case "PROVIDER_CONFIG":
		return errors.New("provider configuration rejected request")
	case "PROVIDER_MALFORMED":
		return errors.New("provider returned malformed response")
	case "PROVIDER_RETRY_EXHAUSTED":
		return errors.New("provider in-call retries exhausted")
	case "PROVIDER_HTTP_ERROR":
		return errors.New("provider request failed")
	default:
		return fmt.Errorf("provider error: %s", code)
	}
}

func boundedFailureClass(class string) string {
	switch class {
	case "TRANSIENT", "PERMANENT":
		return class
	default:
		return "other"
	}
}

func boundedFailureCode(code string) string {
	switch code {
	case "PROVIDER_RATE_LIMITED", "PROVIDER_TIMEOUT", "PROVIDER_UNAVAILABLE",
		"PROVIDER_NETWORK", "PROVIDER_AUTH", "PROVIDER_CONFIG", "PROVIDER_MALFORMED",
		"PROVIDER_RETRY_EXHAUSTED", "PROVIDER_HTTP_ERROR":
		return code
	default:
		return "other"
	}
}

func retryDecisionForSuccess(attempt int) RetryDecision {
	if attempt == 0 {
		return RetryNone
	}
	return RetryScheduled
}

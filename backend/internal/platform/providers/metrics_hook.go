package providers

import (
	"context"
	"time"

	platformmetrics "github.com/synaudio/synaudio/backend/internal/platform/metrics"
)

// WireMetrics connects bounded provider call events to the metrics registry.
func WireMetrics(registry *platformmetrics.Registry) {
	if registry == nil {
		return
	}
	SetCallObserver(func(event CallEvent) {
		registry.ObserveProviderCall(
			event.Provider,
			string(event.Operation),
			string(event.Outcome),
			event.FailureClass,
			string(event.RetryDecision),
			event.Latency,
		)
	})
}

// ResetCallObserver clears the provider observer hook. Tests only.
func ResetCallObserver() {
	callObserver = nil
}

// SetProviderSleepForTests overrides in-call retry backoff sleep. Tests only.
func SetProviderSleepForTests(sleep func(context.Context, time.Duration) error) {
	if sleep == nil {
		providerSleep = sleepWithContext
		return
	}
	providerSleep = sleep
}

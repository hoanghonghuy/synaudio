package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

var providerSleep = sleepWithContext

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type attemptResult struct {
	response   geminiResponse
	statusCode int
	headers    http.Header
	err        error
}

func (c *geminiClient) generate(ctx context.Context, prompt string, generationConfig *geminiGenerationConfig) (geminiResponse, error) {
	var lastClass string
	var lastRetryAfter time.Duration

	for attempt := 0; attempt <= MaxInCallRetries; attempt++ {
		if attempt > 0 {
			delay := retryDelay(attempt, lastRetryAfter)
			if err := providerSleep(ctx, delay); err != nil {
				return geminiResponse{}, err
			}
		}

		started := time.Now()
		result := c.doGenerateOnce(ctx, prompt, generationConfig)
		latency := time.Since(started)

		if result.err == nil {
			observeCall(CallEvent{
				Provider:      providerName,
				Operation:     OpGenerateContent,
				Outcome:       CallSucceeded,
				FailureClass:  "none",
				FailureCode:   "none",
				HTTPStatus:    result.statusCode,
				RetryDecision: retryDecisionForSuccess(attempt),
				Attempt:       attempt + 1,
				Latency:       latency,
			})
			return result.response, nil
		}

		if ctx.Err() != nil && errors.Is(result.err, ctx.Err()) {
			return geminiResponse{}, result.err
		}

		class, code, retryable, retryAfter := classifyAttemptFailure(result)
		lastClass = class
		lastRetryAfter = retryAfter

		decision := RetryNotAllowed
		if retryable {
			if attempt < MaxInCallRetries {
				decision = RetryScheduled
			} else {
				decision = RetryExhausted
				code = "PROVIDER_RETRY_EXHAUSTED"
			}
		}

		observeCall(CallEvent{
			Provider:      providerName,
			Operation:     OpGenerateContent,
			Outcome:       CallFailed,
			FailureClass:  boundedFailureClass(class),
			FailureCode:   boundedFailureCode(code),
			HTTPStatus:    result.statusCode,
			RetryDecision: decision,
			Attempt:       attempt + 1,
			Latency:       latency,
		})

		if !retryable || attempt == MaxInCallRetries {
			return geminiResponse{}, classifiedProviderError(class, code, result.err)
		}
	}

	return geminiResponse{}, classifiedProviderError(lastClass, "PROVIDER_RETRY_EXHAUSTED", errors.New("provider in-call retries exhausted"))
}

func classifyAttemptFailure(result attemptResult) (class, code string, retryable bool, retryAfter time.Duration) {
	var classified *generation.ClassifiedError
	if errors.As(result.err, &classified) {
		return classified.Class, classified.Code, classified.Class == "TRANSIENT", 0
	}

	if result.statusCode > 0 {
		class, code = classifyHTTPStatus(result.statusCode)
		retryAfter = parseRetryAfter(result.headers)
		return class, code, class == "TRANSIENT", retryAfter
	}

	if errors.Is(result.err, context.Canceled) || errors.Is(result.err, context.DeadlineExceeded) {
		return "", "", false, 0
	}

	var netErr net.Error
	if errors.As(result.err, &netErr) && netErr.Timeout() {
		return "TRANSIENT", "PROVIDER_TIMEOUT", true, 0
	}
	return "TRANSIENT", "PROVIDER_NETWORK", true, 0
}

func (c *geminiClient) doGenerateOnce(ctx context.Context, prompt string, generationConfig *geminiGenerationConfig) attemptResult {
	payload, err := json.Marshal(geminiRequest{
		Contents:         []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: generationConfig,
	})
	if err != nil {
		return attemptResult{err: fmt.Errorf("encode Gemini request: %w", err)}
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return attemptResult{err: fmt.Errorf("create Gemini request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return attemptResult{err: err}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return attemptResult{statusCode: resp.StatusCode, headers: resp.Header, err: fmt.Errorf("read Gemini response: %w", err)}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return attemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        fmt.Errorf("provider HTTP %d", resp.StatusCode),
		}
	}

	var out geminiResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return attemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", err),
		}
	}

	if len(out.Candidates) == 0 {
		return attemptResult{
			statusCode: resp.StatusCode,
			headers:    resp.Header,
			err:        classifiedProviderError("PERMANENT", "PROVIDER_MALFORMED", errors.New("provider returned no candidates")),
		}
	}
	return attemptResult{response: out, statusCode: resp.StatusCode, headers: resp.Header}
}

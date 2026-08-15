package generation

import (
	"context"
	"testing"
)

func TestAnalyzeDurationPassesAboveMinimum(t *testing.T) {
	svc := NewService(newFakeStore(), WithDurationAnalyzer(NewMockDurationAnalyzer()))

	// ~4500 words ≈ 30 minutes at 150 wpm.
	text := repeatWords(4500)

	analysis, err := svc.AnalyzeDuration(context.Background(), "c1", "rev1", text)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if analysis.Outcome != "PASS" {
		t.Fatalf("expected PASS, got %q", analysis.Outcome)
	}
	if analysis.EstimatedMinutes < 20 {
		t.Fatalf("expected >= 20 minutes, got %d", analysis.EstimatedMinutes)
	}
}

func TestAnalyzeDurationBlocksBelowMinimum(t *testing.T) {
	svc := NewService(newFakeStore(), WithDurationAnalyzer(NewMockDurationAnalyzer()))

	// ~1500 words ≈ 10 minutes.
	text := repeatWords(1500)

	analysis, err := svc.AnalyzeDuration(context.Background(), "c1", "rev1", text)
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if analysis.Outcome != "OVERRIDABLE_BLOCK" {
		t.Fatalf("expected OVERRIDABLE_BLOCK, got %q", analysis.Outcome)
	}
}

func TestAnalyzeDurationWithoutAnalyzerFails(t *testing.T) {
	svc := NewService(newFakeStore())

	if _, err := svc.AnalyzeDuration(context.Background(), "c1", "rev1", "short"); err == nil {
		t.Fatal("expected error when duration analyzer not configured")
	}
}

func repeatWords(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += "word "
	}
	return out
}

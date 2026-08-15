package generation

import (
	"context"
	"errors"
	"strings"
)

// DurationAnalyzer estimates narration duration and validates against policy.
type DurationAnalyzer interface {
	AnalyzeDuration(ctx context.Context, in DurationInput) (DurationOutput, error)
}

// DurationInput is the text to analyze.
type DurationInput struct {
	Text string
}

// DurationOutput is the result of duration analysis.
type DurationOutput struct {
	EstimatedMinutes int
	Outcome          string
	Report           map[string]any
}

// MockDurationAnalyzer estimates duration at 150 words per minute.
type MockDurationAnalyzer struct {
	MinMinutes int
	TargetMinutes int
}

func NewMockDurationAnalyzer() *MockDurationAnalyzer {
	return &MockDurationAnalyzer{MinMinutes: 20, TargetMinutes: 30}
}

func (a *MockDurationAnalyzer) AnalyzeDuration(_ context.Context, in DurationInput) (DurationOutput, error) {
	words := len(strings.Fields(in.Text))
	minutes := words / 150
	if minutes < 1 {
		minutes = 1
	}

	outcome := "PASS"
	if minutes < a.MinMinutes {
		outcome = "OVERRIDABLE_BLOCK"
	}

	return DurationOutput{
		EstimatedMinutes: minutes,
		Outcome:          outcome,
		Report: map[string]any{
			"word_count":        words,
			"estimated_minutes": minutes,
			"min_minutes":       a.MinMinutes,
			"target_minutes":    a.TargetMinutes,
		},
	}, nil
}

// AnalyzeDuration runs duration analysis on a content revision's text.
func (s *Service) AnalyzeDuration(ctx context.Context, chapterID, revisionID, text string) (DurationOutput, error) {
	if s.durationAnalyzer == nil {
		return DurationOutput{}, errors.New("duration analyzer not configured")
	}

	out, err := s.durationAnalyzer.AnalyzeDuration(ctx, DurationInput{Text: text})
	if err != nil {
		return DurationOutput{}, err
	}

	// Record the review.
	_, _ = s.CreateChapterReview(ctx, chapterID, revisionID, "DURATION", out.Outcome, out.Report)

	return out, nil
}

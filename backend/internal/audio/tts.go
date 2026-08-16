package audio

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrTTSSegmentNotFound = errors.New("tts segment not found")
)

// TTSProvider is the abstraction for text-to-speech providers.
type TTSProvider interface {
	Synthesize(ctx context.Context, in TTSInput) (TTSOutput, error)
}

// TTSInput is the input for a TTS synthesis call.
type TTSInput struct {
	Text    string
	VoiceID string
}

// TTSOutput is the result of a TTS synthesis call.
type TTSOutput struct {
	AudioData []byte
	DurationMs int
	Provider  string
	Model     string
}

// MockTTS is a deterministic TTS provider for development/testing.
type MockTTS struct{}

func NewMockTTS() *MockTTS {
	return &MockTTS{}
}

// Synthesize returns deterministic audio bytes proportional to text length.
func (MockTTS) Synthesize(_ context.Context, in TTSInput) (TTSOutput, error) {
	words := len(strings.Fields(in.Text))
	durationMs := words * 400
	if durationMs < 400 {
		durationMs = 400
	}
	return TTSOutput{
		AudioData:  []byte("MOCK-AUDIO"),
		DurationMs: durationMs,
		Provider:   "mock",
		Model:      "mock-tts-v1",
	}, nil
}

// TTSSegment is a single synthesis unit of a narration revision.
type TTSSegment struct {
	ID                  string
	NarrationRevisionID string
	SegmentNo           int
	Text                string
	Status              string
	Provider            string
	Model               string
	VoiceID             string
	DurationMs          int
	TempStorageKey      string
}

// CreateTTSSegments splits a narration script into sentence-level segments.
func (s *Service) CreateTTSSegments(ctx context.Context, narrationRevisionID string) ([]TTSSegment, error) {
	nar, err := s.store.GetNarrationRevision(ctx, narrationRevisionID)
	if err != nil {
		return nil, err
	}

	sentences := splitSentences(nar.Script)
	segments := make([]TTSSegment, 0, len(sentences))
	for i, text := range sentences {
		seg := TTSSegment{
			ID:                  uuid.NewString(),
			NarrationRevisionID: narrationRevisionID,
			SegmentNo:           i + 1,
			Text:                text,
			Status:              "PENDING",
			VoiceID:             nar.VoiceID,
		}
		created, err := s.store.CreateTTSSegment(ctx, seg)
		if err != nil {
			return nil, err
		}
		segments = append(segments, created)
	}

	return segments, nil
}

// SynthesizeSegment synthesizes a single segment using the TTS provider.
func (s *Service) SynthesizeSegment(ctx context.Context, segmentID string) (TTSSegment, error) {
	if s.tts == nil {
		return TTSSegment{}, errors.New("tts not configured")
	}

	seg, err := s.store.GetTTSSegment(ctx, segmentID)
	if err != nil {
		return TTSSegment{}, err
	}

	out, err := s.tts.Synthesize(ctx, TTSInput{Text: seg.Text, VoiceID: seg.VoiceID})
	if err != nil {
		return TTSSegment{}, err
	}

	seg.Status = "SYNTHESIZED"
	seg.Provider = out.Provider
	seg.Model = out.Model
	seg.DurationMs = out.DurationMs
	seg.TempStorageKey = "tts/" + seg.ID + ".mp3"

	return s.store.UpdateTTSSegment(ctx, seg)
}

// splitSentences splits a script into sentences on ., !, ? boundaries.
func splitSentences(script string) []string {
	var sentences []string
	var current strings.Builder
	for _, r := range script {
		current.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			s := strings.TrimSpace(current.String())
			if s != "" {
				sentences = append(sentences, s)
			}
			current.Reset()
		}
	}
	if tail := strings.TrimSpace(current.String()); tail != "" {
		sentences = append(sentences, tail)
	}
	return sentences
}

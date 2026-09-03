package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

func TestBuildAISelectsMockExplicitly(t *testing.T) {
	set, err := BuildAI(config.Config{AIMode: "mock"})
	if err != nil {
		t.Fatalf("build mock AI: %v", err)
	}
	if _, ok := set.TextAI.(*generation.MockTextAI); !ok {
		t.Fatalf("expected mock text AI, got %T", set.TextAI)
	}
	if set.Architect == nil || set.MemoryExtractor == nil {
		t.Fatal("expected all AI ports to be configured")
	}
}

func TestBuildAIRejectsUnsupportedMode(t *testing.T) {
	if _, err := BuildAI(config.Config{AIMode: "other"}); err == nil {
		t.Fatal("expected unsupported AI mode to fail")
	}
}

func TestBuildAIRejectsMissingGeminiCredential(t *testing.T) {
	_, err := BuildAI(config.Config{AIMode: "gemini", GeminiTextModel: "gemini-test"})
	if err == nil {
		t.Fatal("expected missing Gemini API key to fail")
	}
}

func TestBuildAIConstructsGeminiWithoutNetworkCall(t *testing.T) {
	set, err := BuildAI(config.Config{
		AIMode:          "gemini",
		GeminiAPIKey:    "test-key",
		GeminiTextModel: "gemini-test",
	})
	if err != nil {
		t.Fatalf("build Gemini AI: %v", err)
	}
	if set.Architect == nil || set.MemoryExtractor == nil || set.TextAI == nil {
		t.Fatal("expected all Gemini AI ports to be configured")
	}
}

func TestBuildTTSRejectsUnsupportedMode(t *testing.T) {
	if _, err := BuildTTS(config.Config{TTSMode: "other"}); err == nil {
		t.Fatal("expected unsupported TTS mode to fail")
	}
}

func TestBuildTTSRejectsMissingGeminiCredential(t *testing.T) {
	_, err := BuildTTS(config.Config{
		TTSMode:        "gemini",
		GeminiTTSModel: "gemini-tts-test",
		GeminiTTSVoice: "Kore",
	})
	if err == nil {
		t.Fatal("expected missing Gemini API key to fail")
	}
}

func TestBuildTTSConstructsGeminiWithoutNetworkCall(t *testing.T) {
	provider, err := BuildTTS(config.Config{
		TTSMode:        "gemini",
		GeminiAPIKey:   "test-key",
		GeminiTTSModel: "gemini-tts-test",
		GeminiTTSVoice: "Kore",
	})
	if err != nil {
		t.Fatalf("build Gemini TTS: %v", err)
	}
	if provider == nil {
		t.Fatal("expected Gemini TTS provider")
	}
}

func TestGeminiTTSUsesConfiguredProviderVoiceNotLogicalVoiceID(t *testing.T) {
	var request geminiRequest
	client := &geminiClient{
		apiKey: "test-key",
		model:  "gemini-tts-test",
		http: &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
				t.Fatalf("decode Gemini request: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"candidates":[{"content":{"parts":[{"inlineData":{"mimeType":"audio/L16;codec=pcm;rate=24000","data":"AAAAAA=="}}]}}]}`,
				)),
				Request: req,
			}, nil
		})},
	}

	provider := &geminiTTS{client: client, voice: "Kore"}
	if _, err := provider.Synthesize(context.Background(), audio.TTSInput{
		Text:    "Hello",
		VoiceID: "logical-narrator-voice-id",
	}); err != nil {
		t.Fatalf("synthesize: %v", err)
	}

	if request.GenerationConfig == nil || request.GenerationConfig.SpeechConfig == nil {
		t.Fatal("expected speech configuration in Gemini request")
	}
	voice := request.GenerationConfig.SpeechConfig.VoiceConfig.PrebuiltVoiceConfig.VoiceName
	if voice != "Kore" {
		t.Fatalf("expected configured provider voice Kore, got %q", voice)
	}
	if voice == "logical-narrator-voice-id" {
		t.Fatal("logical Synaudio voice ID must not be forwarded as Gemini voiceName")
	}
}

func TestWrapPCMProducesWAVHeader(t *testing.T) {
	wav := wrapPCM16Mono24kWAV([]byte{0, 0, 1, 0})
	if len(wav) < 44 {
		t.Fatalf("expected WAV header, got %d bytes", len(wav))
	}
	if string(wav[:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		t.Fatalf("expected RIFF/WAVE header, got %q/%q", wav[:4], wav[8:12])
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

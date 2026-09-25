package providers_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/platform/providers"
)

// makeSampleWAV returns minimal valid RIFF WAVE 16-bit mono 16000Hz header with 3200 bytes of PCM (100ms).
func makeSampleWAV() []byte {
	pcmSize := uint32(3200) // 100ms at 16000Hz * 2 bytes/sample
	wav := make([]byte, 44+pcmSize)
	copy(wav[0:4], "RIFF")
	fileSizeMinus8 := 36 + pcmSize
	wav[4] = byte(fileSizeMinus8)
	wav[5] = byte(fileSizeMinus8 >> 8)
	wav[6] = byte(fileSizeMinus8 >> 16)
	wav[7] = byte(fileSizeMinus8 >> 24)
	copy(wav[8:12], "WAVE")
	copy(wav[12:16], "fmt ")
	wav[16] = 16 // SubChunk1Size (16 for PCM)
	wav[20] = 1  // AudioFormat (1 for PCM)
	wav[22] = 1  // NumChannels (1 mono)
	sampleRate := uint32(16000)
	wav[24] = byte(sampleRate)
	wav[25] = byte(sampleRate >> 8)
	wav[26] = byte(sampleRate >> 16)
	wav[27] = byte(sampleRate >> 24)
	byteRate := sampleRate * 2 // 32000
	wav[28] = byte(byteRate)
	wav[29] = byte(byteRate >> 8)
	wav[30] = byte(byteRate >> 16)
	wav[31] = byte(byteRate >> 24)
	wav[32] = 2  // BlockAlign
	wav[34] = 16 // BitsPerSample
	copy(wav[36:40], "data")
	wav[40] = byte(pcmSize)
	wav[41] = byte(pcmSize >> 8)
	wav[42] = byte(pcmSize >> 16)
	wav[43] = byte(pcmSize >> 24)
	return wav
}

func TestSynStudioTTSSynthesizeSuccess(t *testing.T) {
	sampleWAV := makeSampleWAV()
	sampleB64 := base64.StdEncoding.EncodeToString(sampleWAV)

	var receivedMethod string
	var receivedPath string
	var receivedPayload map[string]any

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedMethod = r.Method
		receivedPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&receivedPayload)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":        true,
			"audio_b64": sampleB64,
		})
	}))
	defer server.Close()

	tts, err := providers.NewSynStudioTTS(server.URL, "ngochuyen")
	if err != nil {
		t.Fatalf("unexpected NewSynStudioTTS error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	out, err := tts.Synthesize(ctx, audio.TTSInput{
		Text:    "Xin chào các bạn",
		VoiceID: "ngochuyen",
	})
	if err != nil {
		t.Fatalf("unexpected Synthesize error: %v", err)
	}

	if receivedMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", receivedMethod)
	}
	if receivedPath != "/v1/tts/synthesize" {
		t.Errorf("expected /v1/tts/synthesize, got %s", receivedPath)
	}
	if receivedPayload["text"] != "Xin chào các bạn" {
		t.Errorf("expected text 'Xin chào các bạn', got %v", receivedPayload["text"])
	}
	if receivedPayload["voice"] != "ngochuyen" {
		t.Errorf("expected voice 'ngochuyen', got %v", receivedPayload["voice"])
	}

	if len(out.AudioData) != len(sampleWAV) {
		t.Errorf("expected %d audio bytes, got %d", len(sampleWAV), len(out.AudioData))
	}
	if out.Provider != "synstudio" {
		t.Errorf("expected provider 'synstudio', got %q", out.Provider)
	}
	if out.Model != "piper" {
		t.Errorf("expected model 'piper', got %q", out.Model)
	}
	if out.Format != "wav" {
		t.Errorf("expected format 'wav', got %q", out.Format)
	}
	if out.DurationMs <= 0 {
		t.Errorf("expected positive duration, got %d", out.DurationMs)
	}
}

func TestSynStudioTTSSynthesizeServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok":    false,
			"error": "worker out of memory",
		})
	}))
	defer server.Close()

	tts, err := providers.NewSynStudioTTS(server.URL, "ngochuyen")
	if err != nil {
		t.Fatalf("unexpected NewSynStudioTTS error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = tts.Synthesize(ctx, audio.TTSInput{Text: "Test", VoiceID: "ngochuyen"})
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

func TestSynStudioTTSSynthesizeRejectsEmptyText(t *testing.T) {
	tts, err := providers.NewSynStudioTTS("http://localhost:8080", "ngochuyen")
	if err != nil {
		t.Fatalf("unexpected NewSynStudioTTS error: %v", err)
	}

	_, err = tts.Synthesize(context.Background(), audio.TTSInput{Text: "   ", VoiceID: "ngochuyen"})
	if err == nil {
		t.Fatal("expected error for empty text, got nil")
	}
}

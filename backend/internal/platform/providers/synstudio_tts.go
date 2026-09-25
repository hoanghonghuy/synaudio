package providers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

// SynStudioTTS integrates with the SynStudio / Piper TTS engine service.
type SynStudioTTS struct {
	endpoint     string
	defaultVoice string
	client       *http.Client
}

// NewSynStudioTTS creates a new SynStudio TTS provider client.
func NewSynStudioTTS(endpoint, defaultVoice string) (*SynStudioTTS, error) {
	endpoint = strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if endpoint == "" {
		return nil, errors.New("SynStudio TTS endpoint is required")
	}
	defaultVoice = strings.TrimSpace(defaultVoice)
	if defaultVoice == "" {
		defaultVoice = "ngochuyen"
	}
	return &SynStudioTTS{
		endpoint:     endpoint,
		defaultVoice: defaultVoice,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

type synStudioSynthesizeRequest struct {
	Text  string  `json:"text"`
	Voice string  `json:"voice"`
	Speed float64 `json:"speed"`
}

type synStudioSynthesizeResponse struct {
	OK       bool   `json:"ok"`
	AudioB64 string `json:"audio_b64"`
	Error    string `json:"error"`
}

// Synthesize calls the SynStudio worker endpoint to synthesize speech from text.
func (s *SynStudioTTS) Synthesize(ctx context.Context, in audio.TTSInput) (audio.TTSOutput, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return audio.TTSOutput{}, errors.New("text is required for synthesis")
	}

	voice := strings.TrimSpace(in.VoiceID)
	if voice == "" {
		voice = s.defaultVoice
	}

	payload := synStudioSynthesizeRequest{
		Text:  text,
		Voice: voice,
		Speed: 1.0,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return audio.TTSOutput{}, fmt.Errorf("marshal tts payload: %w", err)
	}

	reqURL := s.endpoint + "/v1/tts/synthesize"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return audio.TTSOutput{}, fmt.Errorf("create tts request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return audio.TTSOutput{}, fmt.Errorf("call tts worker: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return audio.TTSOutput{}, fmt.Errorf("read tts response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp synStudioSynthesizeResponse
		_ = json.Unmarshal(respBytes, &errResp)
		if errResp.Error != "" {
			return audio.TTSOutput{}, fmt.Errorf("tts worker error (status %d): %s", resp.StatusCode, errResp.Error)
		}
		return audio.TTSOutput{}, fmt.Errorf("tts worker returned unexpected status %d", resp.StatusCode)
	}

	var res synStudioSynthesizeResponse
	if err := json.Unmarshal(respBytes, &res); err != nil {
		return audio.TTSOutput{}, fmt.Errorf("decode tts response: %w", err)
	}
	if !res.OK || res.AudioB64 == "" {
		errMsg := res.Error
		if errMsg == "" {
			errMsg = "empty audio in worker response"
		}
		return audio.TTSOutput{}, fmt.Errorf("tts synthesis failed: %s", errMsg)
	}

	audioBytes, err := base64.StdEncoding.DecodeString(res.AudioB64)
	if err != nil {
		return audio.TTSOutput{}, fmt.Errorf("decode audio base64: %w", err)
	}

	durationMs := parseWAVDurationMs(audioBytes)
	if durationMs <= 0 {
		// Fallback proportional to text length
		durationMs = len(strings.Fields(text)) * 400
		if durationMs < 400 {
			durationMs = 400
		}
	}

	return audio.TTSOutput{
		AudioData:  audioBytes,
		DurationMs: durationMs,
		Provider:   "synstudio",
		Model:      "piper",
		Format:     "wav",
	}, nil
}

// parseWAVDurationMs calculates duration in milliseconds from standard RIFF WAVE bytes.
func parseWAVDurationMs(wav []byte) int {
	if len(wav) < 44 || string(wav[0:4]) != "RIFF" || string(wav[8:12]) != "WAVE" {
		return 0
	}

	byteRate := binary.LittleEndian.Uint32(wav[28:32])
	if byteRate == 0 {
		return 0
	}

	// Look for the "data" subchunk
	dataSize := uint32(0)
	offset := 12
	for offset+8 <= len(wav) {
		chunkID := string(wav[offset : offset+4])
		chunkSize := binary.LittleEndian.Uint32(wav[offset+4 : offset+8])
		if chunkID == "data" {
			dataSize = chunkSize
			break
		}
		offset += 8 + int(chunkSize)
	}
	if dataSize == 0 && len(wav) > 44 {
		dataSize = uint32(len(wav) - 44)
	}

	return int((float64(dataSize) / float64(byteRate)) * 1000)
}

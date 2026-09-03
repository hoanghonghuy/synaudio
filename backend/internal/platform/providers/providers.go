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
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/planning"
	"github.com/synaudio/synaudio/backend/internal/platform/config"
)

// AISet groups the AI-backed ports used by the API and worker composition roots.
type AISet struct {
	Architect       planning.Architect
	MemoryExtractor planning.MemoryExtractor
	TextAI          generation.TextAIProvider
}

// BuildAI selects the configured AI implementation without silently falling back.
func BuildAI(cfg config.Config) (AISet, error) {
	switch cfg.AIMode {
	case "mock":
		return AISet{
			Architect:       planning.NewMockArchitect(),
			MemoryExtractor: planning.NewMockMemoryExtractor(),
			TextAI:          generation.NewMockTextAI(),
		}, nil
	case "gemini":
		client, err := newGeminiClient(cfg.GeminiAPIKey, cfg.GeminiTextModel)
		if err != nil {
			return AISet{}, fmt.Errorf("configure gemini AI provider: %w", err)
		}
		adapter := &geminiAI{client: client}
		return AISet{Architect: adapter, MemoryExtractor: adapter, TextAI: adapter}, nil
	default:
		return AISet{}, fmt.Errorf("unsupported AI_MODE %q", cfg.AIMode)
	}
}

// BuildTTS selects the configured TTS implementation without silently falling back.
func BuildTTS(cfg config.Config) (audio.TTSProvider, error) {
	switch cfg.TTSMode {
	case "mock":
		return audio.NewMockTTS(), nil
	case "gemini":
		client, err := newGeminiClient(cfg.GeminiAPIKey, cfg.GeminiTTSModel)
		if err != nil {
			return nil, fmt.Errorf("configure gemini TTS provider: %w", err)
		}
		if strings.TrimSpace(cfg.GeminiTTSVoice) == "" {
			return nil, errors.New("GEMINI_TTS_VOICE is required when TTS_MODE=gemini")
		}
		return &geminiTTS{client: client, voice: cfg.GeminiTTSVoice}, nil
	default:
		return nil, fmt.Errorf("unsupported TTS_MODE %q", cfg.TTSMode)
	}
}

type geminiClient struct {
	apiKey string
	model  string
	http   *http.Client
}

func newGeminiClient(apiKey, model string) (*geminiClient, error) {
	apiKey = strings.TrimSpace(apiKey)
	model = strings.TrimSpace(model)
	if apiKey == "" {
		return nil, errors.New("GEMINI_API_KEY is required")
	}
	if model == "" {
		return nil, errors.New("Gemini model is required")
	}
	return &geminiClient{
		apiKey: apiKey,
		model:  model,
		http:   &http.Client{Timeout: 60 * time.Second},
	}, nil
}

type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string            `json:"text,omitempty"`
	InlineData *geminiInlineData `json:"inlineData,omitempty"`
}

type geminiInlineData struct {
	MIMEType string `json:"mimeType,omitempty"`
	Data     string `json:"data,omitempty"`
}

type geminiGenerationConfig struct {
	ResponseMIMEType string              `json:"responseMimeType,omitempty"`
	ResponseModalities []string          `json:"responseModalities,omitempty"`
	SpeechConfig     *geminiSpeechConfig `json:"speechConfig,omitempty"`
}

type geminiSpeechConfig struct {
	VoiceConfig geminiVoiceConfig `json:"voiceConfig"`
}

type geminiVoiceConfig struct {
	PrebuiltVoiceConfig geminiPrebuiltVoiceConfig `json:"prebuiltVoiceConfig"`
}

type geminiPrebuiltVoiceConfig struct {
	VoiceName string `json:"voiceName"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (c *geminiClient) generate(ctx context.Context, prompt string, generationConfig *geminiGenerationConfig) (geminiResponse, error) {
	payload, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: prompt}}}},
		GenerationConfig: generationConfig,
	})
	if err != nil {
		return geminiResponse{}, fmt.Errorf("encode Gemini request: %w", err)
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", c.model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return geminiResponse{}, fmt.Errorf("create Gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return geminiResponse{}, fmt.Errorf("Gemini request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return geminiResponse{}, fmt.Errorf("read Gemini response: %w", err)
	}

	var out geminiResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return geminiResponse{}, fmt.Errorf("decode Gemini response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(string(body))
		if out.Error != nil && strings.TrimSpace(out.Error.Message) != "" {
			message = out.Error.Message
		}
		return geminiResponse{}, fmt.Errorf("Gemini returned HTTP %d: %s", resp.StatusCode, message)
	}
	if len(out.Candidates) == 0 {
		return geminiResponse{}, errors.New("Gemini returned no candidates")
	}
	return out, nil
}

func responseText(resp geminiResponse) (string, error) {
	for _, part := range resp.Candidates[0].Content.Parts {
		if strings.TrimSpace(part.Text) != "" {
			return strings.TrimSpace(part.Text), nil
		}
	}
	return "", errors.New("Gemini returned no text")
}

type geminiAI struct {
	client *geminiClient
}

func (g *geminiAI) GenerateText(ctx context.Context, in generation.TextAIInput) (generation.TextAIOutput, error) {
	resp, err := g.client.generate(ctx, in.Prompt, nil)
	if err != nil {
		return generation.TextAIOutput{}, err
	}
	text, err := responseText(resp)
	if err != nil {
		return generation.TextAIOutput{}, err
	}
	return generation.TextAIOutput{Text: text, Provider: "gemini", Model: g.client.model}, nil
}

func (g *geminiAI) ProposeFoundation(ctx context.Context, in planning.FoundationInput) (planning.FoundationProposal, error) {
	prompt := fmt.Sprintf(`Create a story foundation for this premise: %s
Return JSON only with this shape:
{"bible":{},"ending":{},"arcs":[{}],"characters":[{"name":"","importance":"MAJOR","profile":{}}]}
Use only MAJOR or MINOR for character importance.`, in.Premise)
	resp, err := g.client.generate(ctx, prompt, &geminiGenerationConfig{ResponseMIMEType: "application/json"})
	if err != nil {
		return planning.FoundationProposal{}, err
	}
	text, err := responseText(resp)
	if err != nil {
		return planning.FoundationProposal{}, err
	}
	var proposal struct {
		Bible      map[string]any `json:"bible"`
		Ending     map[string]any `json:"ending"`
		Arcs       []map[string]any `json:"arcs"`
		Characters []planning.CharacterProposal `json:"characters"`
	}
	if err := json.Unmarshal([]byte(text), &proposal); err != nil {
		return planning.FoundationProposal{}, fmt.Errorf("decode Gemini foundation JSON: %w", err)
	}
	if proposal.Bible == nil || proposal.Ending == nil || len(proposal.Arcs) == 0 {
		return planning.FoundationProposal{}, errors.New("Gemini foundation response is incomplete")
	}
	return planning.FoundationProposal{
		Bible: proposal.Bible,
		Ending: proposal.Ending,
		Arcs: proposal.Arcs,
		Characters: proposal.Characters,
	}, nil
}

func (g *geminiAI) ExtractMemory(ctx context.Context, in planning.MemoryExtractionInput) (planning.MemoryExtraction, error) {
	prompt := fmt.Sprintf(`Extract durable canonical facts from the approved chapter content below.
Return JSON only with this shape:
{"facts":[{"subject_type":"CHARACTER","subject_id":"...","fact_type":"...","value":{}}]}
Story ID: %s
Chapter ID: %s
Content:
%s`, in.StoryID, in.ChapterID, in.ContentText)
	resp, err := g.client.generate(ctx, prompt, &geminiGenerationConfig{ResponseMIMEType: "application/json"})
	if err != nil {
		return planning.MemoryExtraction{}, err
	}
	text, err := responseText(resp)
	if err != nil {
		return planning.MemoryExtraction{}, err
	}
	var extraction struct {
		Facts []planning.ExtractedFact `json:"facts"`
	}
	if err := json.Unmarshal([]byte(text), &extraction); err != nil {
		return planning.MemoryExtraction{}, fmt.Errorf("decode Gemini memory JSON: %w", err)
	}
	return planning.MemoryExtraction{Facts: extraction.Facts}, nil
}

type geminiTTS struct {
	client *geminiClient
	voice  string
}

func (g *geminiTTS) Synthesize(ctx context.Context, in audio.TTSInput) (audio.TTSOutput, error) {
	voice := strings.TrimSpace(in.VoiceID)
	if voice == "" {
		voice = g.voice
	}
	resp, err := g.client.generate(ctx, in.Text, &geminiGenerationConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig: &geminiSpeechConfig{
			VoiceConfig: geminiVoiceConfig{PrebuiltVoiceConfig: geminiPrebuiltVoiceConfig{VoiceName: voice}},
		},
	})
	if err != nil {
		return audio.TTSOutput{}, err
	}

	for _, part := range resp.Candidates[0].Content.Parts {
		if part.InlineData == nil || part.InlineData.Data == "" {
			continue
		}
		pcm, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
		if err != nil {
			return audio.TTSOutput{}, fmt.Errorf("decode Gemini audio: %w", err)
		}
		if len(pcm) == 0 {
			return audio.TTSOutput{}, errors.New("Gemini returned empty audio")
		}
		return audio.TTSOutput{
			AudioData: wrapPCM16Mono24kWAV(pcm),
			DurationMs: len(pcm) * 1000 / (24000 * 2),
			Provider: "gemini",
			Model: g.client.model,
			Format: "wav",
		}, nil
	}
	return audio.TTSOutput{}, errors.New("Gemini returned no audio")
}

func wrapPCM16Mono24kWAV(pcm []byte) []byte {
	const sampleRate = 24000
	const channels = 1
	const bitsPerSample = 16
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8

	buf := bytes.NewBuffer(make([]byte, 0, 44+len(pcm)))
	buf.WriteString("RIFF")
	_ = binary.Write(buf, binary.LittleEndian, uint32(36+len(pcm)))
	buf.WriteString("WAVEfmt ")
	_ = binary.Write(buf, binary.LittleEndian, uint32(16))
	_ = binary.Write(buf, binary.LittleEndian, uint16(1))
	_ = binary.Write(buf, binary.LittleEndian, uint16(channels))
	_ = binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	_ = binary.Write(buf, binary.LittleEndian, uint32(byteRate))
	_ = binary.Write(buf, binary.LittleEndian, uint16(blockAlign))
	_ = binary.Write(buf, binary.LittleEndian, uint16(bitsPerSample))
	buf.WriteString("data")
	_ = binary.Write(buf, binary.LittleEndian, uint32(len(pcm)))
	buf.Write(pcm)
	return buf.Bytes()
}

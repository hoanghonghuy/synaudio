package providers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/planning"
)

func TestOpenAIGenerateTextSucceeds(t *testing.T) {
	var receivedAuth string
	var receivedReq openAIChatRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedReq)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			ID: "chatcmpl-123",
			Choices: []openAIChoice{
				{
					Message: openAIMessage{Role: "assistant", Content: "A dark storm approached the temple."},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := newOpenAIClient(srv.URL, "sk-test-secret", "test-model")
	if err != nil {
		t.Fatalf("unexpected newOpenAIClient err: %v", err)
	}
	adapter := &openAIAI{client: client}

	out, err := adapter.GenerateText(context.Background(), generation.TextAIInput{Prompt: "Write a scene"})
	if err != nil {
		t.Fatalf("unexpected GenerateText err: %v", err)
	}

	if out.Text != "A dark storm approached the temple." {
		t.Errorf("unexpected output text: %q", out.Text)
	}
	if out.Provider != "openai" || out.Model != "test-model" {
		t.Errorf("unexpected provider/model: %s/%s", out.Provider, out.Model)
	}
	if receivedAuth != "Bearer sk-test-secret" {
		t.Errorf("unexpected auth header: %q", receivedAuth)
	}
	if receivedReq.Stream {
		t.Errorf("expected stream=false, got true")
	}
}

func TestOpenAIProposeFoundationHandlesMarkdownFences(t *testing.T) {
	rawJSON := "```json\n" + `{
		"bible": {"world": "Fantasy"},
		"ending": {"resolution": "Victory"},
		"arcs": [{"name": "Arc 1"}],
		"characters": [
			{"name": "Hero", "importance": "MAJOR", "profile": "A brave warrior"},
			{"name": "Guide", "importance": "MINOR", "profile": {"role": "mentor"}}
		]
	}` + "\n```"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			ID: "chatcmpl-456",
			Choices: []openAIChoice{
				{
					Message: openAIMessage{Role: "assistant", Content: rawJSON},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := newOpenAIClient(srv.URL, "sk-test", "test-model")
	if err != nil {
		t.Fatalf("unexpected newOpenAIClient err: %v", err)
	}
	adapter := &openAIAI{client: client}

	proposal, err := adapter.ProposeFoundation(context.Background(), planning.FoundationInput{
		StoryID: "story-1",
		Premise: "A hero journeys to find a lost relic",
	})
	if err != nil {
		t.Fatalf("unexpected ProposeFoundation err: %v", err)
	}

	if proposal.Bible == nil || proposal.Ending == nil || len(proposal.Arcs) != 1 {
		t.Fatalf("incomplete proposal: %+v", proposal)
	}
	if len(proposal.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(proposal.Characters))
	}
	if proposal.Characters[0].Name != "Hero" || proposal.Characters[0].Importance != "MAJOR" {
		t.Errorf("unexpected character 0: %+v", proposal.Characters[0])
	}
	if proposal.Characters[0].Profile["description"] != "A brave warrior" {
		t.Errorf("expected string profile to be mapped into description map: %+v", proposal.Characters[0].Profile)
	}
}

func TestOpenAIExtractMemorySucceeds(t *testing.T) {
	rawJSON := `{
		"facts": [
			{
				"subject_type": "CHARACTER",
				"subject_id": "char-1",
				"fact_type": "status",
				"value": {"state": "injured"}
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			ID: "chatcmpl-789",
			Choices: []openAIChoice{
				{
					Message: openAIMessage{Role: "assistant", Content: rawJSON},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := newOpenAIClient(srv.URL, "sk-test", "test-model")
	if err != nil {
		t.Fatalf("unexpected newOpenAIClient err: %v", err)
	}
	adapter := &openAIAI{client: client}

	ext, err := adapter.ExtractMemory(context.Background(), planning.MemoryExtractionInput{
		StoryID:     "story-1",
		ChapterID:   "chap-1",
		ContentText: "The warrior was wounded in battle.",
	})
	if err != nil {
		t.Fatalf("unexpected ExtractMemory err: %v", err)
	}

	if len(ext.Facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(ext.Facts))
	}
	if ext.Facts[0].SubjectID != "char-1" || ext.Facts[0].FactType != "status" {
		t.Errorf("unexpected fact: %+v", ext.Facts[0])
	}
}

func TestOpenAIRetries429ThenSucceeds(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := attempts.Add(1)
		if att == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Rate limit reached"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(openAIChatResponse{
			ID: "chatcmpl-retry",
			Choices: []openAIChoice{
				{
					Message: openAIMessage{Role: "assistant", Content: "Success after retry"},
				},
			},
		})
	}))
	defer srv.Close()

	client, err := newOpenAIClient(srv.URL, "sk-test", "test-model")
	if err != nil {
		t.Fatalf("unexpected newOpenAIClient err: %v", err)
	}
	adapter := &openAIAI{client: client}

	out, err := adapter.GenerateText(context.Background(), generation.TextAIInput{Prompt: "Hello"})
	if err != nil {
		t.Fatalf("expected success after retry, got: %v", err)
	}
	if out.Text != "Success after retry" {
		t.Errorf("unexpected output: %q", out.Text)
	}
	if attempts.Load() != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts.Load())
	}
}

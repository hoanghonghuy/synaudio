package generation

import (
	"context"
	"testing"
)

func TestMockTextAIWritesDeterministicProse(t *testing.T) {
	p := NewMockTextAI()

	out, err := p.GenerateText(context.Background(), TextAIInput{
		Prompt: "Write chapter 1",
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if out.Text == "" {
		t.Fatal("expected non-empty text")
	}
	if out.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", out.Provider)
	}
	if out.Model == "" {
		t.Fatal("expected non-empty model")
	}
}

func TestMockTextAIProducesStableOutput(t *testing.T) {
	p := NewMockTextAI()

	a, _ := p.GenerateText(context.Background(), TextAIInput{Prompt: "same"})
	b, _ := p.GenerateText(context.Background(), TextAIInput{Prompt: "same"})

	if a.Text != b.Text {
		t.Fatal("expected deterministic output for same prompt")
	}
}

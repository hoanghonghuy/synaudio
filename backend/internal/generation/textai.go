package generation

import (
	"context"
)

// TextAIProvider is the abstraction for text generation providers.
type TextAIProvider interface {
	GenerateText(ctx context.Context, in TextAIInput) (TextAIOutput, error)
}

// TextAIInput is the input for a text generation call.
type TextAIInput struct {
	Prompt string
}

// TextAIOutput is the result of a text generation call.
type TextAIOutput struct {
	Text     string
	Provider string
	Model    string
}

// MockTextAI is a deterministic text provider for development/testing.
type MockTextAI struct{}

func NewMockTextAI() *MockTextAI {
	return &MockTextAI{}
}

// GenerateText returns deterministic prose for development/testing.
func (MockTextAI) GenerateText(_ context.Context, in TextAIInput) (TextAIOutput, error) {
	return TextAIOutput{
		Text:     "The morning light crept over the hills, and the village stirred to life. " + in.Prompt,
		Provider: "mock",
		Model:    "mock-text-v1",
	}, nil
}

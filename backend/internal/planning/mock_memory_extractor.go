package planning

import (
	"context"
)

// MockMemoryExtractor is a deterministic memory extractor for development/testing.
type MockMemoryExtractor struct{}

func NewMockMemoryExtractor() *MockMemoryExtractor {
	return &MockMemoryExtractor{}
}

// ExtractMemory returns a deterministic set of facts for development/testing.
func (MockMemoryExtractor) ExtractMemory(_ context.Context, in MemoryExtractionInput) (MemoryExtraction, error) {
	return MemoryExtraction{
		Facts: []ExtractedFact{
			{
				SubjectType: "CHARACTER",
				SubjectID:   "character-1",
				FactType:    "NAME",
				Value:       map[string]any{"value": "Aria"},
			},
			{
				SubjectType: "WORLD",
				FactType:    "LOCATION",
				Value:       map[string]any{"value": "The village of Elderglen"},
			},
		},
	}, nil
}

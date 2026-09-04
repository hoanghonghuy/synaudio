package story

import (
	"context"
	"errors"
)

type generationPolicyReader interface {
	GetGenerationPolicy(ctx context.Context, storyID string) (GenerationPolicy, error)
}

// GetGenerationPolicy returns the immutable policy snapshot resolved when the
// Story was created. The Story service intentionally exposes no policy update.
func (s *Service) GetGenerationPolicy(ctx context.Context, storyID string) (GenerationPolicy, error) {
	reader, ok := s.store.(generationPolicyReader)
	if !ok {
		return GenerationPolicy{}, errors.New("generation policy reader unavailable")
	}
	return reader.GetGenerationPolicy(ctx, storyID)
}

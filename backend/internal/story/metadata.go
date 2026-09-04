package story

import (
	"context"
	"strings"
)

// UpdateMetadata edits only mutable Story presentation metadata. Lifecycle,
// visibility, immutable policy, planning mode and version pointers are preserved
// and remain governed by their dedicated domain operations.
func (s *Service) UpdateMetadata(ctx context.Context, storyID, title, description string) (Story, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Story{}, ErrInvalidTitle
	}

	st, err := s.store.GetStory(ctx, storyID)
	if err != nil {
		return Story{}, err
	}
	st.Title = title
	st.Description = strings.TrimSpace(description)
	return s.store.UpdateStory(ctx, st)
}

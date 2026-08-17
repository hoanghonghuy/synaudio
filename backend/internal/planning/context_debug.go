package planning

import (
	"context"
	"errors"
)

var (
	ErrContextSnapshotNotFound = errors.New("context snapshot not found")
)

// ThreadInactivity is a report of a plot thread that has not advanced recently.
type ThreadInactivity struct {
	ThreadID   string
	Title      string
	Importance string
	EventCount int
}

// GetContextSnapshot returns a single context snapshot by ID.
func (s *Service) GetContextSnapshot(ctx context.Context, id string) (ContextSnapshot, error) {
	return s.store.GetContextSnapshot(ctx, id)
}

// AnalyzeThreadInactivity reports OPEN plot threads with fewer than
// `minEvents` events, surfacing potential quality drift / thread neglect.
func (s *Service) AnalyzeThreadInactivity(ctx context.Context, storyID string, minEvents int) ([]ThreadInactivity, error) {
	threads, err := s.store.ListPlotThreads(ctx, storyID)
	if err != nil {
		return nil, err
	}

	var out []ThreadInactivity
	for _, t := range threads {
		if t.Status != "OPEN" {
			continue
		}
		events, err := s.store.ListPlotThreadEvents(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		if len(events) < minEvents {
			out = append(out, ThreadInactivity{
				ThreadID:   t.ID,
				Title:      t.Title,
				Importance: t.Importance,
				EventCount: len(events),
			})
		}
	}

	return out, nil
}

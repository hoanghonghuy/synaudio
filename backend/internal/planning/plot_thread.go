package planning

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// PlotThread is a narrative thread tracked across chapters.
type PlotThread struct {
	ID         string
	StoryID    string
	Title      string
	Summary    string
	Importance string
	Status     string
}

// PlotThreadEvent is a discrete event in a plot thread's lifecycle.
type PlotThreadEvent struct {
	ID           string
	PlotThreadID string
	ChapterID    string
	EventType    string
	Detail       map[string]any
}

// CreatePlotThread creates a new OPEN plot thread.
func (s *Service) CreatePlotThread(ctx context.Context, storyID, title, summary, importance string) (PlotThread, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return PlotThread{}, errors.New("plot thread title must not be empty")
	}
	if importance == "" {
		importance = "NORMAL"
	}

	t := PlotThread{
		ID:         uuid.NewString(),
		StoryID:    storyID,
		Title:      title,
		Summary:    summary,
		Importance: importance,
		Status:     "OPEN",
	}

	return s.store.CreatePlotThread(ctx, t)
}

// ListPlotThreads returns all plot threads for a story.
func (s *Service) ListPlotThreads(ctx context.Context, storyID string) ([]PlotThread, error) {
	return s.store.ListPlotThreads(ctx, storyID)
}

// CreatePlotThreadEvent records an event on a plot thread.
func (s *Service) CreatePlotThreadEvent(ctx context.Context, threadID, eventType, chapterID string, detail map[string]any) (PlotThreadEvent, error) {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return PlotThreadEvent{}, errors.New("event type must not be empty")
	}

	e := PlotThreadEvent{
		ID:           uuid.NewString(),
		PlotThreadID: threadID,
		ChapterID:    chapterID,
		EventType:    eventType,
		Detail:       detail,
	}

	return s.store.CreatePlotThreadEvent(ctx, e)
}

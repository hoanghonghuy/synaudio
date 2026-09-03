package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/audit"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

type AuditStore struct {
	q *db.Queries
}

func NewAuditStore(q *db.Queries) *AuditStore {
	return &AuditStore{q: q}
}

func (s *AuditStore) CreateAuditEvent(ctx context.Context, event audit.Event) (audit.Event, error) {
	provenance, err := json.Marshal(event.Provenance)
	if err != nil {
		return audit.Event{}, err
	}
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return audit.Event{}, err
	}
	row, err := s.q.CreateAuditEvent(ctx, db.CreateAuditEventParams{
		ID:              toUUID(event.ID),
		ActorUserID:     toUUID(event.ActorUserID),
		ActorType:       event.ActorType,
		Action:          event.Action,
		ResourceType:    toText(event.ResourceType),
		ResourceID:      toText(event.ResourceID),
		StoryID:         toUUID(event.StoryID),
		ChapterID:       toUUID(event.ChapterID),
		Result:          event.Result,
		CorrelationID:   toText(event.CorrelationID),
		RequestID:       toText(event.RequestID),
		GenerationRunID: toUUID(event.GenerationRunID),
		Provenance:      provenance,
		Metadata:        metadata,
	})
	if err != nil {
		return audit.Event{}, err
	}
	return toAuditEvent(row), nil
}

func (s *AuditStore) GetAuditEvent(ctx context.Context, id string) (audit.Event, error) {
	row, err := s.q.GetAuditEvent(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audit.Event{}, audit.ErrNotFound
		}
		return audit.Event{}, err
	}
	return toAuditEvent(row), nil
}

func (s *AuditStore) ListAuditEvents(ctx context.Context, filter audit.Filter) ([]audit.Event, error) {
	rows, err := s.q.ListAuditEvents(ctx, db.ListAuditEventsParams{
		ActorUserID:     toUUID(filter.ActorUserID),
		Action:          toText(filter.Action),
		ResourceType:    toText(filter.ResourceType),
		ResourceID:      toText(filter.ResourceID),
		StoryID:         toUUID(filter.StoryID),
		ChapterID:       toUUID(filter.ChapterID),
		GenerationRunID: toUUID(filter.GenerationRunID),
		CorrelationID:   toText(filter.CorrelationID),
		Result:          toText(filter.Result),
		CreatedFrom:     pgtypeTimestamptz(filter.From),
		CreatedTo:       pgtypeTimestamptz(filter.To),
		Limit:           int32(filter.Limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]audit.Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAuditEvent(row))
	}
	return out, nil
}

func toAuditEvent(row db.AuditEvent) audit.Event {
	var provenance map[string]any
	var metadata map[string]any
	_ = json.Unmarshal(row.Provenance, &provenance)
	_ = json.Unmarshal(row.Metadata, &metadata)
	return audit.Event{
		ID:              fromUUID(row.ID),
		ActorUserID:     fromUUID(row.ActorUserID),
		ActorType:       row.ActorType,
		Action:          row.Action,
		ResourceType:    fromText(row.ResourceType),
		ResourceID:      fromText(row.ResourceID),
		StoryID:         fromUUID(row.StoryID),
		ChapterID:       fromUUID(row.ChapterID),
		Result:          row.Result,
		CorrelationID:   fromText(row.CorrelationID),
		RequestID:       fromText(row.RequestID),
		GenerationRunID: fromUUID(row.GenerationRunID),
		Provenance:      provenance,
		Metadata:        metadata,
		CreatedAt:       row.CreatedAt.Time,
	}
}

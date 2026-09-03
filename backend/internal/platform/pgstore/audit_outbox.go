package pgstore

import (
	"context"
	"encoding/json"
	"time"

	"github.com/synaudio/synaudio/backend/internal/audit"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func (s *AuditStore) EnqueueAuditIntent(ctx context.Context, event audit.Event) error {
	encoded, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return s.q.EnqueueAuditIntent(ctx, db.EnqueueAuditIntentParams{
		ID:    toUUID(event.ID),
		Event: encoded,
	})
}

func (s *AuditStore) ClaimAuditIntents(ctx context.Context, limit int) ([]audit.DeliveryIntent, error) {
	rows, err := s.q.ClaimAuditIntents(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]audit.DeliveryIntent, 0, len(rows))
	for _, row := range rows {
		var event audit.Event
		if err := json.Unmarshal(row.Event, &event); err != nil {
			return nil, err
		}
		out = append(out, audit.DeliveryIntent{
			ID:           fromUUID(row.ID),
			Event:        event,
			Status:       row.Status,
			AttemptCount: int(row.AttemptCount),
			MaxAttempts:  int(row.MaxAttempts),
		})
	}
	return out, nil
}

func (s *AuditStore) MarkAuditIntentDelivered(ctx context.Context, id string) error {
	return s.q.MarkAuditIntentDelivered(ctx, toUUID(id))
}

func (s *AuditStore) MarkAuditIntentFailed(ctx context.Context, id, lastError string, retryAt time.Time, deadLetter bool) (audit.DeliveryIntent, error) {
	status := audit.DeliveryPending
	if deadLetter {
		status = audit.DeliveryDeadLetter
	}
	row, err := s.q.MarkAuditIntentFailed(ctx, db.MarkAuditIntentFailedParams{
		ID:          toUUID(id),
		Status:      status,
		AvailableAt: pgtypeTimestamptz(retryAt),
		LastError:   toText(lastError),
	})
	if err != nil {
		return audit.DeliveryIntent{}, err
	}
	var event audit.Event
	if err := json.Unmarshal(row.Event, &event); err != nil {
		return audit.DeliveryIntent{}, err
	}
	return audit.DeliveryIntent{
		ID:           fromUUID(row.ID),
		Event:        event,
		Status:       row.Status,
		AttemptCount: int(row.AttemptCount),
		MaxAttempts:  int(row.MaxAttempts),
	}, nil
}

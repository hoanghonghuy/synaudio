package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	DeliveryPending    = "PENDING"
	DeliveryDelivering = "DELIVERING"
	DeliveryDelivered  = "DELIVERED"
	DeliveryDeadLetter = "DEAD_LETTER"
)

type DeliveryIntent struct {
	ID           string
	Event        Event
	Status       string
	AttemptCount int
	MaxAttempts  int
}

type DeliveryReport struct {
	Claimed    int
	Delivered  int
	Retrying   int
	DeadLetter int
}

// OutboxStore is an optional reliability extension for audit stores. The final
// append-only audit event remains the source of truth; the outbox only carries
// sanitized delivery intent until that event is durably present.
type OutboxStore interface {
	EnqueueAuditIntent(ctx context.Context, event Event) error
	ClaimAuditIntents(ctx context.Context, limit int) ([]DeliveryIntent, error)
	MarkAuditIntentDelivered(ctx context.Context, id string) error
	MarkAuditIntentFailed(ctx context.Context, id, lastError string, retryAt time.Time, deadLetter bool) (DeliveryIntent, error)
}

// RecordReliable first attempts the normal append-only write. If that immediate
// write fails and durable outbox support is configured, the exact sanitized
// event (including its stable ID) is queued for later reconciliation. A queued
// event is accepted as durable delivery intent, so callers keep the true
// business response instead of fabricating a retriable failure after commit.
func (s *Service) RecordReliable(ctx context.Context, event Event) (Event, error) {
	prepared, err := s.prepareEvent(event)
	if err != nil {
		return Event{}, err
	}
	created, err := s.store.CreateAuditEvent(ctx, prepared)
	if err == nil {
		return created, nil
	}

	outbox, ok := s.store.(OutboxStore)
	if !ok {
		return Event{}, err
	}
	if enqueueErr := outbox.EnqueueAuditIntent(ctx, prepared); enqueueErr != nil {
		return Event{}, fmt.Errorf("audit append failed: %v; durable intent enqueue failed: %w", err, enqueueErr)
	}
	return prepared, nil
}

// DeliverPending reconciles durable audit intents. Delivery is idempotent by
// stable Event.ID: if a prior attempt appended the final event but crashed before
// acknowledging the outbox row, the existing append-only event is accepted and
// the intent is marked delivered rather than duplicated.
func (s *Service) DeliverPending(ctx context.Context, limit int) (DeliveryReport, error) {
	if s == nil || s.store == nil {
		return DeliveryReport{}, errors.New("audit store not configured")
	}
	outbox, ok := s.store.(OutboxStore)
	if !ok {
		return DeliveryReport{}, errors.New("audit outbox not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	intents, err := outbox.ClaimAuditIntents(ctx, limit)
	if err != nil {
		return DeliveryReport{}, err
	}
	report := DeliveryReport{Claimed: len(intents)}
	for _, intent := range intents {
		_, appendErr := s.store.CreateAuditEvent(ctx, intent.Event)
		if appendErr != nil {
			// The final event may already exist because a previous delivery appended
			// it and failed before marking the outbox row delivered.
			if existing, lookupErr := s.store.GetAuditEvent(ctx, intent.Event.ID); lookupErr == nil && existing.ID == intent.Event.ID {
				if err := outbox.MarkAuditIntentDelivered(ctx, intent.ID); err != nil {
					return report, err
				}
				report.Delivered++
				continue
			}

			deadLetter := intent.AttemptCount >= intent.MaxAttempts
			retryAt := s.now().UTC().Add(auditRetryDelay(intent.AttemptCount))
			if _, err := outbox.MarkAuditIntentFailed(ctx, intent.ID, boundedAuditError(appendErr), retryAt, deadLetter); err != nil {
				return report, err
			}
			if deadLetter {
				report.DeadLetter++
			} else {
				report.Retrying++
			}
			continue
		}

		if err := outbox.MarkAuditIntentDelivered(ctx, intent.ID); err != nil {
			return report, err
		}
		report.Delivered++
	}
	return report, nil
}

func (s *Service) prepareEvent(event Event) (Event, error) {
	if s == nil || s.store == nil {
		return Event{}, errors.New("audit store not configured")
	}
	event.Action = strings.TrimSpace(event.Action)
	event.ActorType = strings.ToUpper(strings.TrimSpace(event.ActorType))
	event.Result = strings.ToUpper(strings.TrimSpace(event.Result))
	if event.Action == "" {
		return Event{}, errors.New("audit action is required")
	}
	if !validActorType(event.ActorType) {
		return Event{}, fmt.Errorf("invalid audit actor type %q", event.ActorType)
	}
	if !validResult(event.Result) {
		return Event{}, fmt.Errorf("invalid audit result %q", event.Result)
	}
	if event.ActorType == ActorUser && strings.TrimSpace(event.ActorUserID) == "" {
		return Event{}, errors.New("USER audit actor requires actor_user_id")
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	event.Provenance = SanitizeMetadata(event.Provenance)
	event.Metadata = SanitizeMetadata(event.Metadata)
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.now().UTC()
	}
	return event, nil
}

func auditRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	shift := attempt - 1
	if shift > 6 {
		shift = 6
	}
	return time.Minute * time.Duration(1<<shift)
}

func boundedAuditError(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimSpace(err.Error())
	const max = 1000
	if len(value) > max {
		value = value[:max]
	}
	return value
}

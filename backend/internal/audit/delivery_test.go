package audit

import (
	"context"
	"errors"
	"testing"
	"time"
)

type reliableStore struct {
	finalEvents      map[string]Event
	intents          map[string]DeliveryIntent
	appendFailures   int
	markFailures     int
	defaultMax       int
}

func newReliableStore() *reliableStore {
	return &reliableStore{
		finalEvents: map[string]Event{},
		intents:     map[string]DeliveryIntent{},
		defaultMax:  3,
	}
}

func (s *reliableStore) CreateAuditEvent(_ context.Context, event Event) (Event, error) {
	if s.appendFailures > 0 {
		s.appendFailures--
		return Event{}, errors.New("injected audit append failure")
	}
	if _, exists := s.finalEvents[event.ID]; exists {
		return Event{}, errors.New("duplicate audit event")
	}
	s.finalEvents[event.ID] = event
	return event, nil
}

func (s *reliableStore) GetAuditEvent(_ context.Context, id string) (Event, error) {
	event, ok := s.finalEvents[id]
	if !ok {
		return Event{}, ErrNotFound
	}
	return event, nil
}

func (s *reliableStore) ListAuditEvents(_ context.Context, _ Filter) ([]Event, error) {
	out := make([]Event, 0, len(s.finalEvents))
	for _, event := range s.finalEvents {
		out = append(out, event)
	}
	return out, nil
}

func (s *reliableStore) EnqueueAuditIntent(_ context.Context, event Event) error {
	if _, exists := s.intents[event.ID]; exists {
		return nil
	}
	s.intents[event.ID] = DeliveryIntent{
		ID:          event.ID,
		Event:       event,
		Status:      DeliveryPending,
		MaxAttempts: s.defaultMax,
	}
	return nil
}

func (s *reliableStore) ClaimAuditIntents(_ context.Context, limit int) ([]DeliveryIntent, error) {
	out := []DeliveryIntent{}
	for id, intent := range s.intents {
		if intent.Status == DeliveryDelivered || intent.Status == DeliveryDeadLetter {
			continue
		}
		intent.Status = DeliveryDelivering
		intent.AttemptCount++
		s.intents[id] = intent
		out = append(out, intent)
		if len(out) == limit {
			break
		}
	}
	return out, nil
}

func (s *reliableStore) MarkAuditIntentDelivered(_ context.Context, id string) error {
	if s.markFailures > 0 {
		s.markFailures--
		return errors.New("injected acknowledgement failure")
	}
	intent := s.intents[id]
	intent.Status = DeliveryDelivered
	s.intents[id] = intent
	return nil
}

func (s *reliableStore) MarkAuditIntentFailed(_ context.Context, id, _ string, _ time.Time, deadLetter bool) (DeliveryIntent, error) {
	intent := s.intents[id]
	if deadLetter {
		intent.Status = DeliveryDeadLetter
	} else {
		intent.Status = DeliveryPending
	}
	s.intents[id] = intent
	return intent, nil
}

func TestRecordReliableQueuesThenRetriesCommittedAudit(t *testing.T) {
	store := newReliableStore()
	store.appendFailures = 1
	svc := NewService(store)

	recorded, err := svc.RecordReliable(context.Background(), Event{
		ActorType: ActorUser,
		ActorUserID: "user-1",
		Action: "STORY_ARCHIVED",
		Result: ResultSucceeded,
		Metadata: map[string]any{"password": "must-never-survive", "route": "/admin/stories/{storyID}/archive"},
	})
	if err != nil {
		t.Fatalf("reliable record: %v", err)
	}
	if recorded.ID == "" {
		t.Fatal("expected stable event id before outbox enqueue")
	}
	if len(store.finalEvents) != 0 {
		t.Fatal("expected immediate append to fail")
	}
	intent := store.intents[recorded.ID]
	if got := intent.Event.Metadata["password"]; got != "[REDACTED]" {
		t.Fatalf("queued intent retained secret: %#v", got)
	}

	report, err := svc.DeliverPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("deliver pending: %v", err)
	}
	if report.Delivered != 1 || len(store.finalEvents) != 1 {
		t.Fatalf("expected one delivered final event, report=%+v final=%d", report, len(store.finalEvents))
	}
	if store.intents[recorded.ID].Status != DeliveryDelivered {
		t.Fatalf("expected delivered intent, got %q", store.intents[recorded.ID].Status)
	}
}

func TestDeliverPendingIsIdempotentAfterAppendBeforeAcknowledgement(t *testing.T) {
	store := newReliableStore()
	store.appendFailures = 1
	store.markFailures = 1
	svc := NewService(store)

	recorded, err := svc.RecordReliable(context.Background(), Event{
		ActorType: ActorSystem,
		Action: "GENERATION_JOB_SUCCEEDED",
		Result: ResultSucceeded,
	})
	if err != nil {
		t.Fatalf("reliable record: %v", err)
	}

	if _, err := svc.DeliverPending(context.Background(), 10); err == nil {
		t.Fatal("expected injected acknowledgement failure")
	}
	if len(store.finalEvents) != 1 {
		t.Fatalf("expected append before ack failure, got %d final events", len(store.finalEvents))
	}

	report, err := svc.DeliverPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("second reconciliation: %v", err)
	}
	if report.Delivered != 1 {
		t.Fatalf("expected duplicate-safe acknowledgement, report=%+v", report)
	}
	if len(store.finalEvents) != 1 {
		t.Fatalf("idempotent retry created duplicate final events: %d", len(store.finalEvents))
	}
	if store.intents[recorded.ID].Status != DeliveryDelivered {
		t.Fatalf("expected delivered intent, got %q", store.intents[recorded.ID].Status)
	}
}

func TestDeliverPendingDeadLettersRepeatedFailures(t *testing.T) {
	store := newReliableStore()
	store.defaultMax = 2
	store.appendFailures = 3
	svc := NewService(store)

	recorded, err := svc.RecordReliable(context.Background(), Event{
		ActorType: ActorSystem,
		Action: "GENERATION_JOB_FAILED",
		Result: ResultFailed,
	})
	if err != nil {
		t.Fatalf("reliable record: %v", err)
	}

	first, err := svc.DeliverPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("first retry: %v", err)
	}
	if first.Retrying != 1 {
		t.Fatalf("expected one retrying intent, report=%+v", first)
	}
	second, err := svc.DeliverPending(context.Background(), 10)
	if err != nil {
		t.Fatalf("second retry: %v", err)
	}
	if second.DeadLetter != 1 {
		t.Fatalf("expected dead letter on max attempts, report=%+v", second)
	}
	if store.intents[recorded.ID].Status != DeliveryDeadLetter {
		t.Fatalf("expected DEAD_LETTER, got %q", store.intents[recorded.ID].Status)
	}
}

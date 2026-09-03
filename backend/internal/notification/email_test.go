package notification

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeOutbox struct {
	items            []OutboxItem
	delivered        []string
	dispatchStarted  []string
	uncertain        []string
	failures         []string
	claimError       error
	markDeliveredErr error
}

func (f *fakeOutbox) Enqueue(_ context.Context, item OutboxItem) error {
	f.items = append(f.items, item)
	return nil
}

func (f *fakeOutbox) Claim(_ context.Context, _, _ time.Time) (OutboxItem, error) {
	if f.claimError != nil {
		return OutboxItem{}, f.claimError
	}
	if len(f.items) == 0 {
		return OutboxItem{}, ErrNoPendingDelivery
	}
	item := f.items[0]
	f.items = f.items[1:]
	item.AttemptCount++
	return item, nil
}

func (f *fakeOutbox) MarkDispatchStarted(_ context.Context, id string, _ time.Time) error {
	f.dispatchStarted = append(f.dispatchStarted, id)
	return nil
}

func (f *fakeOutbox) MarkDelivered(_ context.Context, id string, _ time.Time) error {
	if f.markDeliveredErr != nil {
		return f.markDeliveredErr
	}
	f.delivered = append(f.delivered, id)
	return nil
}

func (f *fakeOutbox) MarkFailed(_ context.Context, id string, _, _ time.Time, failure string) error {
	f.failures = append(f.failures, id+":"+failure)
	return nil
}

func (f *fakeOutbox) MarkDeliveryUncertain(_ context.Context, id string, _ time.Time) error {
	f.uncertain = append(f.uncertain, id)
	return nil
}

type fakeSender struct {
	messages []Message
	err      error
}

func (f *fakeSender) Send(_ context.Context, message Message) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, message)
	return nil
}

func TestVerificationTokenIsEncryptedAtRestAndDelivered(t *testing.T) {
	store := &fakeOutbox{}
	sender := &fakeSender{}
	svc, err := NewService(store, sender, "test-email-payload-secret-that-is-long-enough", "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}

	rawToken := "raw-verification-token-must-not-be-persisted"
	if err := svc.QueueVerification(context.Background(), "reader@example.com", rawToken); err != nil {
		t.Fatal(err)
	}
	if len(store.items) != 1 {
		t.Fatalf("expected one durable intent, got %d", len(store.items))
	}
	item := store.items[0]
	if bytes.Contains(item.EncryptedPayload, []byte(rawToken)) {
		t.Fatal("raw token leaked into durable encrypted payload")
	}

	didWork, err := svc.DeliverNext(context.Background())
	if err != nil || !didWork {
		t.Fatalf("deliver next: didWork=%v err=%v", didWork, err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected one delivered message, got %d", len(sender.messages))
	}
	if !strings.Contains(sender.messages[0].Text, rawToken) {
		t.Fatal("delivered verification link does not contain original one-time token")
	}
	if sender.messages[0].ID != "<synaudio-"+item.ID+"@transactional.local>" {
		t.Fatalf("message id = %q, want stable outbox-derived id", sender.messages[0].ID)
	}
	if len(store.dispatchStarted) != 1 || store.dispatchStarted[0] != item.ID {
		t.Fatal("delivery dispatch boundary was not durably marked before send")
	}
	if len(store.delivered) != 1 || store.delivered[0] != item.ID {
		t.Fatal("delivery was not durably acknowledged")
	}
}

func TestPasswordResetDeliveryFailureIsRecordedWithoutLeakingLink(t *testing.T) {
	store := &fakeOutbox{}
	sender := &fakeSender{err: errors.New("temporary upstream SMTP outage")}
	svc, err := NewService(store, sender, "test-email-payload-secret-that-is-long-enough", "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.QueuePasswordReset(context.Background(), "reader@example.com", "reset-token-secret"); err != nil {
		t.Fatal(err)
	}

	didWork, err := svc.DeliverNext(context.Background())
	if !didWork || err == nil {
		t.Fatalf("expected attempted failed delivery, didWork=%v err=%v", didWork, err)
	}
	if len(store.failures) != 1 {
		t.Fatalf("expected one durable failure update, got %d", len(store.failures))
	}
	if strings.Contains(store.failures[0], "reset-token-secret") || strings.Contains(store.failures[0], "https://app.example.com") {
		t.Fatal("retry/dead-letter metadata leaked a transactional link or token")
	}
}

func TestAcceptedSendWithAcknowledgementFailureIsNotBlindlyResent(t *testing.T) {
	store := &fakeOutbox{markDeliveredErr: errors.New("database unavailable after SMTP acceptance")}
	sender := &fakeSender{}
	svc, err := NewService(store, sender, "test-email-payload-secret-that-is-long-enough", "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.QueueVerification(context.Background(), "reader@example.com", "single-use-token"); err != nil {
		t.Fatal(err)
	}
	original := store.items[0]

	didWork, err := svc.DeliverNext(context.Background())
	if !didWork || err == nil {
		t.Fatalf("expected durable acknowledgement failure after send, didWork=%v err=%v", didWork, err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("first attempt should send exactly once, got %d", len(sender.messages))
	}
	if len(store.dispatchStarted) != 1 {
		t.Fatal("dispatch marker must survive acknowledgement failure")
	}

	// Simulate stale reclaim of the same row. PostgreSQL exposes the persisted
	// dispatch marker as DispatchStarted=true. The service must quarantine it
	// instead of sending the same accepted message again.
	original.DispatchStarted = true
	store.items = append(store.items, original)
	store.markDeliveredErr = nil
	didWork, err = svc.DeliverNext(context.Background())
	if !didWork || !errors.Is(err, ErrDeliveryOutcomeUncertain) {
		t.Fatalf("stale dispatched item should become uncertain, didWork=%v err=%v", didWork, err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("accepted message was blindly re-sent; sends=%d", len(sender.messages))
	}
	if len(store.uncertain) != 1 || store.uncertain[0] != original.ID {
		t.Fatal("stale dispatched item was not quarantined for reconciliation")
	}
}

func TestAmbiguousSMTPFailureIsQuarantinedNotRetried(t *testing.T) {
	store := &fakeOutbox{}
	sender := &fakeSender{err: fmt.Errorf("%w: connection dropped after DATA", ErrDeliveryOutcomeUncertain)}
	svc, err := NewService(store, sender, "test-email-payload-secret-that-is-long-enough", "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.QueueVerification(context.Background(), "reader@example.com", "token"); err != nil {
		t.Fatal(err)
	}
	didWork, err := svc.DeliverNext(context.Background())
	if !didWork || !errors.Is(err, ErrDeliveryOutcomeUncertain) {
		t.Fatalf("expected uncertain SMTP outcome, didWork=%v err=%v", didWork, err)
	}
	if len(store.failures) != 0 {
		t.Fatal("ambiguous SMTP outcome must not enter automatic retry")
	}
	if len(store.uncertain) != 1 {
		t.Fatal("ambiguous SMTP outcome must be quarantined")
	}
}

func TestNoPendingDeliveryIsIdleNotFailure(t *testing.T) {
	svc, err := NewService(&fakeOutbox{}, &fakeSender{}, "test-email-payload-secret-that-is-long-enough", "https://app.example.com")
	if err != nil {
		t.Fatal(err)
	}
	didWork, err := svc.DeliverNext(context.Background())
	if err != nil || didWork {
		t.Fatalf("empty outbox should be idle: didWork=%v err=%v", didWork, err)
	}
}

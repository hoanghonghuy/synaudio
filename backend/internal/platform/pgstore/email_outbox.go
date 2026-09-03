package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/notification"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const smtpDispatchStartedMarker = "SMTP_DISPATCH_STARTED"

type EmailOutboxStore struct {
	db db.DBTX
}

func NewEmailOutboxStore(executor db.DBTX) *EmailOutboxStore {
	return &EmailOutboxStore{db: executor}
}

func (s *EmailOutboxStore) Enqueue(ctx context.Context, item notification.OutboxItem) error {
	// Auth delivery runs inside a request transaction. The advisory transaction
	// lock serializes concurrent resend/reset requests for the same address and
	// purpose so the cooldown/hourly cap cannot be bypassed by a request race.
	if _, err := s.db.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, item.Purpose+":"+item.RecipientEmail); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `
INSERT INTO email_delivery_outbox (
    id, purpose, recipient_email, encrypted_payload, status, attempt_count,
    max_attempts, available_at, created_at, updated_at
)
SELECT $1, $2, $3, $4, 'PENDING', 0, $5, NOW(), NOW(), NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM email_delivery_outbox
    WHERE purpose = $2
      AND lower(recipient_email) = lower($3)
      AND created_at > NOW() - INTERVAL '60 seconds'
)
AND (
    SELECT COUNT(*) FROM email_delivery_outbox
    WHERE purpose = $2
      AND lower(recipient_email) = lower($3)
      AND created_at > NOW() - INTERVAL '1 hour'
) < 5`, item.ID, item.Purpose, item.RecipientEmail, item.EncryptedPayload, item.MaxAttempts)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return notification.ErrRateLimited
	}
	return nil
}

func (s *EmailOutboxStore) Claim(ctx context.Context, now, staleBefore time.Time) (notification.OutboxItem, error) {
	row := s.db.QueryRow(ctx, `
UPDATE email_delivery_outbox
SET status = 'DELIVERING',
    locked_at = $1,
    attempt_count = attempt_count + 1,
    updated_at = $1
WHERE id = (
    SELECT id
    FROM email_delivery_outbox
    WHERE (status = 'PENDING' AND available_at <= $1)
       OR (status = 'DELIVERING' AND locked_at <= $2)
    ORDER BY available_at, created_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING id::text, purpose, recipient_email, encrypted_payload, attempt_count, max_attempts,
          (last_error = '` + smtpDispatchStartedMarker + `') AS dispatch_started`, now, staleBefore)

	var item notification.OutboxItem
	if err := row.Scan(&item.ID, &item.Purpose, &item.RecipientEmail, &item.EncryptedPayload, &item.AttemptCount, &item.MaxAttempts, &item.DispatchStarted); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notification.OutboxItem{}, notification.ErrNoPendingDelivery
		}
		return notification.OutboxItem{}, err
	}
	return item, nil
}

func (s *EmailOutboxStore) MarkDispatchStarted(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.Exec(ctx, `
UPDATE email_delivery_outbox
SET last_error = '`+smtpDispatchStartedMarker+`', updated_at = $2
WHERE id = $1 AND status = 'DELIVERING'`, id, now)
	return err
}

func (s *EmailOutboxStore) MarkDelivered(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.Exec(ctx, `
UPDATE email_delivery_outbox
SET status = 'DELIVERED', locked_at = NULL, last_error = NULL, updated_at = $2
WHERE id = $1 AND status = 'DELIVERING'`, id, now)
	return err
}

func (s *EmailOutboxStore) MarkFailed(ctx context.Context, id string, now, retryAt time.Time, failure string) error {
	_, err := s.db.Exec(ctx, `
UPDATE email_delivery_outbox
SET status = CASE WHEN attempt_count >= max_attempts THEN 'DEAD_LETTER' ELSE 'PENDING' END,
    available_at = CASE WHEN attempt_count >= max_attempts THEN available_at ELSE $3 END,
    locked_at = NULL,
    last_error = $4,
    updated_at = $2
WHERE id = $1 AND status = 'DELIVERING'`, id, now, retryAt, failure)
	return err
}

func (s *EmailOutboxStore) MarkDeliveryUncertain(ctx context.Context, id string, now time.Time) error {
	_, err := s.db.Exec(ctx, `
UPDATE email_delivery_outbox
SET status = 'DEAD_LETTER',
    locked_at = NULL,
    last_error = 'SMTP_DELIVERY_OUTCOME_UNCERTAIN',
    updated_at = $2
WHERE id = $1 AND status = 'DELIVERING'`, id, now)
	return err
}

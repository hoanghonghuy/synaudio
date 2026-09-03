package pgstore

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/notification"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

type EmailOutboxStore struct {
	db db.DBTX
}

func NewEmailOutboxStore(executor db.DBTX) *EmailOutboxStore {
	return &EmailOutboxStore{db: executor}
}

func (s *EmailOutboxStore) Enqueue(ctx context.Context, item notification.OutboxItem) error {
	_, err := s.db.Exec(ctx, `
INSERT INTO email_delivery_outbox (
    id, purpose, recipient_email, encrypted_payload, status, attempt_count,
    max_attempts, available_at, created_at, updated_at
) VALUES ($1, $2, $3, $4, 'PENDING', 0, $5, NOW(), NOW(), NOW())`,
		item.ID, item.Purpose, item.RecipientEmail, item.EncryptedPayload, item.MaxAttempts)
	return err
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
RETURNING id::text, purpose, recipient_email, encrypted_payload, attempt_count, max_attempts`, now, staleBefore)

	var item notification.OutboxItem
	if err := row.Scan(&item.ID, &item.Purpose, &item.RecipientEmail, &item.EncryptedPayload, &item.AttemptCount, &item.MaxAttempts); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return notification.OutboxItem{}, notification.ErrNoPendingDelivery
		}
		return notification.OutboxItem{}, err
	}
	return item, nil
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

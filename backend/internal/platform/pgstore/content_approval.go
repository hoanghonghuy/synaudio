package pgstore

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const (
	lockContentRevisionForApprovalSQL = `
SELECT chapter_id, status
FROM chapter_content_revisions
WHERE id = $1
FOR UPDATE`

	getExistingContentApprovalSQL = `
SELECT id, chapter_id, content_revision_id, approved_by, approved_at,
       warnings_snapshot, override_snapshot
FROM content_approvals
WHERE content_revision_id = $1
ORDER BY approved_at DESC, id DESC
LIMIT 1`
)

// ApproveContentRevision serializes approval on the exact revision row and
// commits the append-only approval record and APPROVED projection together.
// A retry returns the existing approval instead of creating another side effect.
func (s *GenerationStore) ApproveContentRevision(ctx context.Context, a generation.ContentApproval) (generation.ContentApproval, error) {
	beginner, ok := s.q.DBTX().(generationTransactionBeginner)
	if !ok {
		return generation.ContentApproval{}, errors.New("generation store transaction support unavailable")
	}

	tx, err := beginner.Begin(ctx)
	if err != nil {
		return generation.ContentApproval{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	revisionID := toUUID(a.ContentRevisionID)
	var lockedChapterID pgtype.UUID
	var status string
	if err := tx.QueryRow(ctx, lockContentRevisionForApprovalSQL, revisionID).Scan(&lockedChapterID, &status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ContentApproval{}, generation.ErrContentRevisionNotFound
		}
		return generation.ContentApproval{}, err
	}
	if fromUUID(lockedChapterID) != a.ChapterID {
		return generation.ContentApproval{}, generation.ErrContentRevisionChapterMismatch
	}
	if status != "CANDIDATE" && status != "APPROVED" {
		return generation.ContentApproval{}, generation.ErrContentRevisionNotApprovable
	}

	existing, err := getExistingContentApproval(ctx, tx, revisionID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return generation.ContentApproval{}, err
	}

	qtx := s.q.WithTx(tx)
	if err == nil {
		if status != "APPROVED" {
			if _, err := qtx.UpdateContentRevisionStatus(ctx, db.UpdateContentRevisionStatusParams{ID: revisionID, Status: "APPROVED"}); err != nil {
				return generation.ContentApproval{}, err
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return generation.ContentApproval{}, err
		}
		return toContentApproval(existing), nil
	}

	warnings, err := json.Marshal(a.WarningsSnapshot)
	if err != nil {
		return generation.ContentApproval{}, err
	}
	override, err := json.Marshal(a.OverrideSnapshot)
	if err != nil {
		return generation.ContentApproval{}, err
	}
	created, err := qtx.CreateContentApproval(ctx, db.CreateContentApprovalParams{
		ID:                toUUID(a.ID),
		ChapterID:         toUUID(a.ChapterID),
		ContentRevisionID: revisionID,
		ApprovedBy:        toUUID(a.ApprovedBy),
		WarningsSnapshot:  warnings,
		OverrideSnapshot:  override,
	})
	if err != nil {
		return generation.ContentApproval{}, err
	}
	if _, err := qtx.UpdateContentRevisionStatus(ctx, db.UpdateContentRevisionStatusParams{ID: revisionID, Status: "APPROVED"}); err != nil {
		return generation.ContentApproval{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return generation.ContentApproval{}, err
	}
	return toContentApproval(created), nil
}

func getExistingContentApproval(ctx context.Context, tx pgx.Tx, revisionID pgtype.UUID) (db.ContentApproval, error) {
	row := tx.QueryRow(ctx, getExistingContentApprovalSQL, revisionID)
	var approval db.ContentApproval
	err := row.Scan(
		&approval.ID,
		&approval.ChapterID,
		&approval.ContentRevisionID,
		&approval.ApprovedBy,
		&approval.ApprovedAt,
		&approval.WarningsSnapshot,
		&approval.OverrideSnapshot,
	)
	return approval, err
}

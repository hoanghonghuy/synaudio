package pgstore

import (
	"context"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

func TestApproveContentRevisionRepairsLegacyApprovalWithoutDuplicate(t *testing.T) {
	pool := newContentApprovalTestPool(t)
	insertContentApprovalRevision(t, pool, approvalRevisionID, approvalChapterID, "CANDIDATE")
	if _, err := pool.Exec(context.Background(), `
INSERT INTO content_approvals (
    id, chapter_id, content_revision_id, approved_by, warnings_snapshot, override_snapshot
) VALUES ($1, $2, $3, $4, '{}'::jsonb, '{}'::jsonb)`,
		"77777777-7777-7777-7777-777777777777", approvalChapterID, approvalRevisionID, approvalActorID); err != nil {
		t.Fatalf("insert legacy approval: %v", err)
	}

	store := NewGenerationStore(db.New(pool))
	approval, err := store.ApproveContentRevision(context.Background(), generation.ContentApproval{
		ID:                "88888888-8888-8888-8888-888888888888",
		ChapterID:         approvalChapterID,
		ContentRevisionID: approvalRevisionID,
		ApprovedBy:        approvalActorID,
	})
	if err != nil {
		t.Fatalf("repair legacy approval: %v", err)
	}
	if approval.ID != "77777777-7777-7777-7777-777777777777" {
		t.Fatalf("expected existing approval authority, got %q", approval.ID)
	}
	assertContentApprovalState(t, pool, approvalRevisionID, "APPROVED", 1)
}

package pgstore

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

const (
	activationChapterID      = "11111111-1111-1111-1111-111111111111"
	activationNarrationOneID = "22222222-2222-2222-2222-222222222222"
	activationNarrationTwoID = "33333333-3333-3333-3333-333333333333"
	activationAssetID = "44444444-4444-4444-4444-444444444444"
)

func TestSetActiveAudioAssetForLatestNarrationRejectsWhenNewerNarrationExists(t *testing.T) {
	pool := newAudioActivationTestPool(t)
	ctx := context.Background()
	insertAudioActivationFixture(t, pool, activationChapterID, activationNarrationOneID, activationAssetID)
	if _, err := pool.Exec(ctx, `
INSERT INTO narration_revisions (id, chapter_id, revision_no, source_content_revision_id, voice_id, script, status, created_by)
VALUES ($1, $2, 2, $3, 'voice-1', 'newer', 'DRAFT', $4)`,
		activationNarrationTwoID, activationChapterID, "77777777-7777-7777-7777-777777777777", "66666666-6666-6666-6666-666666666666"); err != nil {
		t.Fatalf("insert newer narration: %v", err)
	}

	store := NewAudioStore(db.New(pool), pool)
	_, err := store.SetActiveAudioAssetForLatestNarration(ctx, activationChapterID, activationAssetID)
	if !errors.Is(err, audio.ErrAudioAssetStaleForNarration) {
		t.Fatalf("expected stale activation rejection, got %v", err)
	}
	if _, err := store.GetActiveAudioAsset(ctx, activationChapterID); !errors.Is(err, audio.ErrAudioAssetNotFound) {
		t.Fatalf("expected no active asset after stale activation, got %v", err)
	}
}

func TestSetActiveAudioAssetForLatestNarrationActivatesMatchingLatestReadyAsset(t *testing.T) {
	pool := newAudioActivationTestPool(t)
	ctx := context.Background()
	insertAudioActivationFixture(t, pool, activationChapterID, activationNarrationOneID, activationAssetID)
	store := NewAudioStore(db.New(pool), pool)

	activated, err := store.SetActiveAudioAssetForLatestNarration(ctx, activationChapterID, activationAssetID)
	if err != nil {
		t.Fatalf("activate latest-matching ready asset: %v", err)
	}
	if activated.ID != activationAssetID || !activated.IsActive {
		t.Fatalf("unexpected activated asset: %+v", activated)
	}
	if got, err := store.GetActiveAudioAsset(ctx, activationChapterID); err != nil || got.ID != activationAssetID {
		t.Fatalf("expected active asset %s, got %+v err=%v", activationAssetID, got, err)
	}
}

func TestSetActiveAudioAssetForLatestNarrationSerializesWithNarrationCreation(t *testing.T) {
	pool := newAudioActivationTestPool(t)
	ctx := context.Background()
	insertAudioActivationFixture(t, pool, activationChapterID, activationNarrationOneID, activationAssetID)
	store := NewAudioStore(db.New(pool), pool)

	if _, err := store.CreateNarrationRevisionAtomically(ctx, audio.NarrationRevision{
		ID:                      activationNarrationTwoID,
		ChapterID:               activationChapterID,
		SourceContentRevisionID: "77777777-7777-7777-7777-777777777777",
		VoiceID:                 "voice-1",
		Script:                  "newer narration",
		Status:                  "DRAFT",
		CreatedBy:               "66666666-6666-6666-6666-666666666666",
	}); err != nil {
		t.Fatalf("create newer narration: %v", err)
	}

	_, err := store.SetActiveAudioAssetForLatestNarration(ctx, activationChapterID, activationAssetID)
	if !errors.Is(err, audio.ErrAudioAssetStaleForNarration) {
		t.Fatalf("expected stale activation after narration creation, got %v", err)
	}
}

func newAudioActivationTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL audio-activation regression tests")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect postgres: %v", err)
	}

	schema := fmt.Sprintf("p0_92_activation_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close(ctx)
		t.Fatalf("create test schema: %v", err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		admin.Close(ctx)
		t.Fatalf("parse postgres config: %v", err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		admin.Close(ctx)
		t.Fatalf("create test pool: %v", err)
	}
	if _, err := pool.Exec(ctx, `
CREATE TABLE narration_revisions (
    id UUID PRIMARY KEY,
    chapter_id UUID NOT NULL,
    revision_no INTEGER NOT NULL,
    source_content_revision_id UUID,
    voice_id TEXT,
    script TEXT,
    status TEXT NOT NULL DEFAULT 'DRAFT',
    generation_run_id UUID,
    created_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, revision_no)
);
CREATE TABLE audio_assets (
    id UUID PRIMARY KEY,
    chapter_id UUID NOT NULL,
    version_no INTEGER NOT NULL,
    source_narration_revision_id UUID REFERENCES narration_revisions (id),
    status TEXT NOT NULL DEFAULT 'PENDING',
    storage_key TEXT,
    mime_type TEXT,
    size_bytes BIGINT,
    duration_ms INTEGER,
    bitrate_kbps INTEGER,
    checksum TEXT,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    generation_run_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (chapter_id, version_no)
);
CREATE UNIQUE INDEX idx_audio_assets_active ON audio_assets (chapter_id) WHERE is_active = true;`); err != nil {
		pool.Close()
		_, _ = admin.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(ctx)
		t.Fatalf("create activation tables: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_, _ = admin.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
		admin.Close(context.Background())
	})
	return pool
}

func insertAudioActivationFixture(t *testing.T, pool *pgxpool.Pool, chapterID, narrationID, assetID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
INSERT INTO narration_revisions (id, chapter_id, revision_no, source_content_revision_id, voice_id, script, status, created_by)
VALUES ($1, $2, 1, $3, 'voice-1', 'script', 'DRAFT', $4)`,
		narrationID, chapterID, "88888888-8888-8888-8888-888888888888", "66666666-6666-6666-6666-666666666666"); err != nil {
		t.Fatalf("insert narration fixture: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO audio_assets (id, chapter_id, version_no, source_narration_revision_id, status, storage_key, mime_type, size_bytes, duration_ms, bitrate_kbps, is_active)
VALUES ($1, $2, 1, $3, 'READY', 'key', 'audio/mpeg', 1, 1000, 96, false)`,
		assetID, chapterID, narrationID); err != nil {
		t.Fatalf("insert audio fixture: %v", err)
	}
}

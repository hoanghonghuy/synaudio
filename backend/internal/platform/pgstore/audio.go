package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/platform/db"
)

type transactionBeginner interface {
	Begin(context.Context) (pgx.Tx, error)
}

// AudioStore implements audio.Store backed by PostgreSQL via sqlc.
type AudioStore struct {
	q       *db.Queries
	beginTx transactionBeginner
}

func NewAudioStore(q *db.Queries, executor ...db.DBTX) *AudioStore {
	store := &AudioStore{q: q}
	if len(executor) > 0 {
		if beginner, ok := executor[0].(transactionBeginner); ok {
			store.beginTx = beginner
		}
	}
	return store
}

// ============================================================
// Narration Revisions
// ============================================================

func (s *AudioStore) NextNarrationRevision(ctx context.Context, chapterID string) (int, error) {
	n, err := s.q.NextNarrationRevision(ctx, toUUID(chapterID))
	return int(n), err
}

func createNarrationRevision(ctx context.Context, q *db.Queries, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	row, err := q.CreateNarrationRevision(ctx, db.CreateNarrationRevisionParams{
		ID:                      toUUID(r.ID),
		ChapterID:               toUUID(r.ChapterID),
		RevisionNo:              int32(r.RevisionNo),
		SourceContentRevisionID: toUUID(r.SourceContentRevisionID),
		VoiceID:                 toText(r.VoiceID),
		Script:                  toText(r.Script),
		Status:                  r.Status,
		CreatedBy:               toUUID(r.CreatedBy),
	})
	if err != nil {
		return audio.NarrationRevision{}, err
	}
	return toNarrationRevision(row), nil
}

func (s *AudioStore) CreateNarrationRevision(ctx context.Context, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	return createNarrationRevision(ctx, s.q, r)
}

// CreateNarrationRevisionAtomically serializes version allocation per chapter and
// inserts the immutable revision before releasing the transaction-scoped lock.
// No schema/counter migration is required, and UNIQUE(chapter_id, revision_no)
// remains the database backstop.
func (s *AudioStore) CreateNarrationRevisionAtomically(ctx context.Context, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	if s.beginTx == nil {
		return audio.NarrationRevision{}, errors.New("atomic narration version allocation requires transaction-capable database")
	}
	tx, err := s.beginTx.Begin(ctx)
	if err != nil {
		return audio.NarrationRevision{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "narration-version:"+r.ChapterID); err != nil {
		return audio.NarrationRevision{}, fmt.Errorf("lock narration version allocation: %w", err)
	}
	qtx := s.q.WithTx(tx)
	n, err := qtx.NextNarrationRevision(ctx, toUUID(r.ChapterID))
	if err != nil {
		return audio.NarrationRevision{}, err
	}
	r.RevisionNo = int(n)
	created, err := createNarrationRevision(ctx, qtx, r)
	if err != nil {
		return audio.NarrationRevision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return audio.NarrationRevision{}, err
	}
	return created, nil
}

func (s *AudioStore) GetNarrationRevision(ctx context.Context, revisionID string) (audio.NarrationRevision, error) {
	row, err := s.q.GetNarrationRevision(ctx, toUUID(revisionID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.NarrationRevision{}, audio.ErrNarrationNotFound
		}
		return audio.NarrationRevision{}, err
	}
	return toNarrationRevision(row), nil
}

// ============================================================
// TTS Segments
// ============================================================

func (s *AudioStore) CreateTTSSegment(ctx context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	row, err := s.q.CreateTTSSegment(ctx, db.CreateTTSSegmentParams{
		ID:                  toUUID(seg.ID),
		NarrationRevisionID: toUUID(seg.NarrationRevisionID),
		SegmentNo:           int32(seg.SegmentNo),
		Text:                seg.Text,
		Status:              seg.Status,
		Provider:            toText(seg.Provider),
		Model:               toText(seg.Model),
		VoiceID:             toText(seg.VoiceID),
		DurationMs:          toInt4(seg.DurationMs),
		TempStorageKey:      toText(seg.TempStorageKey),
	})
	if err != nil {
		return audio.TTSSegment{}, err
	}
	return toTTSSegment(row), nil
}

func (s *AudioStore) GetTTSSegment(ctx context.Context, segmentID string) (audio.TTSSegment, error) {
	row, err := s.q.GetTTSSegment(ctx, toUUID(segmentID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.TTSSegment{}, audio.ErrTTSSegmentNotFound
		}
		return audio.TTSSegment{}, err
	}
	return toTTSSegment(row), nil
}

func (s *AudioStore) UpdateTTSSegment(ctx context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	row, err := s.q.UpdateTTSSegment(ctx, db.UpdateTTSSegmentParams{
		ID:             toUUID(seg.ID),
		Status:         seg.Status,
		Provider:       toText(seg.Provider),
		Model:          toText(seg.Model),
		DurationMs:     toInt4(seg.DurationMs),
		TempStorageKey: toText(seg.TempStorageKey),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.TTSSegment{}, audio.ErrTTSSegmentNotFound
		}
		return audio.TTSSegment{}, err
	}
	return toTTSSegment(row), nil
}

// ============================================================
// Audio Assets
// ============================================================

func (s *AudioStore) NextAudioVersion(ctx context.Context, chapterID string) (int, error) {
	n, err := s.q.NextAudioVersion(ctx, toUUID(chapterID))
	return int(n), err
}

func createAudioAsset(ctx context.Context, q *db.Queries, a audio.AudioAsset) (audio.AudioAsset, error) {
	row, err := q.CreateAudioAsset(ctx, db.CreateAudioAssetParams{
		ID:                        toUUID(a.ID),
		ChapterID:                 toUUID(a.ChapterID),
		VersionNo:                 int32(a.VersionNo),
		SourceNarrationRevisionID: toUUID(a.SourceNarrationRevisionID),
		Status:                    a.Status,
		StorageKey:                toText(a.StorageKey),
		MimeType:                  toText(a.MimeType),
		SizeBytes:                 pgtypeInt8(a.SizeBytes),
		DurationMs:                toInt4(a.DurationMs),
		BitrateKbps:               toInt4(a.BitrateKbps),
		Checksum:                  toText(a.Checksum),
		IsActive:                  a.IsActive,
	})
	if err != nil {
		return audio.AudioAsset{}, err
	}
	return toAudioAsset(row), nil
}

func (s *AudioStore) CreateAudioAsset(ctx context.Context, a audio.AudioAsset) (audio.AudioAsset, error) {
	return createAudioAsset(ctx, s.q, a)
}

// CreateAudioAssetAtomically serializes version allocation per chapter and
// commits the READY metadata row in the same transaction. The media object key
// is attempt-unique, so an upload performed before this metadata transaction
// cannot collide with another concurrent synthesis attempt.
func (s *AudioStore) CreateAudioAssetAtomically(ctx context.Context, a audio.AudioAsset) (audio.AudioAsset, error) {
	if s.beginTx == nil {
		return audio.AudioAsset{}, errors.New("atomic audio version allocation requires transaction-capable database")
	}
	tx, err := s.beginTx.Begin(ctx)
	if err != nil {
		return audio.AudioAsset{}, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, "audio-version:"+a.ChapterID); err != nil {
		return audio.AudioAsset{}, fmt.Errorf("lock audio version allocation: %w", err)
	}
	qtx := s.q.WithTx(tx)
	n, err := qtx.NextAudioVersion(ctx, toUUID(a.ChapterID))
	if err != nil {
		return audio.AudioAsset{}, err
	}
	a.VersionNo = int(n)
	created, err := createAudioAsset(ctx, qtx, a)
	if err != nil {
		return audio.AudioAsset{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return audio.AudioAsset{}, err
	}
	return created, nil
}

func (s *AudioStore) GetAudioAsset(ctx context.Context, assetID string) (audio.AudioAsset, error) {
	row, err := s.q.GetAudioAsset(ctx, toUUID(assetID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
		}
		return audio.AudioAsset{}, err
	}
	return toAudioAsset(row), nil
}

func (s *AudioStore) GetActiveAudioAsset(ctx context.Context, chapterID string) (audio.AudioAsset, error) {
	row, err := s.q.GetActiveAudioAsset(ctx, toUUID(chapterID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
		}
		return audio.AudioAsset{}, err
	}
	return toAudioAsset(row), nil
}

func (s *AudioStore) SetActiveAudioAsset(ctx context.Context, chapterID, assetID string) (audio.AudioAsset, error) {
	rows, err := s.q.SetActiveAudioAsset(ctx, db.SetActiveAudioAssetParams{
		ChapterID: toUUID(chapterID),
		ID:        toUUID(assetID),
	})
	if err != nil {
		return audio.AudioAsset{}, err
	}
	for _, row := range rows {
		if fromUUID(row.ID) == assetID {
			return toAudioAsset(row), nil
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}

// ============================================================
// Converters
// ============================================================

func toNarrationRevision(row db.NarrationRevision) audio.NarrationRevision {
	return audio.NarrationRevision{
		ID:                      fromUUID(row.ID),
		ChapterID:               fromUUID(row.ChapterID),
		RevisionNo:              int(row.RevisionNo),
		SourceContentRevisionID: fromUUID(row.SourceContentRevisionID),
		VoiceID:                 fromText(row.VoiceID),
		Script:                  fromText(row.Script),
		Status:                  row.Status,
		CreatedBy:               fromUUID(row.CreatedBy),
	}
}

func toAudioAsset(row db.AudioAsset) audio.AudioAsset {
	return audio.AudioAsset{
		ID:                        fromUUID(row.ID),
		ChapterID:                 fromUUID(row.ChapterID),
		VersionNo:                 int(row.VersionNo),
		SourceNarrationRevisionID: fromUUID(row.SourceNarrationRevisionID),
		Status:                    row.Status,
		StorageKey:                fromText(row.StorageKey),
		MimeType:                  fromText(row.MimeType),
		SizeBytes:                 row.SizeBytes.Int64,
		DurationMs:                int(row.DurationMs.Int32),
		BitrateKbps:               int(row.BitrateKbps.Int32),
		Checksum:                  fromText(row.Checksum),
		IsActive:                  row.IsActive,
	}
}

func toTTSSegment(row db.TtsSegment) audio.TTSSegment {
	return audio.TTSSegment{
		ID:                  fromUUID(row.ID),
		NarrationRevisionID: fromUUID(row.NarrationRevisionID),
		SegmentNo:           int(row.SegmentNo),
		Text:                row.Text,
		Status:              row.Status,
		Provider:            fromText(row.Provider),
		Model:               fromText(row.Model),
		VoiceID:             fromText(row.VoiceID),
		DurationMs:          int(row.DurationMs.Int32),
		TempStorageKey:      fromText(row.TempStorageKey),
	}
}

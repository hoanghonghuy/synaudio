package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/platform/db"
	"github.com/synaudio/synaudio/backend/internal/retcon"
)

// RetconStore implements retcon.Store backed by PostgreSQL via sqlc.
type RetconStore struct {
	q *db.Queries
}

func NewRetconStore(q *db.Queries) *RetconStore {
	return &RetconStore{q: q}
}

func (s *RetconStore) CreateRetconRequest(ctx context.Context, r retcon.RetconRequest) (retcon.RetconRequest, error) {
	row, err := s.q.CreateRetconRequest(ctx, db.CreateRetconRequestParams{
		ID:              toUUID(r.ID),
		StoryID:         toUUID(r.StoryID),
		TargetChapterID: toUUID(r.TargetChapterID),
		Status:          r.Status,
		ImpactScope:     r.ImpactScope,
		ProposedChange:  toText(r.ProposedChange),
		Reason:          r.Reason,
		RequestedBy:     toUUID(r.RequestedBy),
	})
	if err != nil {
		return retcon.RetconRequest{}, err
	}
	return toRetconRequest(row), nil
}

func (s *RetconStore) GetRetconRequest(ctx context.Context, id string) (retcon.RetconRequest, error) {
	row, err := s.q.GetRetconRequest(ctx, toUUID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return retcon.RetconRequest{}, retcon.ErrRetconNotFound
		}
		return retcon.RetconRequest{}, err
	}
	return toRetconRequest(row), nil
}

func (s *RetconStore) ListRetconRequests(ctx context.Context, storyID string) ([]retcon.RetconRequest, error) {
	rows, err := s.q.ListRetconRequests(ctx, toUUID(storyID))
	if err != nil {
		return nil, err
	}
	out := make([]retcon.RetconRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, toRetconRequest(row))
	}
	return out, nil
}

func (s *RetconStore) UpdateRetconRequest(ctx context.Context, r retcon.RetconRequest) (retcon.RetconRequest, error) {
	row, err := s.q.UpdateRetconRequest(ctx, db.UpdateRetconRequestParams{
		ID:         toUUID(r.ID),
		Status:     r.Status,
		ApprovedBy: toUUID(r.ApprovedBy),
		AppliedBy:  toUUID(r.AppliedBy),
	})
	if err != nil {
		return retcon.RetconRequest{}, err
	}
	return toRetconRequest(row), nil
}

func toRetconRequest(row db.RetconRequest) retcon.RetconRequest {
	return retcon.RetconRequest{
		ID:              fromUUID(row.ID),
		StoryID:         fromUUID(row.StoryID),
		TargetChapterID: fromUUID(row.TargetChapterID),
		Status:          row.Status,
		ImpactScope:     row.ImpactScope,
		ProposedChange:  fromText(row.ProposedChange),
		Reason:          row.Reason,
		RequestedBy:     fromUUID(row.RequestedBy),
		ApprovedBy:      fromUUID(row.ApprovedBy),
		AppliedBy:       fromUUID(row.AppliedBy),
	}
}

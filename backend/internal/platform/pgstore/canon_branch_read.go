package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/synaudio/synaudio/backend/internal/planning"
)

func (s *PlanningStore) GetActiveOfficialCanonBranch(ctx context.Context, storyID string) (planning.CanonBranch, error) {
	row, err := s.q.GetActiveOfficialCanonBranch(ctx, toUUID(storyID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return planning.CanonBranch{}, planning.ErrCanonBranchNotFound
		}
		return planning.CanonBranch{}, err
	}
	return toCanonBranch(row), nil
}

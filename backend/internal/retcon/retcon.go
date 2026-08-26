package retcon

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrRetconNotFound = errors.New("retcon request not found")
	ErrRetconNotReady = errors.New("retcon request not ready to apply")
)

// RetconRequest is a controlled request to change published story history.
type RetconRequest struct {
	ID              string
	StoryID         string
	TargetChapterID string
	Status          string
	ImpactScope     string
	ProposedChange  string
	Reason          string
	RequestedBy     string
	ApprovedBy      string
	AppliedBy       string
}

// CreateRetconInput is the input for creating a retcon request.
type CreateRetconInput struct {
	StoryID         string
	TargetChapterID string
	ProposedChange  string
	Reason          string
	RequestedBy     string
}

// Store is the persistence boundary for the retcon service.
type Store interface {
	CreateRetconRequest(ctx context.Context, r RetconRequest) (RetconRequest, error)
	GetRetconRequest(ctx context.Context, id string) (RetconRequest, error)
	ListRetconRequests(ctx context.Context, storyID string) ([]RetconRequest, error)
	UpdateRetconRequest(ctx context.Context, r RetconRequest) (RetconRequest, error)
}

// Service orchestrates retcon requests.
type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateRetconRequest creates a new DRAFT retcon request.
func (s *Service) CreateRetconRequest(ctx context.Context, in CreateRetconInput) (RetconRequest, error) {
	if in.Reason == "" {
		return RetconRequest{}, errors.New("reason must not be empty")
	}
	if in.StoryID == "" {
		return RetconRequest{}, errors.New("story id must not be empty")
	}

	r := RetconRequest{
		ID:              uuid.NewString(),
		StoryID:         in.StoryID,
		TargetChapterID: in.TargetChapterID,
		Status:          "DRAFT",
		ImpactScope:     "LOCAL",
		ProposedChange:  in.ProposedChange,
		Reason:          in.Reason,
		RequestedBy:     in.RequestedBy,
	}

	return s.store.CreateRetconRequest(ctx, r)
}

// ApproveRetconRequest marks a retcon request as APPROVED.
func (s *Service) ApproveRetconRequest(ctx context.Context, id, approvedBy string) (RetconRequest, error) {
	r, err := s.store.GetRetconRequest(ctx, id)
	if err != nil {
		return RetconRequest{}, err
	}

	r.Status = "APPROVED"
	r.ApprovedBy = approvedBy

	return s.store.UpdateRetconRequest(ctx, r)
}

// CancelRetconRequest marks a retcon request as CANCELLED.
func (s *Service) CancelRetconRequest(ctx context.Context, id string) (RetconRequest, error) {
	r, err := s.store.GetRetconRequest(ctx, id)
	if err != nil {
		return RetconRequest{}, err
	}

	r.Status = "CANCELLED"

	return s.store.UpdateRetconRequest(ctx, r)
}

// ListRetconRequests returns all retcon requests for a story.
func (s *Service) ListRetconRequests(ctx context.Context, storyID string) ([]RetconRequest, error) {
	return s.store.ListRetconRequests(ctx, storyID)
}

// AnalyzeRetconRequest moves a DRAFT retcon into ANALYZING.
func (s *Service) AnalyzeRetconRequest(ctx context.Context, id string) (RetconRequest, error) {
	r, err := s.store.GetRetconRequest(ctx, id)
	if err != nil {
		return RetconRequest{}, err
	}

	r.Status = "ANALYZING"

	return s.store.UpdateRetconRequest(ctx, r)
}

// MarkReadyToApply moves an APPROVED retcon into READY_TO_APPLY.
func (s *Service) MarkReadyToApply(ctx context.Context, id string) (RetconRequest, error) {
	r, err := s.store.GetRetconRequest(ctx, id)
	if err != nil {
		return RetconRequest{}, err
	}

	if r.Status != "APPROVED" {
		return RetconRequest{}, ErrRetconNotReady
	}

	r.Status = "READY_TO_APPLY"

	return s.store.UpdateRetconRequest(ctx, r)
}

// ApplyRetconRequest atomically applies a READY_TO_APPLY retcon.
func (s *Service) ApplyRetconRequest(ctx context.Context, id, appliedBy string) (RetconRequest, error) {
	r, err := s.store.GetRetconRequest(ctx, id)
	if err != nil {
		return RetconRequest{}, err
	}

	if r.Status != "READY_TO_APPLY" {
		return RetconRequest{}, ErrRetconNotReady
	}

	r.Status = "APPLIED"
	r.AppliedBy = appliedBy

	return s.store.UpdateRetconRequest(ctx, r)
}

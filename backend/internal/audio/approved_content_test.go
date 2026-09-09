package audio

import (
	"context"
	"errors"
	"testing"

	"github.com/synaudio/synaudio/backend/internal/generation"
)

type mapApprovedContentAuthority struct {
	chapters map[string]string
	approved map[string]bool
}

func (m mapApprovedContentAuthority) RequireApprovedContentRevision(_ context.Context, chapterID, revisionID string) error {
	owner, ok := m.chapters[revisionID]
	if !ok {
		return generation.ErrContentRevisionNotFound
	}
	if owner != chapterID {
		return generation.ErrContentRevisionChapterMismatch
	}
	if !m.approved[revisionID] {
		return generation.ErrContentRevisionNotApproved
	}
	return nil
}

type permissiveApprovedContentAuthority struct{}

func (permissiveApprovedContentAuthority) RequireApprovedContentRevision(context.Context, string, string) error {
	return nil
}

func newTestService(store Store, opts ...Option) *Service {
	allOpts := append([]Option{WithApprovedContentAuthority(permissiveApprovedContentAuthority{})}, opts...)
	return NewService(store, allOpts...)
}

func TestCreateNarrationRevisionRequiresApprovedContentRevision(t *testing.T) {
	audioStore := newFakeStore()
	svc := NewService(audioStore, WithApprovedContentAuthority(mapApprovedContentAuthority{
		chapters: map[string]string{"approved-rev": "c1"},
		approved: map[string]bool{"approved-rev": true},
	}))

	nar, err := svc.CreateNarrationRevision(context.Background(), "c1", "approved-rev", "voice-1", "script text", "u1")
	if err != nil {
		t.Fatalf("create narration from approved revision: %v", err)
	}
	if nar.SourceContentRevisionID != "approved-rev" {
		t.Fatalf("expected source %q, got %q", "approved-rev", nar.SourceContentRevisionID)
	}
}

func TestCreateNarrationRevisionRejectsCandidateContentRevision(t *testing.T) {
	audioStore := newFakeStore()
	svc := NewService(audioStore, WithApprovedContentAuthority(mapApprovedContentAuthority{
		chapters: map[string]string{"candidate-rev": "c1"},
		approved: map[string]bool{"candidate-rev": false},
	}))

	if _, err := svc.CreateNarrationRevision(context.Background(), "c1", "candidate-rev", "voice-1", "script text", "u1"); !errors.Is(err, generation.ErrContentRevisionNotApproved) {
		t.Fatalf("expected ErrContentRevisionNotApproved, got %v", err)
	}
}

func TestCreateNarrationRevisionRejectsCrossChapterContentRevision(t *testing.T) {
	audioStore := newFakeStore()
	svc := NewService(audioStore, WithApprovedContentAuthority(mapApprovedContentAuthority{
		chapters: map[string]string{"approved-rev": "c1"},
		approved: map[string]bool{"approved-rev": true},
	}))

	if _, err := svc.CreateNarrationRevision(context.Background(), "c2", "approved-rev", "voice-1", "script text", "u1"); !errors.Is(err, generation.ErrContentRevisionChapterMismatch) {
		t.Fatalf("expected ErrContentRevisionChapterMismatch, got %v", err)
	}
}

func TestCreateNarrationRevisionRejectsMissingContentRevision(t *testing.T) {
	audioStore := newFakeStore()
	svc := NewService(audioStore, WithApprovedContentAuthority(mapApprovedContentAuthority{
		chapters: map[string]string{},
		approved: map[string]bool{},
	}))

	if _, err := svc.CreateNarrationRevision(context.Background(), "c1", "missing", "voice-1", "script text", "u1"); !errors.Is(err, generation.ErrContentRevisionNotFound) {
		t.Fatalf("expected ErrContentRevisionNotFound, got %v", err)
	}
}

func TestCreateNarrationRevisionFailsClosedWithoutAuthority(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store)

	if _, err := svc.CreateNarrationRevision(context.Background(), "c1", "cr1", "voice-1", "script text", "u1"); !errors.Is(err, ErrApprovedContentAuthorityRequired) {
		t.Fatalf("expected ErrApprovedContentAuthorityRequired, got %v", err)
	}
}

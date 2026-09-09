package planning

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/synaudio/synaudio/backend/internal/audio"
)

type integrationAudioStore struct {
	active     audio.AudioAsset
	hasActive  bool
	narrations []audio.NarrationRevision
}

func (s *integrationAudioStore) NextNarrationRevision(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *integrationAudioStore) CreateNarrationRevision(_ context.Context, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	return r, nil
}

func (s *integrationAudioStore) GetNarrationRevision(_ context.Context, revisionID string) (audio.NarrationRevision, error) {
	return audio.NarrationRevision{ID: revisionID}, nil
}

func (s *integrationAudioStore) GetLatestNarrationRevision(_ context.Context, chapterID string) (audio.NarrationRevision, error) {
	if len(s.narrations) == 0 {
		return audio.NarrationRevision{}, audio.ErrNarrationNotFound
	}
	latest := s.narrations[len(s.narrations)-1]
	if latest.ChapterID != chapterID {
		return audio.NarrationRevision{}, audio.ErrNarrationNotFound
	}
	return latest, nil
}

func (s *integrationAudioStore) CreateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	return seg, nil
}

func (s *integrationAudioStore) GetTTSSegment(_ context.Context, segmentID string) (audio.TTSSegment, error) {
	return audio.TTSSegment{ID: segmentID}, nil
}

func (s *integrationAudioStore) UpdateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	return seg, nil
}

func (s *integrationAudioStore) NextAudioVersion(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *integrationAudioStore) CreateAudioAsset(_ context.Context, a audio.AudioAsset) (audio.AudioAsset, error) {
	return a, nil
}

func (s *integrationAudioStore) GetAudioAsset(_ context.Context, assetID string) (audio.AudioAsset, error) {
	return audio.AudioAsset{ID: assetID}, nil
}

func (s *integrationAudioStore) GetActiveAudioAsset(_ context.Context, chapterID string) (audio.AudioAsset, error) {
	if s.hasActive && s.active.ChapterID == chapterID {
		return s.active, nil
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}

func (s *integrationAudioStore) IssueListenerEligibleAudioURL(
	ctx context.Context,
	chapterID string,
	check audio.ListenerEligibilityChecker,
	issue audio.ListenerEligibleAudioIssuer,
) (string, error) {
	if check != nil {
		if err := check(ctx, chapterID); err != nil {
			return "", err
		}
	}

	latestBefore, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, audio.ErrNarrationNotFound) {
			return "", audio.ErrListenerAudioNotEligible
		}
		return "", err
	}

	asset, err := s.GetActiveAudioAsset(ctx, chapterID)
	if err != nil {
		if errors.Is(err, audio.ErrAudioAssetNotFound) {
			return "", audio.ErrListenerAudioNotEligible
		}
		return "", err
	}
	if asset.Status != "READY" {
		return "", audio.ErrListenerAudioNotEligible
	}

	latestAfter, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		if errors.Is(err, audio.ErrNarrationNotFound) {
			return "", audio.ErrListenerAudioNotEligible
		}
		return "", err
	}
	if latestAfter.ID != latestBefore.ID || asset.SourceNarrationRevisionID != latestAfter.ID {
		return "", audio.ErrListenerAudioNotEligible
	}

	return issue(ctx, asset)
}

func (s *integrationAudioStore) GetLatestReadyAudioAssetForNarration(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, audio.ErrReadyAudioAssetNotFound
}

func (s *integrationAudioStore) SetActiveAudioAsset(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, nil
}

func (s *integrationAudioStore) SetActiveAudioAssetForLatestNarration(_ context.Context, _, _ string) (audio.AudioAsset, error) {
	return audio.AudioAsset{}, nil
}

type integrationPresigner struct{}

func (integrationPresigner) PresignedGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://cdn.example.com/" + key, nil
}

func newListenerAudioHandler(t *testing.T, chapterStatus string, storyStatus, storyVisibility string, hasActive bool) http.Handler {
	store := newPublishFakeStore()
	ch, err := store.CreateChapter(t.Context(), Chapter{ID: "ch-1", StoryID: "s1", Title: "Chapter 1", Status: chapterStatus})
	if err != nil {
		t.Fatalf("create chapter: %v", err)
	}

	gate := NewListenerEligibility(store, &fakeStoryVisibilityReader{
		status:     storyStatus,
		visibility: storyVisibility,
	})

	audioStore := &integrationAudioStore{
		hasActive: hasActive,
		active: audio.AudioAsset{
			ID:                        "asset-1",
			ChapterID:                 ch.ID,
			Status:                    "READY",
			StorageKey:                "audio/ch-1/v1.mp3",
			IsActive:                  true,
			SourceNarrationRevisionID: "nar-1",
		},
		narrations: []audio.NarrationRevision{{ID: "nar-1", ChapterID: ch.ID, RevisionNo: 1}},
	}
	svc := audio.NewService(audioStore, audio.WithPresigner(integrationPresigner{}))
	svc.SetListenerAudioGate(gate)
	return audio.NewHandler(svc)
}

func TestListenerAudioURLHandlerWithRealEligibilityGateRejectsUnpublishedChapter(t *testing.T) {
	handler := newListenerAudioHandler(t, "READY", "ACTIVE", "PUBLIC", true)

	req := httptest.NewRequest(http.MethodGet, "/chapters/ch-1/audio-url", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unpublished chapter, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAudioNotAvailableCode(t, rec.Body.Bytes())
}

func TestListenerAudioURLHandlerWithRealEligibilityGateRejectsPrivateStory(t *testing.T) {
	handler := newListenerAudioHandler(t, "PUBLISHED", "ACTIVE", "PRIVATE", true)

	req := httptest.NewRequest(http.MethodGet, "/chapters/ch-1/audio-url", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for private story, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAudioNotAvailableCode(t, rec.Body.Bytes())
}

func TestListenerAudioURLHandlerWithRealEligibilityGateRejectsInactiveStory(t *testing.T) {
	handler := newListenerAudioHandler(t, "PUBLISHED", "DRAFT", "PUBLIC", true)

	req := httptest.NewRequest(http.MethodGet, "/chapters/ch-1/audio-url", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for inactive story, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAudioNotAvailableCode(t, rec.Body.Bytes())
}

func TestListenerAudioURLHandlerWithRealEligibilityGateRejectsStaleActiveAudio(t *testing.T) {
	store := newPublishFakeStore()
	ch, err := store.CreateChapter(context.Background(), Chapter{ID: "ch-1", StoryID: "s1", Title: "Chapter 1", Status: "PUBLISHED"})
	if err != nil {
		t.Fatalf("create chapter: %v", err)
	}

	gate := NewListenerEligibility(store, &fakeStoryVisibilityReader{status: "ACTIVE", visibility: "PUBLIC"})
	audioStore := &integrationAudioStore{
		hasActive: true,
		active: audio.AudioAsset{
			ID:                        "asset-1",
			ChapterID:                 ch.ID,
			Status:                    "READY",
			StorageKey:                "audio/ch-1/v1.mp3",
			IsActive:                  true,
			SourceNarrationRevisionID: "nar-1",
		},
		narrations: []audio.NarrationRevision{
			{ID: "nar-1", ChapterID: ch.ID, RevisionNo: 1},
			{ID: "nar-2", ChapterID: ch.ID, RevisionNo: 2},
		},
	}
	svc := audio.NewService(audioStore, audio.WithPresigner(integrationPresigner{}))
	svc.SetListenerAudioGate(gate)
	handler := audio.NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/chapters/"+ch.ID+"/audio-url", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for stale active audio, got %d: %s", rec.Code, rec.Body.String())
	}
	assertAudioNotAvailableCode(t, rec.Body.Bytes())
}

func TestListenerAudioURLHandlerWithRealEligibilityGateReturnsPresignedURL(t *testing.T) {
	handler := newListenerAudioHandler(t, "PUBLISHED", "ACTIVE", "PUBLIC", true)

	req := httptest.NewRequest(http.MethodGet, "/chapters/ch-1/audio-url", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for eligible chapter, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["url"] != "https://cdn.example.com/audio/ch-1/v1.mp3" {
		t.Fatalf("unexpected url: %q", body["url"])
	}
}

func assertAudioNotAvailableCode(t *testing.T, body []byte) {
	var payload map[string]map[string]string
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if payload["error"]["code"] != "AUDIO_NOT_AVAILABLE" {
		t.Fatalf("expected AUDIO_NOT_AVAILABLE, got %#v", payload["error"])
	}
}

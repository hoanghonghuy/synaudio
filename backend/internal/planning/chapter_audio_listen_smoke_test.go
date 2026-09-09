package planning

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/synaudio/synaudio/backend/internal/audio"
	"github.com/synaudio/synaudio/backend/internal/generation"
	"github.com/synaudio/synaudio/backend/internal/story"
)

// Repository-owned deterministic smoke for #97: Chapter → generation → approval →
// narration → TTS READY audio → activation → publish → listener audio URL.
func TestSmokeChapterAudioListenPipeline(t *testing.T) {
	env := newSmokePipelineEnv(t)
	actor := func(context.Context, *http.Request) (string, error) { return "admin-1", nil }

	genHandler := generation.NewLatestRunAwareHandler(generation.NewHandler(env.genSvc, actor), env.genSvc)
	audioHandler := audio.NewHandler(env.audioSvc)
	planHandler := NewHandler(env.planSvc)

	router := flattenHandlers(genHandler, audioHandler, planHandler)

	storyID := "story-smoke"
	chapterID := "chapter-smoke"
	env.seedChapter(storyID, chapterID)

	env.postJSON(t, router, http.MethodPost, "/admin/stories/"+storyID+"/batch-generate", map[string]any{
		"chapter_ids": []string{chapterID},
	})

	if err := env.worker.ProcessOne(context.Background()); err != nil && !errors.Is(err, generation.ErrNoRunnableJob) {
		t.Fatalf("worker: %v", err)
	}

	content := env.getJSON(t, router, "/admin/chapters/"+chapterID+"/content")
	jobView, ok := content["generation_job"].(map[string]any)
	if !ok || jobView["Observation"] != "succeeded" {
		t.Fatalf("expected succeeded generation_job, got %#v", content["generation_job"])
	}

	revisions, _ := content["revisions"].([]any)
	if len(revisions) == 0 {
		t.Fatal("expected content revision from generation")
	}
	rev := revisions[0].(map[string]any)
	revID := rev["ID"].(string)

	env.postJSON(t, router, http.MethodPost, "/admin/chapters/"+chapterID+"/approve", map[string]any{
		"revision_id": revID,
	})

	nar := env.postJSON(t, router, http.MethodPost, "/admin/chapters/"+chapterID+"/narration", map[string]any{
		"source_content_revision_id": revID,
		"voice_id":                   "voice-smoke",
		"script":                     rev["ContentText"],
	})
	narID := nar["ID"].(string)

	asset := env.postJSON(t, router, http.MethodPost, "/admin/chapters/"+chapterID+"/narration/"+narID+"/synthesize", nil)
	assetID := asset["ID"].(string)
	env.postJSON(t, router, http.MethodPost, "/admin/chapters/"+chapterID+"/audio/"+assetID+"/activate", nil)

	if _, err := env.planStore.UpdateChapterStatus(context.Background(), chapterID, "READY"); err != nil {
		t.Fatalf("set READY: %v", err)
	}

	unpublished := env.getRecorder(t, router, "/chapters/"+chapterID+"/audio-url")
	if unpublished.Code != http.StatusNotFound {
		t.Fatalf("unpublished chapter must deny listener audio, got %d: %s", unpublished.Code, unpublished.Body.String())
	}

	env.postJSON(t, router, http.MethodPost, "/admin/chapters/"+chapterID+"/publish", nil)

	published := env.getRecorder(t, router, "/chapters/"+chapterID+"/audio-url")
	if published.Code != http.StatusOK {
		t.Fatalf("published chapter expected listener audio URL, got %d: %s", published.Code, published.Body.String())
	}
	var audioBody map[string]string
	if err := json.NewDecoder(published.Body).Decode(&audioBody); err != nil {
		t.Fatalf("decode audio url: %v", err)
	}
	if audioBody["url"] == "" {
		t.Fatalf("expected presigned listener url, got %#v", audioBody)
	}
}

type smokePipelineEnv struct {
	genSvc    *generation.Service
	worker    *generation.Worker
	audioSvc  *audio.Service
	planSvc   *Service
	planStore *publishFakeStore
}

func newSmokePipelineEnv(t *testing.T) *smokePipelineEnv {
	genStore := generation.NewMemoryStore()
	if err := generation.BindWriterPlan(genStore, "chapter-smoke", generation.WriterJobInput{
		ChapterID:          "chapter-smoke",
		PlanRevisionID:     "plan-smoke",
		Plan:               map[string]any{"beat": "smoke opening"},
		BaseCanonVersionID: "canon-smoke",
	}); err != nil {
		t.Fatalf("bind writer plan: %v", err)
	}

	genSvc := generation.NewService(genStore, generation.WithTextAI(&smokeTextAI{text: "Smoke chapter prose for listener path."}))
	worker := generation.NewWorker(genSvc, "smoke-worker", func(ctx context.Context, job generation.GenerationJob) error {
		_, err := genSvc.ExecuteWriterJob(ctx, job)
		return err
	})

	audioStore := newSmokeAudioStore()
	objects := newSmokeObjectStorage()
	audioSvc := audio.NewService(audioStore,
		audio.WithTTS(audio.MockTTS{}),
		audio.WithObjectStorage(objects),
		audio.WithPresigner(smokePresigner{}),
		audio.WithAudioProcessor(audio.NewMockAudioProcessor()),
		audio.WithApprovedContentAuthority(genSvc),
	)

	planStore := newPublishFakeStore()
	storySvc := story.NewService(newSmokeStoryStore())
	planSvc := NewService(planStore,
		WithPublishChecker(NewCompositePublishChecker(planStore, genSvc, audioSvc, storySvc)),
	)
	audioSvc.SetListenerAudioGate(NewListenerEligibility(planStore, storySvc))

	return &smokePipelineEnv{
		genSvc:    genSvc,
		worker:    worker,
		audioSvc:  audioSvc,
		planSvc:   planSvc,
		planStore: planStore,
	}
}

func (e *smokePipelineEnv) seedChapter(storyID, chapterID string) {
	ch, _ := e.planStore.CreateChapter(context.Background(), Chapter{
		ID:      chapterID,
		StoryID: storyID,
		Title:   "Smoke Chapter",
		Status:  "DRAFT",
	})
	e.planStore.chapters[storyID] = []Chapter{ch}
}

func (e *smokePipelineEnv) postJSON(t *testing.T, router http.Handler, method, path string, body any) map[string]any {
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s expected success, got %d: %s", method, path, rec.Code, rec.Body.String())
	}
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

func (e *smokePipelineEnv) getJSON(t *testing.T, router http.Handler, path string) map[string]any {
	rec := e.getRecorder(t, router, path)
	var out map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}

func (e *smokePipelineEnv) getRecorder(t *testing.T, router http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func flattenHandlers(handlers ...http.Handler) chi.Router {
	router := chi.NewRouter()
	for _, handler := range handlers {
		routes, ok := handler.(chi.Routes)
		if !ok {
			panic("smoke handler must expose chi routes")
		}
		if err := chi.Walk(routes, func(method, route string, h http.Handler, _ ...func(http.Handler) http.Handler) error {
			router.Method(method, route, h)
			return nil
		}); err != nil {
			panic("smoke route flatten failed: " + err.Error())
		}
	}
	return router
}

type smokeTextAI struct{ text string }

func (p *smokeTextAI) GenerateText(_ context.Context, in generation.TextAIInput) (generation.TextAIOutput, error) {
	return generation.TextAIOutput{Text: p.text, Provider: "smoke", Model: "smoke"}, nil
}

type smokeStoryStore struct {
	stories map[string]story.Story
}

func newSmokeStoryStore() *smokeStoryStore {
	return &smokeStoryStore{stories: map[string]story.Story{
		"story-smoke": {ID: "story-smoke", Status: story.StatusActive, Visibility: story.VisibilityPublic},
	}}
}

func (s *smokeStoryStore) GetStory(_ context.Context, storyID string) (story.Story, error) {
	st, ok := s.stories[storyID]
	if !ok {
		return story.Story{}, story.ErrStoryNotFound
	}
	return st, nil
}

func (s *smokeStoryStore) CreateStory(context.Context, story.Story) (story.Story, error) {
	return story.Story{}, errors.New("not implemented")
}
func (s *smokeStoryStore) CreateGenerationPolicy(context.Context, story.GenerationPolicy) error { return nil }
func (s *smokeStoryStore) HasGenerationPolicy(context.Context, string) (bool, error) { return true, nil }
func (s *smokeStoryStore) SlugExists(context.Context, string) (bool, error) { return false, nil }
func (s *smokeStoryStore) ListGenres(context.Context) ([]story.Genre, error) { return nil, nil }
func (s *smokeStoryStore) ListStories(context.Context, bool) ([]story.Story, error) { return nil, nil }
func (s *smokeStoryStore) GetWorkflowSettings(context.Context, string) (story.WorkflowSettings, error) {
	return story.WorkflowSettings{}, nil
}
func (s *smokeStoryStore) UpdateWorkflowSettings(context.Context, story.WorkflowSettings) (story.WorkflowSettings, error) {
	return story.WorkflowSettings{}, nil
}
func (s *smokeStoryStore) NextContentProfileVersion(context.Context, string) (int, error) { return 1, nil }
func (s *smokeStoryStore) CreateContentProfileVersion(context.Context, story.ContentProfileVersion) (story.ContentProfileVersion, error) {
	return story.ContentProfileVersion{}, nil
}
func (s *smokeStoryStore) GetCurrentContentProfile(context.Context, string) (story.ContentProfileVersion, error) {
	return story.ContentProfileVersion{}, story.ErrContentProfileNotFound
}
func (s *smokeStoryStore) UpdateStory(context.Context, story.Story) (story.Story, error) { return story.Story{}, nil }
func (s *smokeStoryStore) CreateStoryAsset(context.Context, story.StoryAsset) (story.StoryAsset, error) {
	return story.StoryAsset{}, nil
}
func (s *smokeStoryStore) LinkCoverAsset(context.Context, string, string) error { return nil }
func (s *smokeStoryStore) SearchStories(context.Context, story.SearchStoriesInput) ([]story.Story, error) {
	return nil, nil
}

type smokeAudioStore struct {
	narrations map[string][]audio.NarrationRevision
	nextNar    map[string]int
	segments   map[string][]audio.TTSSegment
	assets     map[string][]audio.AudioAsset
	nextVer    map[string]int
}

func newSmokeAudioStore() *smokeAudioStore {
	return &smokeAudioStore{
		narrations: map[string][]audio.NarrationRevision{},
		nextNar:    map[string]int{},
		segments:   map[string][]audio.TTSSegment{},
		assets:     map[string][]audio.AudioAsset{},
		nextVer:    map[string]int{},
	}
}

func (s *smokeAudioStore) NextNarrationRevision(_ context.Context, chapterID string) (int, error) {
	s.nextNar[chapterID]++
	return s.nextNar[chapterID], nil
}
func (s *smokeAudioStore) CreateNarrationRevision(_ context.Context, r audio.NarrationRevision) (audio.NarrationRevision, error) {
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	s.narrations[r.ChapterID] = append(s.narrations[r.ChapterID], r)
	return r, nil
}
func (s *smokeAudioStore) GetNarrationRevision(_ context.Context, revisionID string) (audio.NarrationRevision, error) {
	for _, rs := range s.narrations {
		for _, r := range rs {
			if r.ID == revisionID {
				return r, nil
			}
		}
	}
	return audio.NarrationRevision{}, audio.ErrNarrationNotFound
}
func (s *smokeAudioStore) GetLatestNarrationRevision(_ context.Context, chapterID string) (audio.NarrationRevision, error) {
	rs := s.narrations[chapterID]
	if len(rs) == 0 {
		return audio.NarrationRevision{}, audio.ErrNarrationNotFound
	}
	return rs[len(rs)-1], nil
}
func (s *smokeAudioStore) CreateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	if seg.ID == "" {
		seg.ID = uuid.NewString()
	}
	s.segments[seg.NarrationRevisionID] = append(s.segments[seg.NarrationRevisionID], seg)
	return seg, nil
}
func (s *smokeAudioStore) GetTTSSegment(_ context.Context, segmentID string) (audio.TTSSegment, error) {
	for _, segs := range s.segments {
		for _, seg := range segs {
			if seg.ID == segmentID {
				return seg, nil
			}
		}
	}
	return audio.TTSSegment{}, audio.ErrTTSSegmentNotFound
}
func (s *smokeAudioStore) UpdateTTSSegment(_ context.Context, seg audio.TTSSegment) (audio.TTSSegment, error) {
	for narID, segs := range s.segments {
		for i, item := range segs {
			if item.ID == seg.ID {
				s.segments[narID][i] = seg
				return seg, nil
			}
		}
	}
	return audio.TTSSegment{}, audio.ErrTTSSegmentNotFound
}
func (s *smokeAudioStore) NextAudioVersion(_ context.Context, chapterID string) (int, error) {
	s.nextVer[chapterID]++
	return s.nextVer[chapterID], nil
}
func (s *smokeAudioStore) CreateAudioAsset(_ context.Context, a audio.AudioAsset) (audio.AudioAsset, error) {
	if a.ID == "" {
		a.ID = uuid.NewString()
	}
	s.assets[a.ChapterID] = append(s.assets[a.ChapterID], a)
	return a, nil
}
func (s *smokeAudioStore) GetAudioAsset(_ context.Context, assetID string) (audio.AudioAsset, error) {
	for _, as := range s.assets {
		for _, a := range as {
			if a.ID == assetID {
				return a, nil
			}
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}
func (s *smokeAudioStore) GetActiveAudioAsset(_ context.Context, chapterID string) (audio.AudioAsset, error) {
	for _, a := range s.assets[chapterID] {
		if a.IsActive {
			return a, nil
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}
func (s *smokeAudioStore) GetLatestReadyAudioAssetForNarration(_ context.Context, chapterID, narrationRevisionID string) (audio.AudioAsset, error) {
	for _, a := range s.assets[chapterID] {
		if a.Status == "READY" && !a.IsActive && a.SourceNarrationRevisionID == narrationRevisionID {
			return a, nil
		}
	}
	return audio.AudioAsset{}, audio.ErrReadyAudioAssetNotFound
}
func (s *smokeAudioStore) SetActiveAudioAsset(_ context.Context, chapterID, assetID string) (audio.AudioAsset, error) {
	for i, a := range s.assets[chapterID] {
		a.IsActive = a.ID == assetID
		s.assets[chapterID][i] = a
		if a.IsActive {
			return a, nil
		}
	}
	return audio.AudioAsset{}, audio.ErrAudioAssetNotFound
}
func (s *smokeAudioStore) SetActiveAudioAssetForLatestNarration(ctx context.Context, chapterID, assetID string) (audio.AudioAsset, error) {
	latest, err := s.GetLatestNarrationRevision(ctx, chapterID)
	if err != nil {
		return audio.AudioAsset{}, err
	}
	asset, err := s.GetAudioAsset(ctx, assetID)
	if err != nil {
		return audio.AudioAsset{}, err
	}
	if asset.SourceNarrationRevisionID != latest.ID {
		return audio.AudioAsset{}, audio.ErrAudioAssetStaleForNarration
	}
	return s.SetActiveAudioAsset(ctx, chapterID, assetID)
}

type smokeObjectStorage struct{ objects map[string][]byte }

func newSmokeObjectStorage() *smokeObjectStorage {
	return &smokeObjectStorage{objects: map[string][]byte{}}
}

func (s *smokeObjectStorage) Put(_ context.Context, key string, data []byte) error {
	s.objects[key] = data
	return nil
}
func (s *smokeObjectStorage) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, errors.New("object not found")
	}
	return data, nil
}
func (s *smokeObjectStorage) DownloadToFile(_ context.Context, key, path string) error {
	data, err := s.Get(context.Background(), key)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}
func (s *smokeObjectStorage) UploadFile(ctx context.Context, key, path string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if err := s.Put(ctx, key, data); err != nil {
		return 0, err
	}
	return int64(len(data)), nil
}

type smokePresigner struct{}

func (smokePresigner) PresignedGetObject(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://cdn.smoke.test/" + key, nil
}

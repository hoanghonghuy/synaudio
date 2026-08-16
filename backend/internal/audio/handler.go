package audio

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) http.Handler {
	h := &Handler{svc: svc}

	r := chi.NewRouter()
	r.Post("/admin/chapters/{chapterID}/narration", h.createNarration)
	r.Post("/admin/chapters/{chapterID}/narration/{narrationID}/segments", h.createSegments)
	r.Post("/admin/chapters/{chapterID}/narration/{narrationID}/synthesize", h.synthesizeNarration)
	r.Post("/admin/tts-segments/{segmentID}/synthesize", h.synthesizeSegment)
	r.Post("/admin/chapters/{chapterID}/audio", h.createAudioAsset)
	r.Post("/admin/chapters/{chapterID}/audio/{assetID}/activate", h.activateAudioAsset)
	r.Get("/admin/chapters/{chapterID}/audio", h.getActiveAudioAsset)
	r.Get("/chapters/{chapterID}/audio-url", h.getAudioURL)
	return r
}

type createNarrationRequest struct {
	SourceContentRevisionID string `json:"source_content_revision_id"`
	VoiceID                 string `json:"voice_id"`
	Script                  string `json:"script"`
	CreatedBy               string `json:"created_by"`
}

func (h *Handler) createNarration(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req createNarrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	nar, err := h.svc.CreateNarrationRevision(r.Context(), chapterID, req.SourceContentRevisionID, req.VoiceID, req.Script, req.CreatedBy)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_NARRATION", "invalid narration")
		return
	}

	writeJSON(w, http.StatusCreated, nar)
}

func (h *Handler) createSegments(w http.ResponseWriter, r *http.Request) {
	narrationID := chi.URLParam(r, "narrationID")

	segments, err := h.svc.CreateTTSSegments(r.Context(), narrationID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			writeError(w, http.StatusNotFound, "NARRATION_NOT_FOUND", "narration revision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, segments)
}

func (h *Handler) synthesizeNarration(w http.ResponseWriter, r *http.Request) {
	narrationID := chi.URLParam(r, "narrationID")

	asset, err := h.svc.SynthesizeNarration(r.Context(), narrationID)
	if err != nil {
		if errors.Is(err, ErrNarrationNotFound) {
			writeError(w, http.StatusNotFound, "NARRATION_NOT_FOUND", "narration revision not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

func (h *Handler) synthesizeSegment(w http.ResponseWriter, r *http.Request) {
	segmentID := chi.URLParam(r, "segmentID")

	seg, err := h.svc.SynthesizeSegment(r.Context(), segmentID)
	if err != nil {
		if errors.Is(err, ErrTTSSegmentNotFound) {
			writeError(w, http.StatusNotFound, "TTS_SEGMENT_NOT_FOUND", "tts segment not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, seg)
}

type createAudioAssetRequest struct {
	SourceNarrationRevisionID string `json:"source_narration_revision_id"`
	StorageKey                string `json:"storage_key"`
	MimeType                  string `json:"mime_type"`
	SizeBytes                 int64  `json:"size_bytes"`
	DurationMs                int    `json:"duration_ms"`
	BitrateKbps               int    `json:"bitrate_kbps"`
}

func (h *Handler) createAudioAsset(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	var req createAudioAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	asset, err := h.svc.CreateAudioAsset(r.Context(), chapterID, req.SourceNarrationRevisionID, req.StorageKey, req.MimeType, req.SizeBytes, req.DurationMs, req.BitrateKbps)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, asset)
}

func (h *Handler) activateAudioAsset(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")
	assetID := chi.URLParam(r, "assetID")

	asset, err := h.svc.ActivateAudioAsset(r.Context(), chapterID, assetID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			writeError(w, http.StatusNotFound, "AUDIO_ASSET_NOT_FOUND", "audio asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *Handler) getActiveAudioAsset(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	asset, err := h.svc.GetActiveAudioAsset(r.Context(), chapterID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			writeError(w, http.StatusNotFound, "AUDIO_ASSET_NOT_FOUND", "audio asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, asset)
}

func (h *Handler) getAudioURL(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")

	url, err := h.svc.GetAudioURL(r.Context(), chapterID)
	if err != nil {
		if errors.Is(err, ErrAudioAssetNotFound) {
			writeError(w, http.StatusNotFound, "AUDIO_ASSET_NOT_FOUND", "audio asset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

package audio

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) getAdminAudioPreviewURL(w http.ResponseWriter, r *http.Request) {
	chapterID := chi.URLParam(r, "chapterID")
	assetID := chi.URLParam(r, "assetID")

	url, err := h.svc.GetAdminAudioPreviewURL(r.Context(), chapterID, assetID)
	if err != nil {
		switch {
		case errors.Is(err, ErrAudioAssetNotFound):
			writeError(w, http.StatusNotFound, "AUDIO_ASSET_NOT_FOUND", "audio asset not found")
		case errors.Is(err, ErrAdminAudioPreviewNotEligible):
			writeError(w, http.StatusConflict, "AUDIO_PREVIEW_NOT_READY", "audio asset is not ready for preview")
		case errors.Is(err, ErrAudioPresignerRequired):
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL", "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

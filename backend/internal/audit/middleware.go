package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type RecordFunc func(context.Context, Event) (Event, error)
type ActorResolver func(context.Context, *http.Request) (string, error)

type routeDescriptor struct {
	Action       string
	ResourceType string
	ResourceParam string
}

// WrapRoute audits one mutating route. The handler response is buffered until
// the append-only audit write has been attempted. An audit-store failure is
// surfaced as 503 rather than silently returning a successful unaudited write.
func WrapRoute(next http.Handler, method, route string, record RecordFunc, resolveActor ActorResolver) http.Handler {
	if record == nil || !isMutation(method) {
		return next
	}
	desc := describeRoute(method, route)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actorID := ""
		if resolveActor != nil {
			actorID, _ = resolveActor(r.Context(), r)
		}

		buffer := newBufferedResponse()
		next.ServeHTTP(buffer, r)
		status := buffer.status
		if status == 0 {
			status = http.StatusOK
		}

		event := eventForResponse(r, method, route, desc, actorID, status, buffer.body.Bytes())
		if _, err := record(r.Context(), event); err != nil {
			writeAuditUnavailable(w)
			return
		}
		buffer.flushTo(w)
	})
}

// WrapAuth audits mutating routes mounted beneath /api/v1/auth. It deliberately
// never inspects request bodies, so passwords, refresh credentials and MFA
// secrets cannot enter audit metadata through this boundary.
func WrapAuth(next http.Handler, record RecordFunc, resolveActor ActorResolver) http.Handler {
	if record == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isMutation(r.Method) {
			next.ServeHTTP(w, r)
			return
		}
		route := authRoute(r.URL.Path)
		desc := describeAuthRoute(r.Method, route)
		actorID := ""
		if resolveActor != nil {
			actorID, _ = resolveActor(r.Context(), r)
		}
		buffer := newBufferedResponse()
		next.ServeHTTP(buffer, r)
		status := buffer.status
		if status == 0 {
			status = http.StatusOK
		}
		event := eventForResponse(r, r.Method, route, desc, actorID, status, buffer.body.Bytes())
		if _, err := record(r.Context(), event); err != nil {
			writeAuditUnavailable(w)
			return
		}
		buffer.flushTo(w)
	})
}

func eventForResponse(r *http.Request, method, route string, desc routeDescriptor, actorID string, status int, body []byte) Event {
	actorType := ActorAnonymous
	if actorID != "" {
		actorType = ActorUser
	}
	result := ResultSucceeded
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		result = ResultDenied
	} else if status >= 400 {
		result = ResultFailed
	}
	requestID := chimiddleware.GetReqID(r.Context())
	correlationID := strings.TrimSpace(r.Header.Get("X-Correlation-ID"))
	if correlationID == "" {
		correlationID = requestID
	}

	resourceID := routeParam(r, desc.ResourceParam)
	storyID := routeParam(r, "storyID")
	chapterID := routeParam(r, "chapterID")
	runID := routeParam(r, "runID")
	responseRefs := extractResponseReferences(body)
	if resourceID == "" {
		resourceID = firstString(responseRefs, "id", "revision_id", "asset_id", "run_id")
	}
	if storyID == "" {
		storyID = firstString(responseRefs, "story_id")
	}
	if chapterID == "" {
		chapterID = firstString(responseRefs, "chapter_id", "target_chapter_id")
	}
	if runID == "" {
		runID = firstString(responseRefs, "generation_run_id")
		if desc.ResourceType == "GENERATION_RUN" {
			runID = resourceID
		}
	}
	if desc.ResourceType == "STORY" && storyID == "" {
		storyID = resourceID
	}
	if desc.ResourceType == "CHAPTER" && chapterID == "" {
		chapterID = resourceID
	}

	provenance := map[string]any{}
	for _, key := range []string{"generation_run_id", "plan_revision_id", "content_revision_id", "source_content_revision_id", "canon_version_id", "context_snapshot_id", "audio_asset_id", "narration_revision_id", "provider", "model"} {
		if value, ok := responseRefs[key]; ok {
			provenance[key] = value
		}
	}

	return Event{
		ActorUserID:     actorID,
		ActorType:       actorType,
		Action:          desc.Action,
		ResourceType:    desc.ResourceType,
		ResourceID:      resourceID,
		StoryID:         storyID,
		ChapterID:       chapterID,
		Result:          result,
		CorrelationID:   correlationID,
		RequestID:       requestID,
		GenerationRunID: runID,
		Provenance:      provenance,
		Metadata: map[string]any{
			"http_method": method,
			"route":       route,
			"status":      status,
		},
	}
}

func describeRoute(method, route string) routeDescriptor {
	key := strings.ToUpper(method) + " " + route
	if desc, ok := semanticRoutes[key]; ok {
		return desc
	}
	return routeDescriptor{Action: "ADMIN_MUTATION", ResourceType: "ADMIN_RESOURCE"}
}

func describeAuthRoute(method, route string) routeDescriptor {
	key := strings.ToUpper(method) + " " + route
	if desc, ok := authRoutes[key]; ok {
		return desc
	}
	return routeDescriptor{Action: "SECURITY_MUTATION", ResourceType: "SECURITY"}
}

var semanticRoutes = map[string]routeDescriptor{
	"POST /admin/stories":                                      {"STORY_CREATED", "STORY", ""},
	"PUT /admin/stories/{storyID}/workflow-settings":           {"STORY_WORKFLOW_SETTINGS_CHANGED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/content-profile":            {"STORY_CONTENT_PROFILE_CHANGED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/activate":                   {"STORY_ACTIVATED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/archive":                    {"STORY_ARCHIVED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/restore":                    {"STORY_RESTORED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/make-public":                {"STORY_MADE_PUBLIC", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/make-private":               {"STORY_MADE_PRIVATE", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/cover":                      {"STORY_COVER_CHANGED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/foundation":                 {"STORY_FOUNDATION_GENERATED", "STORY", "storyID"},
	"POST /admin/stories/{storyID}/chapters":                   {"CHAPTER_CREATED", "CHAPTER", ""},
	"POST /admin/chapters/{chapterID}/plans":                   {"CHAPTER_PLAN_CHANGED", "CHAPTER", "chapterID"},
	"POST /admin/canon-branches/{branchID}/commit":              {"CANON_COMMITTED", "CANON_BRANCH", "branchID"},
	"POST /admin/canon-versions/{versionID}/promote":            {"CANON_VERSION_PROMOTED", "CANON_VERSION", "versionID"},
	"POST /admin/stories/{storyID}/canon-repair":               {"CANON_REPAIRED", "STORY", "storyID"},
	"POST /admin/chapters/{chapterID}/publish":                 {"CHAPTER_PUBLISHED", "CHAPTER", "chapterID"},
	"POST /admin/chapters/{chapterID}/unpublish":               {"CHAPTER_UNPUBLISHED", "CHAPTER", "chapterID"},
	"POST /admin/stories/{storyID}/creative-decisions":         {"CREATIVE_DECISION_PROPOSED", "CREATIVE_DECISION", ""},
	"POST /admin/creative-decisions/{decisionID}/select":       {"CREATIVE_DECISION_APPROVED", "CREATIVE_DECISION", "decisionID"},
	"POST /admin/creative-decisions/{decisionID}/reject":       {"CREATIVE_DECISION_REJECTED", "CREATIVE_DECISION", "decisionID"},
	"POST /admin/chapters/{chapterID}/content":                 {"CHAPTER_GENERATED", "CONTENT_REVISION", ""},
	"POST /admin/chapters/{chapterID}/approve":                 {"CHAPTER_APPROVED", "CHAPTER", "chapterID"},
	"POST /admin/chapters/{chapterID}/edit":                    {"CHAPTER_CONTENT_EDITED", "CONTENT_REVISION", ""},
	"POST /admin/chapters/{chapterID}/regenerate":              {"CHAPTER_REGENERATED", "CONTENT_REVISION", ""},
	"POST /admin/revisions/{revisionID}/reject":                {"CHAPTER_CONTENT_REJECTED", "CONTENT_REVISION", "revisionID"},
	"POST /admin/runs":                                         {"GENERATION_RUN_CREATED", "GENERATION_RUN", ""},
	"POST /admin/runs/{runID}/jobs":                            {"GENERATION_JOB_CREATED", "GENERATION_RUN", "runID"},
	"POST /admin/stories/{storyID}/batch-generate":             {"BATCH_GENERATION_STARTED", "STORY", "storyID"},
	"POST /admin/runs/{runID}/mark-stale":                      {"GENERATION_DOWNSTREAM_MARKED_STALE", "GENERATION_RUN", "runID"},
	"POST /admin/retcons":                                      {"RETCON_REQUESTED", "RETCON", ""},
	"POST /admin/retcons/{id}/approve":                         {"RETCON_APPROVED", "RETCON", "id"},
	"POST /admin/retcons/{id}/cancel":                          {"RETCON_CANCELLED", "RETCON", "id"},
	"POST /admin/retcons/{id}/analyze":                         {"RETCON_ANALYZED", "RETCON", "id"},
	"POST /admin/retcons/{id}/ready":                           {"RETCON_READY", "RETCON", "id"},
	"POST /admin/retcons/{id}/apply":                           {"RETCON_APPLIED", "RETCON", "id"},
	"POST /admin/chapters/{chapterID}/narration":               {"NARRATION_CREATED", "NARRATION_REVISION", ""},
	"POST /admin/chapters/{chapterID}/narration/{narrationID}/synthesize": {"AUDIO_GENERATED", "NARRATION_REVISION", "narrationID"},
	"POST /admin/tts-segments/{segmentID}/synthesize":           {"TTS_SEGMENT_GENERATED", "TTS_SEGMENT", "segmentID"},
	"POST /admin/chapters/{chapterID}/audio":                   {"AUDIO_ASSET_CREATED", "AUDIO_ASSET", ""},
	"POST /admin/chapters/{chapterID}/audio/{assetID}/activate": {"AUDIO_ACTIVATED", "AUDIO_ASSET", "assetID"},
	"POST /admin/chapters/{chapterID}/revision-impact":         {"LISTENER_REVISION_IMPACT_APPLIED", "CHAPTER", "chapterID"},
}

var authRoutes = map[string]routeDescriptor{
	"POST /register":                    {"USER_REGISTERED", "USER", ""},
	"POST /login":                       {"AUTH_LOGIN", "AUTH_SESSION", ""},
	"POST /logout":                      {"AUTH_LOGOUT", "AUTH_SESSION", ""},
	"POST /logout-all":                  {"AUTH_LOGOUT_ALL", "USER", ""},
	"POST /refresh":                     {"AUTH_REFRESH", "AUTH_SESSION", ""},
	"DELETE /sessions/{sessionID}":      {"AUTH_SESSION_REVOKED", "AUTH_SESSION", "sessionID"},
	"POST /email/verify":                {"EMAIL_VERIFIED", "USER", ""},
	"POST /password/reset":              {"PASSWORD_RESET", "USER", ""},
	"POST /mfa/totp/setup":              {"MFA_SETUP_STARTED", "USER", ""},
	"POST /mfa/totp/confirm":            {"MFA_ENABLED", "USER", ""},
	"POST /mfa/totp/disable":            {"MFA_DISABLED", "USER", ""},
	"POST /account/deletion/request":    {"ACCOUNT_DELETION_REQUESTED", "USER", ""},
	"POST /account/deletion/cancel":     {"ACCOUNT_DELETION_CANCELLED", "USER", ""},
}

func routeParam(r *http.Request, name string) string {
	if name == "" {
		return ""
	}
	return chi.URLParam(r, name)
}

func authRoute(path string) string {
	const prefix = "/api/v1/auth"
	if strings.HasPrefix(path, prefix) {
		path = strings.TrimPrefix(path, prefix)
	}
	if strings.HasPrefix(path, "/sessions/") {
		return "/sessions/{sessionID}"
	}
	if path == "" {
		return "/"
	}
	return path
}

func extractResponseReferences(body []byte) map[string]any {
	if len(body) == 0 || len(body) > 1<<20 {
		return map[string]any{}
	}
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil {
		return map[string]any{}
	}
	return value
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isMutation(method string) bool {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

type bufferedResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newBufferedResponse() *bufferedResponse {
	return &bufferedResponse{header: make(http.Header)}
}

func (w *bufferedResponse) Header() http.Header { return w.header }
func (w *bufferedResponse) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *bufferedResponse) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(body)
}
func (w *bufferedResponse) flushTo(dst http.ResponseWriter) {
	for key, values := range w.header {
		for _, value := range values {
			dst.Header().Add(key, value)
		}
	}
	status := w.status
	if status == 0 {
		status = http.StatusOK
	}
	dst.WriteHeader(status)
	_, _ = dst.Write(w.body.Bytes())
}

func writeAuditUnavailable(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Synaudio-Audit-Status", "unavailable")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte(`{"error":{"code":"AUDIT_UNAVAILABLE","message":"mutation completed with unavailable audit persistence; verify state before retrying"}}`))
}

package openapi

import (
	"fmt"
	"os"
	"strings"

	"go.yaml.in/yaml/v3"
)

// OperationBinding declares the executable request/response contract for one operation.
type OperationBinding struct {
	Request       string // components/schemas name; empty when no JSON body
	Response      string // components/schemas name for success JSON body
	SuccessStatus int    // defaults to 200
	Description   string // success response description
	Multipart     bool   // request is multipart/form-data (e.g. cover upload)
}

// OperationBindings is the authoritative map from operationId to schema bindings.
// contract_test.go enforces api.yaml matches these bindings byte-for-byte on enriched fields.
var OperationBindings = map[string]OperationBinding{
	// Platform
	"getHealth":    {Response: "HealthResponse", Description: "Liveness probe"},
	"getReady":     {Response: "ReadyResponse", Description: "Readiness probe with dependency status"},
	// Auth
	"postApiV1AuthRegister":              {Request: "RegisterRequest", Response: "UserResponse", SuccessStatus: 201, Description: "Registered user"},
	"postApiV1AuthLogin":               {Request: "LoginRequest", Response: "TokenResponse", Description: "Authenticated session tokens"},
	"getApiV1AuthMe":                     {Response: "MeResponse", Description: "Current authenticated principal"},
	"postApiV1AuthLogout":                {Response: "StatusResponse", Description: "Current session revoked"},
	"postApiV1AuthLogoutAll":             {Response: "StatusResponse", Description: "All sessions revoked"},
	"postApiV1AuthRefresh":               {Response: "TokenResponse", Description: "Access token refreshed"},
	"getApiV1AuthSessions":               {Response: "SessionListResponse", Description: "Active refresh sessions"},
	"deleteApiV1AuthSessionsSessionid":   {Response: "StatusResponse", Description: "Session revoked"},
	"postApiV1AuthEmailVerify":           {Request: "EmailVerifyRequest", Response: "StatusResponse", Description: "Email verified"},
	"postApiV1AuthEmailResend":           {Request: "EmailResendRequest", Response: "StatusResponse", Description: "Verification email accepted for delivery"},
	"postApiV1AuthPasswordForgot":        {Request: "PasswordForgotRequest", Response: "StatusResponse", Description: "Password reset accepted for delivery"},
	"postApiV1AuthPasswordReset":         {Request: "PasswordResetRequest", Response: "StatusResponse", Description: "Password reset completed"},
	"postApiV1AuthMfaTotpSetup":          {Response: "MFASetupResponse", Description: "TOTP enrollment secret"},
	"postApiV1AuthMfaTotpConfirm":        {Request: "MFAConfirmRequest", Response: "MFAConfirmResponse", Description: "TOTP confirmed with recovery codes"},
	"postApiV1AuthMfaTotpDisable":        {Response: "StatusResponse", Description: "TOTP disabled"},
	"postApiV1AuthAccountDeletionRequest":  {Response: "StatusResponse", Description: "Account deletion scheduled"},
	"postApiV1AuthAccountDeletionCancel":   {Response: "StatusResponse", Description: "Account deletion cancelled"},
	// Public catalog
	"getApiV1Stories":              {Response: "StoryListResponse", Description: "Public story catalog"},
	"getApiV1StoriesStoryid":       {Response: "StorySummary", Description: "Public story detail"},
	"getApiV1StoriesStoryidChapters": {Response: "PublishedChapterListResponse", Description: "Published chapters for listeners"},
	"getApiV1Genres":               {Response: "GenreListResponse", Description: "Story genres"},
	"getApiV1ChaptersChapteridContent": {Response: "PublishedChapterContentResponse", Description: "Published chapter prose"},
	"getApiV1ChaptersChapteridAudioUrl": {Response: "AudioURLResponse", Description: "Presigned audio playback URL"},
	// Listener
	"getApiV1MeLibrary":                    {Response: "LibraryResponse", Description: "Authenticated listener dashboard"},
	"getApiV1MeFavorites":                  {Response: "FavoriteListResponse", Description: "Favorited stories"},
	"putApiV1MeFavoritesStoryid":           {Response: "FavoriteStatusResponse", Description: "Story favorited"},
	"deleteApiV1MeFavoritesStoryid":        {Response: "FavoriteStatusResponse", Description: "Story unfavorited"},
	"getApiV1MeProgressChapterid":          {Response: "ListeningProgressPascal", Description: "Playback progress for chapter"},
	"putApiV1MeProgressChapterid":          {Request: "SaveProgressRequest", Response: "ListeningProgressPascal", Description: "Playback progress saved"},
	"postApiV1MeProgressChapteridComplete": {Response: "ListeningProgressPascal", Description: "Chapter marked completed"},
	// Story admin
	"postApiV1AdminStories":                        {Request: "CreateStoryRequest", Response: "StorySummary", SuccessStatus: 201, Description: "Story created"},
	"getApiV1AdminStories":                         {Response: "StoryListResponse", Description: "Admin story list"},
	"getApiV1AdminStoriesStoryidActivationReadiness": {Response: "ActivationReadinessResponse", Description: "Story activation readiness gates"},
	"postApiV1AdminStoriesStoryidActivate":           {Response: "StorySummary", Description: "Story activated"},
	"postApiV1AdminStoriesStoryidArchive":            {Response: "StorySummary", Description: "Story archived"},
	"postApiV1AdminStoriesStoryidRestore":            {Response: "StorySummary", Description: "Story restored from archive"},
	"postApiV1AdminStoriesStoryidMakePublic":         {Response: "StorySummary", Description: "Story made public"},
	"postApiV1AdminStoriesStoryidMakePrivate":        {Response: "StorySummary", Description: "Story made private"},
	"getApiV1AdminStoriesStoryidWorkflowSettings":    {Response: "WorkflowSettingsResponse", Description: "Story workflow settings"},
	"putApiV1AdminStoriesStoryidWorkflowSettings":    {Request: "WorkflowSettingsRequest", Response: "WorkflowSettingsResponse", Description: "Story workflow settings updated"},
	"postApiV1AdminStoriesStoryidContentProfile":     {Request: "ContentProfileRequest", Response: "ContentProfileResponse", SuccessStatus: 201, Description: "Content profile version created"},
	"getApiV1AdminStoriesStoryidContentProfile":      {Response: "ContentProfileResponse", Description: "Active content profile"},
	"postApiV1AdminStoriesStoryidCover":              {Response: "StoryAssetResponse", SuccessStatus: 201, Description: "Cover asset uploaded", Multipart: true},
	"getApiV1AdminStoriesStoryidGenerationPolicy":    {Response: "GenerationPolicyResponse", Description: "Story generation policy"},
	"putApiV1AdminStoriesStoryidMetadata":            {Request: "UpdateMetadataRequest", Response: "StorySummary", Description: "Story metadata updated"},
	// Planning
	"postApiV1AdminStoriesStoryidFoundation":       {Request: "FoundationRequest", Response: "FoundationResponse", Description: "Story foundation generated"},
	"getApiV1AdminStoriesStoryidBible":             {Response: "StoryBibleVersionPascal", Description: "Active story bible version"},
	"postApiV1AdminStoriesStoryidBibleVersions":    {Request: "VersionContentRequest", Response: "StoryBibleVersionPascal", SuccessStatus: 201, Description: "Story bible version created"},
	"getApiV1AdminStoriesStoryidEnding":            {Response: "EndingPlanVersionPascal", Description: "Active ending plan version"},
	"postApiV1AdminStoriesStoryidEndingVersions":   {Request: "VersionContentRequest", Response: "EndingPlanVersionPascal", SuccessStatus: 201, Description: "Ending plan version created"},
	"getApiV1AdminStoriesStoryidArcs":              {Response: "StoryArcListResponse", Description: "Story arcs"},
	"postApiV1AdminStoriesStoryidArcs":              {Request: "CreateArcRequest", Response: "StoryArcPascal", SuccessStatus: 201, Description: "Story arc created"},
	"getApiV1AdminStoriesStoryidCharacters":        {Response: "CharacterListResponse", Description: "Story characters"},
	"postApiV1AdminStoriesStoryidCharacters":       {Request: "CreateCharacterRequest", Response: "CharacterPascal", SuccessStatus: 201, Description: "Character created"},
	"postApiV1AdminStoriesStoryidChapters":         {Request: "CreateChapterRequest", Response: "ChapterPascal", SuccessStatus: 201, Description: "Chapter created"},
	"getApiV1AdminStoriesStoryidChapters":          {Response: "ChapterListResponse", Description: "Story chapters"},
	"postApiV1AdminChaptersChapteridPlans":        {Request: "CreatePlanRequest", Response: "ChapterPlanRevisionPascal", SuccessStatus: 201, Description: "Chapter plan revision created"},
	"postApiV1AdminStoriesStoryidFacts":            {Request: "CreateFactRequest", Response: "StoryFactPascal", SuccessStatus: 201, Description: "Story fact created"},
	"getApiV1AdminStoriesStoryidFacts":             {Response: "StoryFactListResponse", Description: "Story facts"},
	"postApiV1AdminStoriesStoryidPlotThreads":      {Request: "CreatePlotThreadRequest", Response: "PlotThreadPascal", SuccessStatus: 201, Description: "Plot thread created"},
	"getApiV1AdminStoriesStoryidPlotThreads":       {Response: "PlotThreadListResponse", Description: "Plot threads"},
	"postApiV1AdminPlotThreadsThreadidEvents":      {Request: "CreatePlotThreadEventRequest", Response: "PlotThreadEventPascal", SuccessStatus: 201, Description: "Plot thread event recorded"},
	"postApiV1AdminStoriesStoryidCanonBranches":    {Request: "CreateCanonBranchRequest", Response: "CanonBranchPascal", SuccessStatus: 201, Description: "Canon branch created"},
	"postApiV1AdminCanonBranchesBranchidVersions": {Request: "CreateCanonVersionRequest", Response: "CanonVersionPascal", SuccessStatus: 201, Description: "Canon version created"},
	"getApiV1AdminCanonBranchesBranchidVersions":  {Response: "CanonVersionListResponse", Description: "Canon versions"},
	"postApiV1AdminCanonBranchesBranchidProvisionalVersions": {Request: "CreateProvisionalCanonVersionRequest", Response: "CanonVersionPascal", SuccessStatus: 201, Description: "Provisional canon version created"},
	"postApiV1AdminCanonVersionsVersionidPromote": {Request: "PromoteProvisionalVersionRequest", Response: "CanonVersionPascal", Description: "Provisional canon version promoted"},
	"postApiV1AdminStoriesStoryidContextSnapshots": {Request: "CreateContextSnapshotRequest", Response: "ContextSnapshotPascal", SuccessStatus: 201, Description: "Context snapshot created"},
	"getApiV1AdminStoriesStoryidContextSnapshots":  {Response: "ContextSnapshotListResponse", Description: "Context snapshots"},
	"getApiV1AdminContextSnapshotsSnapshotid":      {Response: "ContextSnapshotPascal", Description: "Context snapshot detail"},
	"postApiV1AdminCanonBranchesBranchidCommit":   {Request: "CommitCanonRequest", Response: "CanonCommitResultPascal", Description: "Canon committed"},
	"postApiV1AdminStoriesStoryidCanonRepair":     {Request: "RepairCanonRequest", Response: "CanonCommitResultPascal", Description: "Canon repaired"},
	"getApiV1AdminChaptersChapteridPublishReadiness": {Response: "ReadinessResponse", Description: "Chapter publish readiness"},
	"postApiV1AdminChaptersChapteridReady":         {Response: "ChapterPascal", Description: "Chapter marked ready"},
	"postApiV1AdminChaptersChapteridPublish":       {Response: "ChapterPascal", Description: "Chapter published"},
	"postApiV1AdminChaptersChapteridUnpublish":     {Response: "ChapterPascal", Description: "Chapter unpublished"},
	"getApiV1AdminStoriesStoryidCreativeDecisions": {Response: "CreativeDecisionListResponse", Description: "Creative decisions"},
	"postApiV1AdminStoriesStoryidCreativeDecisions": {Request: "CreateCreativeDecisionRequest", Response: "CreativeDecisionPascal", SuccessStatus: 201, Description: "Creative decision proposed"},
	"postApiV1AdminCreativeDecisionsDecisionidSelect": {Request: "SelectCreativeDecisionRequest", Response: "CreativeDecisionPascal", Description: "Creative decision selected"},
	"postApiV1AdminCreativeDecisionsDecisionidReject": {Request: "RejectCreativeDecisionRequest", Response: "CreativeDecisionPascal", Description: "Creative decision rejected"},
	"getApiV1AdminStoriesStoryidArcsArcidCompletion": {Response: "ArcCompletionResultPascal", Description: "Arc completion review"},
	"getApiV1AdminStoriesStoryidAttention":         {Response: "AttentionItemListResponse", Description: "Attention items"},
	"postApiV1AdminStoriesStoryidAttention":        {Request: "CreateAttentionItemRequest", Response: "AttentionItemPascal", SuccessStatus: 201, Description: "Attention item created"},
	"postApiV1AdminAttentionItemidResolve":       {Response: "AttentionItemPascal", Description: "Attention item resolved"},
	"getApiV1AdminStoriesStoryidThreadInactivity": {Response: "ThreadInactivityListResponse", Description: "Inactive plot threads"},
	// Generation
	"postApiV1AdminChaptersChapteridContent":       {Request: "WriteChapterRequest", Response: "ContentRevisionPascal", SuccessStatus: 201, Description: "Chapter content written"},
	"getApiV1AdminChaptersChapteridContent":      {Response: "AdminChapterContentResponse", Description: "Chapter content revisions with generation state"},
	"postApiV1AdminChaptersChapteridApprove":     {Request: "ApproveContentRequest", Response: "ContentApprovalPascal", Description: "Content revision approved"},
	"postApiV1AdminChaptersChapteridReviews":     {Request: "CreateReviewRequest", Response: "ChapterReviewPascal", SuccessStatus: 201, Description: "Chapter review recorded"},
	"getApiV1AdminChaptersChapteridReviews":      {Response: "ChapterReviewListResponse", Description: "Chapter reviews"},
	"postApiV1AdminChaptersChapteridDuration":    {Request: "AnalyzeDurationRequest", Response: "DurationOutputPascal", Description: "Duration analysis"},
	"postApiV1AdminChaptersChapteridContinuity":  {Request: "RunReviewRequest", Response: "ChapterReviewPascal", Description: "Continuity review"},
	"postApiV1AdminChaptersChapteridQuality":     {Request: "RunReviewRequest", Response: "ChapterReviewPascal", Description: "Quality review"},
	"postApiV1AdminChaptersChapteridSafety":      {Request: "RunReviewRequest", Response: "ChapterReviewPascal", Description: "Safety review"},
	"postApiV1AdminChaptersChapteridRewrite":     {Request: "RewriteChapterRequest", Response: "ContentRevisionPascal", SuccessStatus: 201, Description: "Chapter rewritten"},
	"postApiV1AdminChaptersChapteridEdit":        {Request: "EditContentRequest", Response: "ContentRevisionPascal", SuccessStatus: 201, Description: "Chapter edited"},
	"postApiV1AdminChaptersChapteridRegenerate":  {Request: "RegenerateContentRequest", Response: "ContentRevisionPascal", SuccessStatus: 201, Description: "Chapter regenerated"},
	"postApiV1AdminRevisionsRevisionidReject":    {Request: "RejectContentRequest", Response: "ContentRevisionPascal", Description: "Content revision rejected"},
	"postApiV1AdminRuns":                         {Request: "CreateRunRequest", Response: "GenerationRunPascal", SuccessStatus: 201, Description: "Generation run created"},
	"getApiV1AdminRunsRunid":                     {Response: "GenerationRunPascal", Description: "Generation run detail"},
	"getApiV1AdminChaptersChapteridGenerationRunLatest": {Response: "GenerationRunPascal", Description: "Latest chapter generation run"},
	"postApiV1AdminRunsRunidJobs":                {Request: "CreateJobRequest", Response: "GenerationJobPascal", SuccessStatus: 201, Description: "Generation job created"},
	"getApiV1AdminGenerationJobsJobid":           {Response: "JobViewPascal", Description: "Generation job observation"},
	"postApiV1AdminGenerationJobsJobidRetry":      {Response: "JobViewPascal", Description: "Generation job requeued"},
	"postApiV1AdminStoriesStoryidBatchGenerate":  {Request: "StartBatchRequest", Response: "GenerationRunPascal", SuccessStatus: 201, Description: "Batch generation started"},
	"postApiV1AdminRunsRunidMarkStale":           {Request: "MarkStaleRequest", Response: "StaleJobsResponse", Description: "Downstream jobs marked stale"},
	"postApiV1AdminAttemptsAttemptidUsage":       {Request: "RecordUsageRequest", Response: "JobAttemptPascal", Description: "Provider usage recorded"},
	"getApiV1AdminStoriesStoryidUsage":           {Response: "JobAttemptListResponse", Description: "Story provider usage"},
	// Audio
	"postApiV1AdminChaptersChapteridNarration": {Request: "CreateNarrationRequest", Response: "NarrationRevisionPascal", SuccessStatus: 201, Description: "Narration revision created"},
	"getApiV1AdminChaptersChapteridNarrationLatest": {Response: "NarrationRevisionPascal", Description: "Latest narration revision"},
	"postApiV1AdminChaptersChapteridNarrationNarrationidSegments": {Response: "TTSSegmentListResponse", SuccessStatus: 201, Description: "TTS segments created"},
	"postApiV1AdminChaptersChapteridNarrationNarrationidSynthesize": {Response: "AudioAssetPascal", Description: "Narration synthesized to audio asset"},
	"postApiV1AdminTtsSegmentsSegmentidSynthesize": {Response: "TTSSegmentPascal", Description: "TTS segment synthesized"},
	"postApiV1AdminChaptersChapteridAudio": {Request: "CreateAudioAssetRequest", Response: "AudioAssetPascal", SuccessStatus: 201, Description: "Audio asset created"},
	"getApiV1AdminChaptersChapteridAudio":    {Response: "AudioAssetPascal", Description: "Active audio asset"},
	"getApiV1AdminChaptersChapteridNarrationNarrationidAudioLatestReady": {Response: "AudioAssetPascal", Description: "Latest ready audio asset for narration"},
	"postApiV1AdminChaptersChapteridAudioAssetidActivate": {Response: "AudioAssetPascal", Description: "Audio asset activated"},
	// Listener admin
	"postApiV1AdminChaptersChapteridRevisionImpact": {Request: "RevisionImpactRequest", Response: "RevisionImpactResponse", Description: "Revision relisten impact applied"},
	// Retcon
	"postApiV1AdminRetcons":            {Request: "CreateRetconRequest", Response: "RetconRequestPascal", SuccessStatus: 201, Description: "Retcon proposed"},
	"getApiV1AdminRetcons":             {Response: "RetconListResponse", Description: "Retcon requests"},
	"getApiV1AdminRetconsId":           {Response: "RetconRequestPascal", Description: "Retcon detail"},
	"postApiV1AdminRetconsIdApprove":   {Request: "ApproveRetconRequest", Response: "RetconRequestPascal", Description: "Retcon approved"},
	"postApiV1AdminRetconsIdCancel":    {Response: "RetconRequestPascal", Description: "Retcon cancelled"},
	"postApiV1AdminRetconsIdAnalyze":   {Response: "RetconRequestPascal", Description: "Retcon impact analyzed"},
	"postApiV1AdminRetconsIdReady":     {Response: "RetconRequestPascal", Description: "Retcon marked ready to apply"},
	"postApiV1AdminRetconsIdApply":     {Request: "ApplyRetconRequest", Response: "RetconRequestPascal", Description: "Retcon applied"},
	// Audit
	"getApiV1AdminAudit":         {Response: "AuditEventListResponse", Description: "Audit events"},
	"getApiV1AdminAuditEventid":  {Response: "AuditEvent", Description: "Audit event detail"},
}

// Document is the OpenAPI contract root structure enriched by this package.
type Document struct {
	OpenAPI    string                      `yaml:"openapi"`
	Info       map[string]any              `yaml:"info"`
	Servers    []map[string]any            `yaml:"servers"`
	Paths      map[string]map[string]any   `yaml:"paths"`
	Components map[string]any              `yaml:"components"`
}

func LoadDocument(path string) (*Document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc Document
	if err := yaml.Unmarshal(body, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func SaveDocument(path string, doc *Document) error {
	body, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o644)
}

func loadYAMLFile(path string) (map[string]any, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(body, &root); err != nil {
		return nil, err
	}
	return root, nil
}

func mergeComponentSchemas(doc *Document, schemas map[string]any) {
	if doc.Components == nil {
		doc.Components = map[string]any{}
	}
	existing, _ := doc.Components["schemas"].(map[string]any)
	if existing == nil {
		existing = map[string]any{}
	}
	for name, schema := range schemas {
		existing[name] = schema
	}
	doc.Components["schemas"] = existing
}

func enrichOperation(op map[string]any, method string, binding OperationBinding) {
	if binding.Multipart {
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"multipart/form-data": map[string]any{
					"schema": map[string]any{
						"type":     "object",
						"required": []any{"file"},
						"properties": map[string]any{
							"file": map[string]any{"type": "string", "format": "binary"},
						},
					},
				},
			},
		}
	} else if binding.Request != "" {
		op["requestBody"] = map[string]any{
			"required": true,
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": map[string]any{"$ref": "#/components/schemas/" + binding.Request},
				},
			},
		}
	}
	status := binding.SuccessStatus
	if status == 0 {
		status = 200
	}
	statusKey := fmt.Sprintf("%d", status)
	responses, _ := op["responses"].(map[string]any)
	if responses == nil {
		responses = map[string]any{}
		op["responses"] = responses
	}
	delete(responses, "200")
	delete(responses, "201")
	responses[statusKey] = map[string]any{
		"description": binding.Description,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{"$ref": "#/components/schemas/" + binding.Response},
			},
		},
	}
}

func EnrichDocument(doc *Document) error {
	schemasRoot, err := loadYAMLFile("schemas.yaml")
	if err != nil {
		return fmt.Errorf("load schemas.yaml: %w", err)
	}
	components, _ := schemasRoot["components"].(map[string]any)
	schemaMap, _ := components["schemas"].(map[string]any)
	if schemaMap == nil {
		return fmt.Errorf("schemas.yaml missing components.schemas")
	}
	mergeComponentSchemas(doc, schemaMap)

	for path, item := range doc.Paths {
		for method, raw := range item {
			if !isHTTPMethod(method) {
				continue
			}
			op, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			id, _ := op["operationId"].(string)
			binding, ok := OperationBindings[id]
			if !ok {
				return fmt.Errorf("missing OperationBinding for %s %s (%s)", strings.ToUpper(method), path, id)
			}
			enrichOperation(op, method, binding)
		}
	}
	return nil
}

func isHTTPMethod(method string) bool {
	switch strings.ToLower(method) {
	case "get", "post", "put", "delete", "patch":
		return true
	default:
		return false
	}
}

// ValidateOperationSchemas ensures every bound operation declares concrete JSON schemas.
func ValidateOperationSchemas(doc *Document) error {
	for path, item := range doc.Paths {
		for method, raw := range item {
			if !isHTTPMethod(method) {
				continue
			}
			op, ok := raw.(map[string]any)
			if !ok {
				return fmt.Errorf("%s %s operation must be an object", strings.ToUpper(method), path)
			}
			id, _ := op["operationId"].(string)
			binding, ok := OperationBindings[id]
			if !ok {
				return fmt.Errorf("missing OperationBinding for %s %s (%s)", strings.ToUpper(method), path, id)
			}
			switch {
			case binding.Multipart:
				if err := requireMultipartRequest(op, id); err != nil {
					return err
				}
			case binding.Request != "":
				if err := requireJSONRequestSchema(op, id); err != nil {
					return err
				}
			}
			if err := requireSuccessResponseSchema(op, id, binding); err != nil {
				return err
			}
		}
	}
	return nil
}

func requireJSONRequestSchema(op map[string]any, operationID string) error {
	body, ok := op["requestBody"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s missing requestBody", operationID)
	}
	content, ok := body["content"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s requestBody missing content", operationID)
	}
	jsonContent, ok := content["application/json"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s requestBody missing application/json", operationID)
	}
	schema, ok := jsonContent["schema"].(map[string]any)
	if !ok || schema["$ref"] == nil {
		return fmt.Errorf("%s requestBody missing schema $ref", operationID)
	}
	return nil
}

func requireMultipartRequest(op map[string]any, operationID string) error {
	body, ok := op["requestBody"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s missing multipart requestBody", operationID)
	}
	content, ok := body["content"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s multipart requestBody missing content", operationID)
	}
	if _, ok := content["multipart/form-data"]; !ok {
		return fmt.Errorf("%s missing multipart/form-data content", operationID)
	}
	return nil
}

func requireSuccessResponseSchema(op map[string]any, operationID string, binding OperationBinding) error {
	status := binding.SuccessStatus
	if status == 0 {
		status = 200
	}
	responses, ok := op["responses"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s missing responses", operationID)
	}
	raw, ok := responses[fmt.Sprintf("%d", status)].(map[string]any)
	if !ok {
		return fmt.Errorf("%s missing %d response", operationID, status)
	}
	content, ok := raw["content"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s %d response missing content", operationID, status)
	}
	jsonContent, ok := content["application/json"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s %d response missing application/json", operationID, status)
	}
	schema, ok := jsonContent["schema"].(map[string]any)
	if !ok || schema["$ref"] == nil {
		return fmt.Errorf("%s %d response missing schema $ref", operationID, status)
	}
	return nil
}

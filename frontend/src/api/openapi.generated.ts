// Code generated from backend/openapi/api.yaml contract surface. DO NOT EDIT.
// backend/openapi/contract_test.go verifies this file stays synchronized.

export const API_OPERATIONS = [
  { method: "POST", path: "/api/v1/admin/attempts/{attemptID}/usage", operationId: "postApiV1AdminAttemptsAttemptidUsage" },
  { method: "POST", path: "/api/v1/admin/attention/{itemID}/resolve", operationId: "postApiV1AdminAttentionItemidResolve" },
  { method: "GET", path: "/api/v1/admin/audit", operationId: "getApiV1AdminAudit" },
  { method: "GET", path: "/api/v1/admin/audit/{eventID}", operationId: "getApiV1AdminAuditEventid" },
  { method: "POST", path: "/api/v1/admin/canon-branches/{branchID}/commit", operationId: "postApiV1AdminCanonBranchesBranchidCommit" },
  { method: "POST", path: "/api/v1/admin/canon-branches/{branchID}/provisional-versions", operationId: "postApiV1AdminCanonBranchesBranchidProvisionalVersions" },
  { method: "GET", path: "/api/v1/admin/canon-branches/{branchID}/versions", operationId: "getApiV1AdminCanonBranchesBranchidVersions" },
  { method: "POST", path: "/api/v1/admin/canon-branches/{branchID}/versions", operationId: "postApiV1AdminCanonBranchesBranchidVersions" },
  { method: "POST", path: "/api/v1/admin/canon-versions/{versionID}/promote", operationId: "postApiV1AdminCanonVersionsVersionidPromote" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/approve", operationId: "postApiV1AdminChaptersChapteridApprove" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/audio", operationId: "getApiV1AdminChaptersChapteridAudio" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/audio", operationId: "postApiV1AdminChaptersChapteridAudio" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/audio/{assetID}/activate", operationId: "postApiV1AdminChaptersChapteridAudioAssetidActivate" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/audio/{assetID}/preview-url", operationId: "getApiV1AdminChaptersChapteridAudioAssetidPreviewUrl" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/content", operationId: "getApiV1AdminChaptersChapteridContent" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/content", operationId: "postApiV1AdminChaptersChapteridContent" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/continuity", operationId: "postApiV1AdminChaptersChapteridContinuity" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/duration", operationId: "postApiV1AdminChaptersChapteridDuration" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/edit", operationId: "postApiV1AdminChaptersChapteridEdit" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/generation-run/latest", operationId: "getApiV1AdminChaptersChapteridGenerationRunLatest" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/narration", operationId: "postApiV1AdminChaptersChapteridNarration" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/narration/latest", operationId: "getApiV1AdminChaptersChapteridNarrationLatest" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/narration/{narrationID}/audio/latest-ready", operationId: "getApiV1AdminChaptersChapteridNarrationNarrationidAudioLatestReady" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/narration/{narrationID}/segments", operationId: "postApiV1AdminChaptersChapteridNarrationNarrationidSegments" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/narration/{narrationID}/synthesize", operationId: "postApiV1AdminChaptersChapteridNarrationNarrationidSynthesize" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/plans", operationId: "postApiV1AdminChaptersChapteridPlans" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/publish", operationId: "postApiV1AdminChaptersChapteridPublish" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/publish-readiness", operationId: "getApiV1AdminChaptersChapteridPublishReadiness" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/quality", operationId: "postApiV1AdminChaptersChapteridQuality" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/ready", operationId: "postApiV1AdminChaptersChapteridReady" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/regenerate", operationId: "postApiV1AdminChaptersChapteridRegenerate" },
  { method: "GET", path: "/api/v1/admin/chapters/{chapterID}/reviews", operationId: "getApiV1AdminChaptersChapteridReviews" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/reviews", operationId: "postApiV1AdminChaptersChapteridReviews" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/revision-impact", operationId: "postApiV1AdminChaptersChapteridRevisionImpact" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/rewrite", operationId: "postApiV1AdminChaptersChapteridRewrite" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/safety", operationId: "postApiV1AdminChaptersChapteridSafety" },
  { method: "POST", path: "/api/v1/admin/chapters/{chapterID}/unpublish", operationId: "postApiV1AdminChaptersChapteridUnpublish" },
  { method: "GET", path: "/api/v1/admin/context-snapshots/{snapshotID}", operationId: "getApiV1AdminContextSnapshotsSnapshotid" },
  { method: "POST", path: "/api/v1/admin/creative-decisions/{decisionID}/postpone", operationId: "postApiV1AdminCreativeDecisionsDecisionidPostpone" },
  { method: "POST", path: "/api/v1/admin/creative-decisions/{decisionID}/reject", operationId: "postApiV1AdminCreativeDecisionsDecisionidReject" },
  { method: "POST", path: "/api/v1/admin/creative-decisions/{decisionID}/select", operationId: "postApiV1AdminCreativeDecisionsDecisionidSelect" },
  { method: "GET", path: "/api/v1/admin/generation-jobs/{jobID}", operationId: "getApiV1AdminGenerationJobsJobid" },
  { method: "POST", path: "/api/v1/admin/generation-jobs/{jobID}/retry", operationId: "postApiV1AdminGenerationJobsJobidRetry" },
  { method: "POST", path: "/api/v1/admin/plot-threads/{threadID}/events", operationId: "postApiV1AdminPlotThreadsThreadidEvents" },
  { method: "GET", path: "/api/v1/admin/retcons", operationId: "getApiV1AdminRetcons" },
  { method: "POST", path: "/api/v1/admin/retcons", operationId: "postApiV1AdminRetcons" },
  { method: "GET", path: "/api/v1/admin/retcons/{id}", operationId: "getApiV1AdminRetconsId" },
  { method: "POST", path: "/api/v1/admin/retcons/{id}/analyze", operationId: "postApiV1AdminRetconsIdAnalyze" },
  { method: "POST", path: "/api/v1/admin/retcons/{id}/apply", operationId: "postApiV1AdminRetconsIdApply" },
  { method: "POST", path: "/api/v1/admin/retcons/{id}/approve", operationId: "postApiV1AdminRetconsIdApprove" },
  { method: "POST", path: "/api/v1/admin/retcons/{id}/cancel", operationId: "postApiV1AdminRetconsIdCancel" },
  { method: "POST", path: "/api/v1/admin/retcons/{id}/ready", operationId: "postApiV1AdminRetconsIdReady" },
  { method: "POST", path: "/api/v1/admin/revisions/{revisionID}/reject", operationId: "postApiV1AdminRevisionsRevisionidReject" },
  { method: "POST", path: "/api/v1/admin/runs", operationId: "postApiV1AdminRuns" },
  { method: "GET", path: "/api/v1/admin/runs/{runID}", operationId: "getApiV1AdminRunsRunid" },
  { method: "POST", path: "/api/v1/admin/runs/{runID}/jobs", operationId: "postApiV1AdminRunsRunidJobs" },
  { method: "POST", path: "/api/v1/admin/runs/{runID}/mark-stale", operationId: "postApiV1AdminRunsRunidMarkStale" },
  { method: "GET", path: "/api/v1/admin/stories", operationId: "getApiV1AdminStories" },
  { method: "POST", path: "/api/v1/admin/stories", operationId: "postApiV1AdminStories" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/activate", operationId: "postApiV1AdminStoriesStoryidActivate" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/activation-readiness", operationId: "getApiV1AdminStoriesStoryidActivationReadiness" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/archive", operationId: "postApiV1AdminStoriesStoryidArchive" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/arcs", operationId: "getApiV1AdminStoriesStoryidArcs" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/arcs", operationId: "postApiV1AdminStoriesStoryidArcs" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/arcs/{arcID}/completion", operationId: "getApiV1AdminStoriesStoryidArcsArcidCompletion" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/attention", operationId: "getApiV1AdminStoriesStoryidAttention" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/attention", operationId: "postApiV1AdminStoriesStoryidAttention" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/batch-generate", operationId: "postApiV1AdminStoriesStoryidBatchGenerate" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/bible", operationId: "getApiV1AdminStoriesStoryidBible" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/bible/versions", operationId: "postApiV1AdminStoriesStoryidBibleVersions" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/canon-branches", operationId: "postApiV1AdminStoriesStoryidCanonBranches" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/canon-branches/active-official", operationId: "getApiV1AdminStoriesStoryidCanonBranchesActiveOfficial" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/canon-repair", operationId: "postApiV1AdminStoriesStoryidCanonRepair" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/chapters", operationId: "getApiV1AdminStoriesStoryidChapters" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/chapters", operationId: "postApiV1AdminStoriesStoryidChapters" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/characters", operationId: "getApiV1AdminStoriesStoryidCharacters" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/characters", operationId: "postApiV1AdminStoriesStoryidCharacters" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/content-profile", operationId: "getApiV1AdminStoriesStoryidContentProfile" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/content-profile", operationId: "postApiV1AdminStoriesStoryidContentProfile" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/context-snapshots", operationId: "getApiV1AdminStoriesStoryidContextSnapshots" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/context-snapshots", operationId: "postApiV1AdminStoriesStoryidContextSnapshots" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/cover", operationId: "postApiV1AdminStoriesStoryidCover" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/creative-decisions", operationId: "getApiV1AdminStoriesStoryidCreativeDecisions" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/creative-decisions", operationId: "postApiV1AdminStoriesStoryidCreativeDecisions" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/ending", operationId: "getApiV1AdminStoriesStoryidEnding" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/ending/versions", operationId: "postApiV1AdminStoriesStoryidEndingVersions" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/facts", operationId: "getApiV1AdminStoriesStoryidFacts" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/facts", operationId: "postApiV1AdminStoriesStoryidFacts" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/foundation", operationId: "postApiV1AdminStoriesStoryidFoundation" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/generation-policy", operationId: "getApiV1AdminStoriesStoryidGenerationPolicy" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/make-private", operationId: "postApiV1AdminStoriesStoryidMakePrivate" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/make-public", operationId: "postApiV1AdminStoriesStoryidMakePublic" },
  { method: "PUT", path: "/api/v1/admin/stories/{storyID}/metadata", operationId: "putApiV1AdminStoriesStoryidMetadata" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/plot-threads", operationId: "getApiV1AdminStoriesStoryidPlotThreads" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/plot-threads", operationId: "postApiV1AdminStoriesStoryidPlotThreads" },
  { method: "POST", path: "/api/v1/admin/stories/{storyID}/restore", operationId: "postApiV1AdminStoriesStoryidRestore" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/thread-inactivity", operationId: "getApiV1AdminStoriesStoryidThreadInactivity" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/usage", operationId: "getApiV1AdminStoriesStoryidUsage" },
  { method: "GET", path: "/api/v1/admin/stories/{storyID}/workflow-settings", operationId: "getApiV1AdminStoriesStoryidWorkflowSettings" },
  { method: "PUT", path: "/api/v1/admin/stories/{storyID}/workflow-settings", operationId: "putApiV1AdminStoriesStoryidWorkflowSettings" },
  { method: "POST", path: "/api/v1/admin/tts-segments/{segmentID}/synthesize", operationId: "postApiV1AdminTtsSegmentsSegmentidSynthesize" },
  { method: "GET", path: "/api/v1/admin/users", operationId: "getApiV1AdminUsers" },
  { method: "GET", path: "/api/v1/admin/users/{userID}", operationId: "getApiV1AdminUsersUserid" },
  { method: "DELETE", path: "/api/v1/admin/users/{userID}/roles/admin", operationId: "deleteApiV1AdminUsersUseridRolesAdmin" },
  { method: "POST", path: "/api/v1/admin/users/{userID}/roles/admin", operationId: "postApiV1AdminUsersUseridRolesAdmin" },
  { method: "PATCH", path: "/api/v1/admin/users/{userID}/status", operationId: "patchApiV1AdminUsersUseridStatus" },
  { method: "POST", path: "/api/v1/auth/account/deletion/cancel", operationId: "postApiV1AuthAccountDeletionCancel" },
  { method: "POST", path: "/api/v1/auth/account/deletion/request", operationId: "postApiV1AuthAccountDeletionRequest" },
  { method: "POST", path: "/api/v1/auth/email/resend", operationId: "postApiV1AuthEmailResend" },
  { method: "POST", path: "/api/v1/auth/email/verify", operationId: "postApiV1AuthEmailVerify" },
  { method: "POST", path: "/api/v1/auth/login", operationId: "postApiV1AuthLogin" },
  { method: "POST", path: "/api/v1/auth/logout", operationId: "postApiV1AuthLogout" },
  { method: "POST", path: "/api/v1/auth/logout-all", operationId: "postApiV1AuthLogoutAll" },
  { method: "GET", path: "/api/v1/auth/me", operationId: "getApiV1AuthMe" },
  { method: "POST", path: "/api/v1/auth/mfa/totp/confirm", operationId: "postApiV1AuthMfaTotpConfirm" },
  { method: "POST", path: "/api/v1/auth/mfa/totp/disable", operationId: "postApiV1AuthMfaTotpDisable" },
  { method: "POST", path: "/api/v1/auth/mfa/totp/setup", operationId: "postApiV1AuthMfaTotpSetup" },
  { method: "POST", path: "/api/v1/auth/password/forgot", operationId: "postApiV1AuthPasswordForgot" },
  { method: "POST", path: "/api/v1/auth/password/reset", operationId: "postApiV1AuthPasswordReset" },
  { method: "POST", path: "/api/v1/auth/re-auth", operationId: "postApiV1AuthReAuth" },
  { method: "POST", path: "/api/v1/auth/refresh", operationId: "postApiV1AuthRefresh" },
  { method: "POST", path: "/api/v1/auth/register", operationId: "postApiV1AuthRegister" },
  { method: "GET", path: "/api/v1/auth/sessions", operationId: "getApiV1AuthSessions" },
  { method: "DELETE", path: "/api/v1/auth/sessions/{sessionID}", operationId: "deleteApiV1AuthSessionsSessionid" },
  { method: "GET", path: "/api/v1/chapters/{chapterID}/audio-url", operationId: "getApiV1ChaptersChapteridAudioUrl" },
  { method: "GET", path: "/api/v1/chapters/{chapterID}/content", operationId: "getApiV1ChaptersChapteridContent" },
  { method: "GET", path: "/api/v1/genres", operationId: "getApiV1Genres" },
  { method: "GET", path: "/api/v1/me/favorites", operationId: "getApiV1MeFavorites" },
  { method: "DELETE", path: "/api/v1/me/favorites/{storyID}", operationId: "deleteApiV1MeFavoritesStoryid" },
  { method: "PUT", path: "/api/v1/me/favorites/{storyID}", operationId: "putApiV1MeFavoritesStoryid" },
  { method: "GET", path: "/api/v1/me/library", operationId: "getApiV1MeLibrary" },
  { method: "GET", path: "/api/v1/me/progress/{chapterID}", operationId: "getApiV1MeProgressChapterid" },
  { method: "PUT", path: "/api/v1/me/progress/{chapterID}", operationId: "putApiV1MeProgressChapterid" },
  { method: "POST", path: "/api/v1/me/progress/{chapterID}/complete", operationId: "postApiV1MeProgressChapteridComplete" },
  { method: "GET", path: "/api/v1/stories", operationId: "getApiV1Stories" },
  { method: "GET", path: "/api/v1/stories/{storyID}", operationId: "getApiV1StoriesStoryid" },
  { method: "GET", path: "/api/v1/stories/{storyID}/chapters", operationId: "getApiV1StoriesStoryidChapters" },
  { method: "GET", path: "/health", operationId: "getHealth" },
  { method: "GET", path: "/ready", operationId: "getReady" },
] as const

export type ApiOperation = (typeof API_OPERATIONS)[number]
export type ApiMethod = ApiOperation["method"]
export type ApiPath = ApiOperation["path"]
export type ApiOperationId = ApiOperation["operationId"]

export interface ActivationReadinessResponse {
  generation_policy?: GenerationPolicyResponse
  missing: string[]
  ready: boolean
  story_workspace?: Record<string, unknown>
}

export interface AdminChapterContentResponse {
  generation_job?: JobViewPascal
  generation_run?: GenerationRunPascal
  revisions: ContentRevisionPascal[]
}

export interface AdminStatusRequest {
  status: "ACTIVE" | "SUSPENDED" | "DEACTIVATED"
}

export interface AdminUserListResponse {
  users: AdminUserSummary[]
}

export interface AdminUserSummary {
  display_name: string
  email: string
  email_verified: boolean
  id: string
  roles: ("GUEST" | "USER" | "ADMIN")[]
  status: "ACTIVE" | "SUSPENDED" | "DEACTIVATED"
}

export interface AnalyzeDurationRequest {
  revision_id?: string
  text?: string
}

export interface ApplyRetconRequest {
  applied_by?: string
}

export interface ApproveContentRequest {
  approved_by?: string
  revision_id: string
}

export interface ApproveRetconRequest {
  approved_by?: string
}

export interface ArcCompletionResultPascal {
  ArcID?: string
  Complete?: boolean
  CompletedChapters?: number
  PendingChapters?: string[]
  TotalChapters?: number
}

export interface AttentionItemListResponse {
  items: AttentionItemPascal[]
}

export interface AttentionItemPascal {
  Action?: string
  ChapterID?: string
  Detail?: string
  ID?: string
  Kind?: string
  Priority?: string
  Resolved?: boolean
  StoryID?: string
  Title?: string
}

export interface AudioAssetPascal {
  BitrateKbps?: number
  ChapterID?: string
  Checksum?: string
  DurationMs?: number
  ID?: string
  IsActive?: boolean
  MimeType?: string
  SizeBytes?: number
  SourceNarrationRevisionID?: string
  Status?: "READY"
  StorageKey?: string
  VersionNo?: number
}

export interface AudioURLResponse {
  url: string
}

export interface AuditEvent {
  action: string
  actor_type: "USER" | "SYSTEM" | "AI" | "ANONYMOUS"
  actor_user_id?: string
  chapter_id?: string
  correlation_id?: string
  created_at: string
  generation_run_id?: string
  id: string
  metadata?: Record<string, unknown>
  provenance?: Record<string, unknown>
  request_id?: string
  resource_id?: string
  resource_type?: string
  result: "SUCCEEDED" | "FAILED" | "DENIED"
  story_id?: string
}

export interface AuditEventListResponse {
  items: AuditEvent[]
}

export interface CanonBranchPascal {
  ID?: string
  Status?: string
  StoryID?: string
  Type?: string
}

export interface CanonChangeItemPascal {
  CanonVersionID?: string
  ChangeType?: string
  EntityID?: string
  EntityType?: string
  ID?: string
  Metadata?: Record<string, unknown>
}

export interface CanonCommitResultPascal {
  ChangeItems?: CanonChangeItemPascal[]
  Version?: CanonVersionPascal
}

export interface CanonVersionListResponse {
  versions: CanonVersionPascal[]
}

export interface CanonVersionPascal {
  BranchID?: string
  CommittedBy?: string
  ID?: string
  ParentVersionID?: string
  SequenceNo?: number
  SourceChapterID?: string
  SourceContentRevisionID?: string
  SourceProvisionalVersionID?: string
  Status?: string
  StoryID?: string
}

export interface ChapterListResponse {
  chapters: ChapterPascal[]
}

export interface ChapterPascal {
  ArcID?: string
  ChapterNumber: number
  ID: string
  Status: "DRAFT" | "READY" | "PUBLISHED" | "ARCHIVED"
  StoryID: string
  Title: string
}

export interface ChapterPlanRevisionPascal {
  ChapterID?: string
  CreatedBy?: string
  ID?: string
  Plan?: Record<string, unknown>
  RevisionNo?: number
  SourceType?: string
}

export interface ChapterReviewListResponse {
  reviews: ChapterReviewPascal[]
}

export interface ChapterReviewPascal {
  ChapterID?: string
  ContentRevisionID?: string
  ID?: string
  Outcome?: string
  Report?: Record<string, unknown>
  ReviewType?: string
}

export interface CharacterListResponse {
  characters: CharacterPascal[]
}

export interface CharacterPascal {
  CanonicalName?: string
  CurrentProfileVersionID?: string
  ID?: string
  Importance?: string
  StoryID?: string
}

export interface CommitCanonRequest {
  committed_by?: string
  content_revision_id: string
  source_chapter_id: string
  story_id: string
}

export interface ContentApprovalPascal {
  ApprovedBy?: string
  ChapterID?: string
  ContentRevisionID?: string
  ID?: string
  OverrideSnapshot?: Record<string, unknown>
  WarningsSnapshot?: Record<string, unknown>
}

export interface ContentProfileRequest {
  allowed_themes?: string[]
  constraints?: Record<string, unknown>
  disallowed_themes?: string[]
  language_limits?: string
  maturity_target?: string
  romance_limits?: string
  violence_level?: string
}

export interface ContentProfileResponse {
  id: string
  profile: Record<string, unknown>
  story_id: string
  version_no: number
}

export interface ContentRevisionPascal {
  BaseCanonVersionID?: string
  BasedOnRevisionID?: string
  ChapterID?: string
  ContentText?: string
  CreatedBy?: string
  GenerationRunID?: string
  ID?: string
  PlanRevisionID?: string
  RevisionNo?: number
  SourceType?: string
  Status?: "CANDIDATE" | "APPROVED"
}

export interface ContextSnapshotListResponse {
  snapshots: ContextSnapshotPascal[]
}

export interface ContextSnapshotPascal {
  ArcVersionID?: string
  BibleVersionID?: string
  ChapterID?: string
  ContentProfileVersionID?: string
  EndingPlanVersionID?: string
  ID?: string
  Model?: string
  PromptVersion?: string
  Provider?: string
  StoryID?: string
  WorkflowVersion?: string
}

export interface CreateArcRequest {
  content: Record<string, unknown>
}

export interface CreateAttentionItemRequest {
  action?: string
  chapter_id?: string
  detail?: string
  kind?: string
  priority?: string
  title: string
}

export interface CreateAudioAssetRequest {
  bitrate_kbps?: number
  duration_ms?: number
  mime_type: string
  size_bytes?: number
  source_narration_revision_id: string
  storage_key: string
}

export interface CreateCanonBranchRequest {
  type?: string
}

export interface CreateCanonVersionRequest {
  committed_by?: string
  source_chapter_id: string
  story_id: string
}

export interface CreateChapterRequest {
  created_by?: string
  title: string
}

export interface CreateCharacterRequest {
  importance?: string
  name: string
  profile?: Record<string, unknown>
}

export interface CreateContextSnapshotRequest {
  arc_version_id?: string
  bible_version_id?: string
  chapter_id?: string
  content_profile_version_id?: string
  ending_plan_version_id?: string
  model?: string
  prompt_version?: string
  provider?: string
  workflow_version?: string
}

export interface CreateCreativeDecisionRequest {
  arc_id?: string
  blocking_level?: string
  chapter_id?: string
  context_summary?: string
  created_by?: string
  decision_type?: string
  origin?: string
  question: string
  severity?: string
}

export interface CreateFactRequest {
  fact_type: string
  importance?: string
  subject_id: string
  subject_type: string
  value: Record<string, unknown>
}

export interface CreateJobRequest {
  job_type: string
  max_attempts?: number
}

export interface CreateNarrationRequest {
  created_by?: string
  script?: string
  source_content_revision_id: string
  voice_id: string
}

export interface CreatePlanRequest {
  created_by?: string
  plan: Record<string, unknown>
}

export interface CreatePlotThreadEventRequest {
  chapter_id?: string
  detail?: Record<string, unknown>
  event_type: string
}

export interface CreatePlotThreadRequest {
  importance?: string
  summary?: string
  title: string
}

export interface CreateProvisionalCanonVersionRequest {
  source_chapter_id: string
  source_content_revision_id: string
  story_id: string
}

export interface CreateRetconRequest {
  proposed_change: string
  reason: string
  requested_by?: string
  story_id: string
  target_chapter_id: string
}

export interface CreateReviewRequest {
  content_revision_id: string
  outcome: "PASS" | "OVERRIDABLE_BLOCK"
  report?: Record<string, unknown>
  review_type: "CONTINUITY" | "QUALITY" | "SAFETY" | "DURATION"
}

export interface CreateRunRequest {
  chapter_id?: string
  requested_by?: string
  run_type: string
  story_id?: string
}

export interface CreateStoryRequest {
  created_by?: string
  description?: string
  policy?: GenerationPolicyInput
  title: string
}

export interface CreativeDecisionListResponse {
  decisions: CreativeDecisionPascal[]
}

export interface CreativeDecisionPascal {
  ArcID?: string
  BlockingLevel?: string
  ChapterID?: string
  ContextSummary?: string
  DecisionType?: string
  ID?: string
  Origin?: string
  Question?: string
  Severity?: string
  Status?: string
  StoryID?: string
}

export interface DurationOutputPascal {
  EstimatedMinutes?: number
  Outcome?: string
  Report?: Record<string, unknown>
}

export interface EditContentRequest {
  based_on_revision_id?: string
  edited_by?: string
  text: string
}

export interface EmailResendRequest {
  email: string
}

export interface EmailVerifyRequest {
  email: string
  token: string
}

export interface EndingPlanVersionPascal {
  BasedOnVersionID?: string
  Content?: Record<string, unknown>
  CreatedBy?: string
  ID?: string
  StoryID?: string
  VersionNo?: number
}

export interface ErrorBody {
  code: string
  message: string
}

export interface ErrorResponse {
  error: ErrorBody
}

export interface FavoriteListResponse {
  favorites: Record<string, unknown>[]
}

export interface FavoriteStatusResponse {
  status: "favorited" | "unfavorited"
}

export interface FavoriteStory {
  description?: string
  favorited_at?: string
  slug?: string
  story_id?: string
  title?: string
}

export interface FoundationRequest {
  created_by?: string
  premise: string
}

export interface FoundationResponse {
  arcs?: StoryArcPascal[]
  bible?: StoryBibleVersionPascal
  characters?: CharacterPascal[]
  ending?: EndingPlanVersionPascal
}

export interface GenerationJobPascal {
  AttemptCount?: number
  ID?: string
  InputFingerprint?: string
  JobType?: string
  LastErrorClass?: string
  LastErrorCode?: string
  LockedBy?: string
  MaxAttempts?: number
  OutputRef?: string
  Priority?: number
  RunID?: string
  Status?: string
}

export interface GenerationPolicyInput {
  content_origin?: string
  language?: string
  minimum_audio_duration_sec?: number
  narration_language?: string
  target_audio_duration_sec?: number
}

export interface GenerationPolicyResponse {
  content_origin: string
  language: string
  minimum_audio_duration_sec: number
  narration_language: string
  policy_version: number
  story_id: string
  target_audio_duration_sec: number
}

export interface GenerationRunPascal {
  BaseCanonVersionID?: string
  ChapterID?: string
  ContextSnapshotID?: string
  ID?: string
  IdempotencyKey?: string
  Priority?: number
  RequestedBy?: string
  RunType?: string
  Status?: string
  StoryID?: string
  WaitingReason?: string
  WorkflowVersion?: string
}

export interface Genre {
  id: string
  name: string
  slug: string
}

export interface GenreListResponse {
  genres: Genre[]
}

export interface HealthResponse {
  status: "ok"
}

export interface JobAttemptListResponse {
  usage: JobAttemptPascal[]
}

export interface JobAttemptPascal {
  AttemptNo?: number
  ErrorClass?: string
  ErrorCode?: string
  ID?: string
  JobID?: string
  LatencyMs?: number
  Model?: string
  Provider?: string
  Status?: string
  Usage?: Record<string, unknown>
}

export interface JobViewPascal {
  AttemptCount?: number
  AttemptsExhausted?: boolean
  ID?: string
  JobType?: string
  LastErrorClass?: string
  LastErrorCode?: string
  MaxAttempts?: number
  Observation?: string
  Retryable?: boolean
  RunID?: string
  Status?: string
}

export interface LibraryItem {
  audio_asset_id?: string
  audio_duration_ms?: number
  chapter_id?: string
  chapter_number?: number
  chapter_title?: string
  completed_at?: string
  last_listened_at?: string
  position_ms?: number
  relisten_status?: RelistenStatus
  story_description?: string
  story_id?: string
  story_slug?: string
  story_title?: string
  updated_at?: string
}

export interface LibraryResponse {
  completed: LibraryItem[]
  continue_listening?: LibraryItem
  favorites: FavoriteStory[]
  recent: LibraryItem[]
}

export interface ListeningProgressPascal {
  ChapterID?: string
  CompletedAt?: string
  LastAudioAssetID?: string
  LastPlaybackSessionID?: string
  PositionMs?: number
  RelistenStatus?: RelistenStatus
  UserID?: string
  Version?: number
}

export interface LoginRequest {
  email: string
  password: string
}

export interface MFAConfirmRequest {
  code: string
}

export interface MFAConfirmResponse {
  recovery_codes: string[]
}

export interface MFASetupResponse {
  secret: string
}

export interface MarkStaleRequest {
  chapter_id?: string
}

export interface MeResponse {
  email: string
  email_verified: boolean
  id: string
  mfa_enabled: boolean
  roles: ("GUEST" | "USER" | "ADMIN")[]
  status: string
}

export interface NarrationRevisionPascal {
  ChapterID?: string
  CreatedBy?: string
  ID?: string
  RevisionNo?: number
  Script?: string
  SourceContentRevisionID?: string
  Status?: string
  VoiceID?: string
}

export interface PasswordForgotRequest {
  email: string
}

export interface PasswordResetRequest {
  email: string
  new_password: string
  token: string
}

export interface PlotThreadEventPascal {
  ChapterID?: string
  Detail?: Record<string, unknown>
  EventType?: string
  ID?: string
  PlotThreadID?: string
}

export interface PlotThreadListResponse {
  plot_threads: PlotThreadPascal[]
}

export interface PlotThreadPascal {
  ID?: string
  Importance?: string
  Status?: string
  StoryID?: string
  Summary?: string
  Title?: string
}

export interface PostponeCreativeDecisionRequest {
  reason: string
}

export interface ProgressConflictResponse {
  error: ErrorBody
  progress: ListeningProgressPascal
}

export interface PromoteProvisionalVersionRequest {
  committed_by?: string
}

export interface PublishedChapterContentResponse {
  chapter_id: string
  content_text: string
  revision_id: string
}

export interface PublishedChapterListResponse {
  chapters: ChapterPascal[]
}

export interface ReAuthRequest {
  code?: string
  recovery_code?: string
}

export interface ReadinessResponse {
  missing: string[]
  ready: boolean
}

export interface ReadyResponse {
  dependencies?: Record<string, unknown>
  error?: string
  status: "ready" | "degraded" | "unavailable"
}

export interface RecordUsageRequest {
  usage?: Record<string, unknown>
}

export interface RegenerateContentRequest {
  based_on_revision_id?: string
  requested_by?: string
}

export interface RegisterRequest {
  email: string
  password: string
}

export interface RejectContentRequest {
  reason?: string
  rejected_by?: string
}

export interface RejectCreativeDecisionRequest {
  rejected_by?: string
  scope?: string
}

export type RelistenStatus = "NO_RELISTEN_NEEDED" | "RELISTEN_REQUIRED" | "RELISTEN_RECOMMENDED"

export interface RepairCanonRequest {
  branch_id: string
  committed_by?: string
  content_revision_id: string
  source_chapter_id: string
}

export interface RetconListResponse {
  retcons: RetconRequestPascal[]
}

export interface RetconRequestPascal {
  AppliedBy?: string
  ApprovedBy?: string
  ID?: string
  ImpactScope?: string
  ProposedChange?: string
  Reason?: string
  RequestedBy?: string
  Status?: "DRAFT" | "ANALYZING" | "APPROVED" | "READY_TO_APPLY" | "APPLIED" | "CANCELLED"
  StoryID?: string
  TargetChapterID?: string
}

export interface RevisionImpactRequest {
  relisten_status: RelistenStatus
}

export interface RevisionImpactResponse {
  affected_listeners: number
}

export interface RewriteChapterRequest {
  based_on_revision_id?: string
  created_by?: string
  feedback?: string
}

export interface RunReviewRequest {
  revision_id?: string
  text?: string
}

export interface SaveProgressRequest {
  audio_asset_id?: string
  expected_version?: number
  playback_session_id?: string
  position_ms: number
}

export interface SelectCreativeDecisionRequest {
  selected_by?: string
}

export interface SessionListResponse {
  items: SessionResponse[]
}

export interface SessionResponse {
  created_at: string
  current: boolean
  expires_at: string
  id: string
  last_used_at: string
  safe_ip_metadata?: string
  user_agent_summary?: string
}

export interface StaleJobsResponse {
  stale_jobs: GenerationJobPascal[]
}

export interface StartBatchRequest {
  chapter_ids: string[]
  requested_by?: string
}

export interface StatusResponse {
  status: string
}

export interface StoryArcListResponse {
  arcs: StoryArcPascal[]
}

export interface StoryArcPascal {
  CurrentVersionID?: string
  ID?: string
  Ordinal?: number
  Status?: string
  StoryID?: string
}

export interface StoryAssetResponse {
  id: string
  mime_type: string
  size_bytes: number
  status: "PENDING" | "READY"
  storage_key: string
  story_id: string
  type: "COVER"
}

export interface StoryBibleVersionPascal {
  BasedOnVersionID?: string
  Content?: Record<string, unknown>
  CreatedBy?: string
  ID?: string
  StoryID?: string
  VersionNo?: number
}

export interface StoryFactListResponse {
  facts: StoryFactPascal[]
}

export interface StoryFactPascal {
  FactType?: string
  ID?: string
  Importance?: string
  Status?: string
  StoryID?: string
  SubjectID?: string
  SubjectType?: string
  SupersedesFactID?: string
  Value?: Record<string, unknown>
}

export interface StoryListResponse {
  stories: StorySummary[]
}

export interface StorySummary {
  description: string
  id: string
  slug: string
  status: "DRAFT" | "ACTIVE" | "COMPLETED" | "ARCHIVED"
  title: string
  visibility: "PRIVATE" | "PUBLIC"
}

export interface TTSSegmentListResponse {
  segments: TTSSegmentPascal[]
}

export interface TTSSegmentPascal {
  DurationMs?: number
  ID?: string
  Model?: string
  NarrationRevisionID?: string
  Provider?: string
  SegmentNo?: number
  Status?: string
  TempStorageKey?: string
  Text?: string
  VoiceID?: string
}

export interface ThreadInactivityListResponse {
  inactive_threads: Record<string, unknown>[]
}

export interface TokenResponse {
  access_token: string
  expires_in: number
  status: string
  token_type: "Bearer"
}

export interface UpdateMetadataRequest {
  description?: string
  title?: string
}

export interface UserResponse {
  email: string
  id: string
  status: "ACTIVE" | "SUSPENDED" | "DEACTIVATED"
}

export interface VersionContentRequest {
  content: Record<string, unknown>
}

export interface WorkflowSettingsRequest {
  auto_ai_review?: boolean
  batch_generation_size?: number
  creative_autonomy?: string
  fallback_policy?: Record<string, unknown>
  pause_before_tts?: boolean
  planning_horizon?: number
  preferred_text_model?: string
  preferred_text_provider?: string
  preferred_tts_provider?: string
  preferred_voice_id?: string
}

export interface WorkflowSettingsResponse {
  auto_ai_review?: boolean
  batch_generation_size?: number
  creative_autonomy?: string
  fallback_policy?: Record<string, unknown>
  pause_before_tts?: boolean
  planning_horizon?: number
  preferred_text_model?: string
  preferred_text_provider?: string
  preferred_tts_provider?: string
  preferred_voice_id?: string
  story_id: string
}

export interface WriteChapterRequest {
  created_by?: string
  prompt?: string
}


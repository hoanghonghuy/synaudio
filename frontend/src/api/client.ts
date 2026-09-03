import type {
  ApiError,
  AuditEvent,
  AuditListResponse,
  AuthUser,
  ArcCompletionResult,
  AttentionListResponse,
  AudioURLResponse,
  Chapter,
  ChapterContent,
  ChapterListResponse,
  ChapterReview,
  ChapterReviewListResponse,
  ContentRevision,
  ContentRevisionListResponse,
  CreateStoryInput,
  CreativeDecisionListResponse,
  FavoriteListResponse,
  GenreListResponse,
  ListeningProgress,
  Story,
  StoryListResponse,
  ThreadInactivityResponse,
  UsageListResponse,
} from './types'

const BASE = '/api/v1'

type TokenResponse = {
  status: string
  access_token: string
  token_type: string
  expires_in: number
}

export type AuthSession = {
  id: string
  current: boolean
  created_at: string
  last_used_at: string
  expires_at: string
  user_agent_summary?: string
  safe_ip_metadata?: string
}

export type AuditFilters = {
  actor_id?: string
  action?: string
  resource_type?: string
  resource_id?: string
  story_id?: string
  chapter_id?: string
  run_id?: string
  correlation_id?: string
  result?: string
  from?: string
  to?: string
  limit?: number
}

export type StoryWorkflowSettings = {
  story_id: string
  batch_generation_size: number
  creative_autonomy: string
  preferred_text_provider: string
  preferred_text_model: string
  preferred_tts_provider: string
  preferred_voice_id: string
  pause_before_tts: boolean
  auto_ai_review: boolean
  planning_horizon: number
  fallback_policy: Record<string, unknown>
}

export type StoryContentProfile = {
  id: string
  story_id: string
  version_no: number
  profile: Record<string, unknown>
}

export type StoryBibleVersion = {
  ID: string
  StoryID: string
  VersionNo: number
  Content: Record<string, unknown>
  BasedOnVersionID: string
  CreatedBy: string
}

export type EndingPlanVersion = {
  ID: string
  StoryID: string
  VersionNo: number
  Content: Record<string, unknown>
  BasedOnVersionID: string
  CreatedBy: string
}

export type StoryArc = {
  ID: string
  StoryID: string
  Ordinal: number
  Status: string
  CurrentVersionID: string
}

export type PlanningCharacter = {
  ID: string
  StoryID: string
  CanonicalName: string
  Importance: string
  CurrentProfileVersionID: string
}

export type PolicySnapshot = {
  story_id: string
  minimum_audio_duration_sec: number
  target_audio_duration_sec: number
  content_origin: string
  language: string
  narration_language: string
  policy_version: number
}

export type StoryWorkspaceSnapshot = {
  planning_mode: string
  planning_phase: string
  public_rating: string
  public_warnings: string[]
  cover_asset_id: string
}

export type ActivationReadiness = {
  ready: boolean
  missing: string[]
  generation_policy?: PolicySnapshot
  story_workspace?: StoryWorkspaceSnapshot
}

export type FoundationResult = {
  bible: StoryBibleVersion
  ending: EndingPlanVersion
  arcs: StoryArc[]
  characters: PlanningCharacter[]
}

export type ContentProfileInput = {
  maturity_target: string
  allowed_themes: string[]
  disallowed_themes: string[]
  violence_level: string
  language_limits: string
  romance_limits: string
  constraints: Record<string, unknown>
}

export type ChapterPlanRevision = {
  ID: string
  ChapterID: string
  RevisionNo: number
  Plan: Record<string, unknown>
  BasedOnRevisionID: string
  CreatedBy: string
}

export type StoryAssetResponse = {
  id: string
  story_id: string
  type: string
  storage_key: string
  mime_type: string
  size_bytes: number
  status: string
}

let accessToken: string | null = null
let refreshInFlight: Promise<string | null> | null = null

function captureAccessToken(response: TokenResponse): TokenResponse {
  accessToken = response.access_token
  return response
}

export function clearAccessToken(): void {
  accessToken = null
}

function mayRefresh(path: string): boolean {
  if (!path.startsWith('/auth/')) return true
  return (
    path === '/auth/me' ||
    path === '/auth/logout' ||
    path === '/auth/logout-all' ||
    path === '/auth/sessions' ||
    path.startsWith('/auth/sessions/') ||
    path.startsWith('/auth/mfa/') ||
    path.startsWith('/auth/account/')
  )
}

async function parseFailure(res: Response): Promise<Error> {
  let body: ApiError | null = null
  try {
    body = (await res.json()) as ApiError
  } catch {
    // Ignore non-JSON error bodies.
  }
  return new Error(body?.error?.message ?? `Request failed (${res.status})`)
}

async function refreshAccessToken(): Promise<string | null> {
  if (refreshInFlight) return refreshInFlight

  refreshInFlight = (async () => {
    const res = await fetch(`${BASE}/auth/refresh`, {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
    })
    if (!res.ok) {
      accessToken = null
      return null
    }
    const response = (await res.json()) as TokenResponse
    accessToken = response.access_token
    return accessToken
  })().finally(() => {
    refreshInFlight = null
  })

  return refreshInFlight
}

async function request<T>(path: string, init?: RequestInit, retryAuth = true): Promise<T> {
  const headers = new Headers(init?.headers)
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData
  if (init?.body != null && !isFormData && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  if (accessToken) {
    headers.set('Authorization', `Bearer ${accessToken}`)
  }

  const res = await fetch(`${BASE}${path}`, {
    ...init,
    credentials: init?.credentials ?? 'same-origin',
    headers,
  })

  if (res.status === 401 && retryAuth && mayRefresh(path)) {
    const refreshed = await refreshAccessToken()
    if (refreshed) {
      return request<T>(path, init, false)
    }
  }

  if (!res.ok) {
    throw await parseFailure(res)
  }

  return (await res.json()) as T
}

export function listGenres(): Promise<GenreListResponse> {
  return request<GenreListResponse>('/genres')
}

export function listPublicStories(params?: {
  q?: string
  genre?: string
  sort?: string
}): Promise<StoryListResponse> {
  const qs = new URLSearchParams()
  if (params?.q) qs.set('q', params.q)
  if (params?.genre) qs.set('genre', params.genre)
  if (params?.sort) qs.set('sort', params.sort)

  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return request<StoryListResponse>(`/stories${suffix}`)
}

export function listAdminStories(): Promise<StoryListResponse> {
  return request<StoryListResponse>('/admin/stories')
}

export function getPublicStory(storyID: string): Promise<Story> {
  return request<Story>(`/stories/${storyID}`)
}

export function createStory(input: CreateStoryInput): Promise<Story> {
  return request<Story>('/admin/stories', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function updateStoryMetadata(
  storyID: string,
  input: { title: string; description: string },
): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/metadata`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function uploadStoryCover(storyID: string, file: File): Promise<StoryAssetResponse> {
  const body = new FormData()
  body.append('file', file)
  return request<StoryAssetResponse>(`/admin/stories/${storyID}/cover`, {
    method: 'POST',
    body,
  })
}

export function getStoryWorkflowSettings(storyID: string): Promise<StoryWorkflowSettings> {
  return request<StoryWorkflowSettings>(`/admin/stories/${storyID}/workflow-settings`)
}

export function updateStoryWorkflowSettings(
  storyID: string,
  input: StoryWorkflowSettings,
): Promise<StoryWorkflowSettings> {
  return request<StoryWorkflowSettings>(`/admin/stories/${storyID}/workflow-settings`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function getStoryContentProfile(storyID: string): Promise<StoryContentProfile> {
  return request<StoryContentProfile>(`/admin/stories/${storyID}/content-profile`)
}

export function createStoryContentProfile(
  storyID: string,
  input: ContentProfileInput,
): Promise<StoryContentProfile> {
  return request<StoryContentProfile>(`/admin/stories/${storyID}/content-profile`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function generateStoryFoundation(storyID: string, premise: string): Promise<FoundationResult> {
  return request<FoundationResult>(`/admin/stories/${storyID}/foundation`, {
    method: 'POST',
    body: JSON.stringify({ premise }),
  })
}

export function getStoryBible(storyID: string): Promise<StoryBibleVersion> {
  return request<StoryBibleVersion>(`/admin/stories/${storyID}/bible`)
}

export function createStoryBibleVersion(
  storyID: string,
  content: Record<string, unknown>,
): Promise<StoryBibleVersion> {
  return request<StoryBibleVersion>(`/admin/stories/${storyID}/bible/versions`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export function getStoryEnding(storyID: string): Promise<EndingPlanVersion> {
  return request<EndingPlanVersion>(`/admin/stories/${storyID}/ending`)
}

export function createStoryEndingVersion(
  storyID: string,
  content: Record<string, unknown>,
): Promise<EndingPlanVersion> {
  return request<EndingPlanVersion>(`/admin/stories/${storyID}/ending/versions`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export function listStoryArcs(storyID: string): Promise<{ arcs: StoryArc[] }> {
  return request<{ arcs: StoryArc[] }>(`/admin/stories/${storyID}/arcs`)
}

export function createStoryArc(storyID: string, content: Record<string, unknown>): Promise<StoryArc> {
  return request<StoryArc>(`/admin/stories/${storyID}/arcs`, {
    method: 'POST',
    body: JSON.stringify({ content }),
  })
}

export function listStoryCharacters(storyID: string): Promise<{ characters: PlanningCharacter[] }> {
  return request<{ characters: PlanningCharacter[] }>(`/admin/stories/${storyID}/characters`)
}

export function createPlanningCharacter(
  storyID: string,
  input: { name: string; importance: string; profile: Record<string, unknown> },
): Promise<PlanningCharacter> {
  return request<PlanningCharacter>(`/admin/stories/${storyID}/characters`, {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function getActivationReadiness(storyID: string): Promise<ActivationReadiness> {
  return request<ActivationReadiness>(`/admin/stories/${storyID}/activation-readiness`)
}

export function activateStory(storyID: string): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/activate`, { method: 'POST' })
}

export function archiveStory(storyID: string): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/archive`, { method: 'POST' })
}

export function restoreStory(storyID: string): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/restore`, { method: 'POST' })
}

export function makeStoryPublic(storyID: string): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/make-public`, { method: 'POST' })
}

export function makeStoryPrivate(storyID: string): Promise<Story> {
  return request<Story>(`/admin/stories/${storyID}/make-private`, { method: 'POST' })
}

export async function login(email: string, password: string): Promise<TokenResponse> {
  const response = await request<TokenResponse>(
    '/auth/login',
    {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    },
    false,
  )
  return captureAccessToken(response)
}

export function getCurrentUser(): Promise<AuthUser> {
  return request<AuthUser>('/auth/me')
}

export async function refreshSession(): Promise<TokenResponse> {
  const response = await request<TokenResponse>('/auth/refresh', { method: 'POST' }, false)
  return captureAccessToken(response)
}

export async function logout(): Promise<{ status: string }> {
  try {
    return await request<{ status: string }>('/auth/logout', { method: 'POST' })
  } finally {
    clearAccessToken()
  }
}

export async function logoutAll(): Promise<{ status: string }> {
  try {
    return await request<{ status: string }>('/auth/logout-all', { method: 'POST' })
  } finally {
    clearAccessToken()
  }
}

export function listAuthSessions(): Promise<{ items: AuthSession[] }> {
  return request<{ items: AuthSession[] }>('/auth/sessions')
}

export function revokeAuthSession(sessionID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/auth/sessions/${sessionID}`, { method: 'DELETE' })
}

export function setupTOTP(): Promise<{ secret: string }> {
  return request<{ secret: string }>('/auth/mfa/totp/setup', {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function confirmTOTP(code: string): Promise<{ recovery_codes: string[] }> {
  return request<{ recovery_codes: string[] }>('/auth/mfa/totp/confirm', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

export function disableTOTP(): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/mfa/totp/disable', {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function requestAccountDeletion(): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/account/deletion/request', {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function cancelAccountDeletion(): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/account/deletion/cancel', {
    method: 'POST',
    body: JSON.stringify({}),
  })
}

export function register(email: string, password: string): Promise<{ id: string; email: string; status: string }> {
  return request<{ id: string; email: string; status: string }>(
    '/auth/register',
    {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    },
    false,
  )
}

export function requestPasswordReset(email: string): Promise<{ status: string }> {
  return request<{ status: string }>(
    '/auth/password/forgot',
    {
      method: 'POST',
      body: JSON.stringify({ email }),
    },
    false,
  )
}

export function resetPassword(
  email: string,
  token: string,
  newPassword: string,
): Promise<{ status: string }> {
  return request<{ status: string }>(
    '/auth/password/reset',
    {
      method: 'POST',
      body: JSON.stringify({ email, token, new_password: newPassword }),
    },
    false,
  )
}

export function verifyEmail(email: string, token: string): Promise<{ status: string }> {
  return request<{ status: string }>(
    '/auth/email/verify',
    {
      method: 'POST',
      body: JSON.stringify({ email, token }),
    },
    false,
  )
}

export function resendEmailVerification(email: string): Promise<{ status: string }> {
  return request<{ status: string }>(
    '/auth/email/resend',
    {
      method: 'POST',
      body: JSON.stringify({ email }),
    },
    false,
  )
}

export function listPublishedChapters(storyID: string): Promise<ChapterListResponse> {
  return request<ChapterListResponse>(`/stories/${storyID}/chapters`)
}

export function listAdminChapters(storyID: string): Promise<ChapterListResponse> {
  return request<ChapterListResponse>(`/admin/stories/${storyID}/chapters`)
}

export function createPlanningChapter(storyID: string, title: string): Promise<Chapter> {
  return request<Chapter>(`/admin/stories/${storyID}/chapters`, {
    method: 'POST',
    body: JSON.stringify({ title }),
  })
}

export function createChapterPlanRevision(
  chapterID: string,
  plan: Record<string, unknown>,
): Promise<ChapterPlanRevision> {
  return request<ChapterPlanRevision>(`/admin/chapters/${chapterID}/plans`, {
    method: 'POST',
    body: JSON.stringify({ plan }),
  })
}

export function listContentRevisions(chapterID: string): Promise<ContentRevisionListResponse> {
  return request<ContentRevisionListResponse>(`/admin/chapters/${chapterID}/content`)
}

export function listChapterReviews(chapterID: string): Promise<ChapterReviewListResponse> {
  return request<ChapterReviewListResponse>(`/admin/chapters/${chapterID}/reviews`)
}

export function editContent(
  chapterID: string,
  basedOnRevisionID: string,
  text: string,
): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/chapters/${chapterID}/edit`, {
    method: 'POST',
    body: JSON.stringify({
      based_on_revision_id: basedOnRevisionID,
      text,
    }),
  })
}

export function regenerateContent(
  chapterID: string,
  basedOnRevisionID: string,
): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/chapters/${chapterID}/regenerate`, {
    method: 'POST',
    body: JSON.stringify({
      based_on_revision_id: basedOnRevisionID,
    }),
  })
}

export function approveContent(
  chapterID: string,
  revisionID: string,
): Promise<Record<string, unknown>> {
  return request<Record<string, unknown>>(`/admin/chapters/${chapterID}/approve`, {
    method: 'POST',
    body: JSON.stringify({ revision_id: revisionID }),
  })
}

export function rejectContent(
  revisionID: string,
  reason: string,
): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/revisions/${revisionID}/reject`, {
    method: 'POST',
    body: JSON.stringify({ reason }),
  })
}

export function runContentReview(
  chapterID: string,
  reviewType: 'CONTINUITY' | 'QUALITY' | 'SAFETY',
  revisionID: string,
  text: string,
): Promise<ChapterReview> {
  const paths = {
    CONTINUITY: 'continuity',
    QUALITY: 'quality',
    SAFETY: 'safety',
  } as const
  return request<ChapterReview>(`/admin/chapters/${chapterID}/${paths[reviewType]}`, {
    method: 'POST',
    body: JSON.stringify({ revision_id: revisionID, text }),
  })
}

export function getChapterContent(chapterID: string): Promise<ChapterContent> {
  return request<ChapterContent>(`/chapters/${chapterID}/content`)
}

export function getAudioURL(chapterID: string): Promise<AudioURLResponse> {
  return request<AudioURLResponse>(`/chapters/${chapterID}/audio-url`)
}

export function listFavorites(): Promise<FavoriteListResponse> {
  return request<FavoriteListResponse>('/me/favorites')
}

export function addFavorite(storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, {
    method: 'PUT',
  })
}

export function removeFavorite(storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, {
    method: 'DELETE',
  })
}

export function getProgress(chapterID: string): Promise<ListeningProgress> {
  return request<ListeningProgress>(`/me/progress/${chapterID}`)
}

export function saveProgress(
  chapterID: string,
  input: { position_ms: number; audio_asset_id: string; playback_session_id: string },
): Promise<ListeningProgress> {
  return request<ListeningProgress>(`/me/progress/${chapterID}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  })
}

export function listCreativeDecisions(storyID: string): Promise<CreativeDecisionListResponse> {
  return request<CreativeDecisionListResponse>(`/admin/stories/${storyID}/creative-decisions`)
}

export function listAttentionItems(storyID: string): Promise<AttentionListResponse> {
  return request<AttentionListResponse>(`/admin/stories/${storyID}/attention`)
}

export function reviewArcCompletion(storyID: string, arcID: string): Promise<ArcCompletionResult> {
  return request<ArcCompletionResult>(`/admin/stories/${storyID}/arcs/${arcID}/completion`)
}

export function analyzeThreadInactivity(storyID: string): Promise<ThreadInactivityResponse> {
  return request<ThreadInactivityResponse>(`/admin/stories/${storyID}/thread-inactivity`)
}

export function listUsage(storyID: string): Promise<UsageListResponse> {
  return request<UsageListResponse>(`/admin/stories/${storyID}/usage`)
}

export function listAuditEvents(filters: AuditFilters = {}): Promise<AuditListResponse> {
  const qs = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      qs.set(key, String(value))
    }
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return request<AuditListResponse>(`/admin/audit${suffix}`)
}

export function getAuditEvent(eventID: string): Promise<AuditEvent> {
  return request<AuditEvent>(`/admin/audit/${eventID}`)
}
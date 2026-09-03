import type {
  ApiError,
  AuditEvent,
  AuditListResponse,
  AuthUser,
  ArcCompletionResult,
  AttentionListResponse,
  AudioURLResponse,
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
  if (init?.body != null && !headers.has('Content-Type')) {
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

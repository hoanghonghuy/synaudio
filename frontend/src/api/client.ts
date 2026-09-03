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
  ListenerLibrary,
  ListeningProgress,
  Story,
  StoryListResponse,
  ThreadInactivityResponse,
  UsageListResponse,
} from './types'
import { ApiRequestError } from './http-error.ts'

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

async function parseFailure(res: Response): Promise<ApiRequestError> {
  let body: ApiError | null = null
  try {
    body = (await res.json()) as ApiError
  } catch {
    // Ignore non-JSON error bodies while retaining the HTTP status.
  }
  return new ApiRequestError(
    res.status,
    body?.error?.message ?? `Request failed (${res.status})`,
    body?.error?.code,
  )
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

export function setupMFA(): Promise<{ secret: string; otpauth_uri: string }> {
  return request<{ secret: string; otpauth_uri: string }>('/auth/mfa/setup', { method: 'POST' })
}

export function confirmMFA(code: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/mfa/confirm', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

export function verifyMFA(code: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/mfa/verify', {
    method: 'POST',
    body: JSON.stringify({ code }),
  })
}

export function disableMFA(code: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/mfa', {
    method: 'DELETE',
    body: JSON.stringify({ code }),
  })
}

export function requestPasswordReset(email: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/password/forgot', {
    method: 'POST',
    body: JSON.stringify({ email }),
  })
}

export function resetPassword(token: string, password: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/password/reset', {
    method: 'POST',
    body: JSON.stringify({ token, password }),
  })
}

export function verifyEmail(token: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/email/verify', {
    method: 'POST',
    body: JSON.stringify({ token }),
  })
}

export function resendVerificationEmail(): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/email/resend', { method: 'POST' })
}

export function deleteAccount(password: string): Promise<{ status: string }> {
  return request<{ status: string }>('/auth/account', {
    method: 'DELETE',
    body: JSON.stringify({ password }),
  })
}

export function getStoryContent(storyID: string): Promise<ChapterListResponse> {
  return request<ChapterListResponse>(`/stories/${storyID}/chapters`)
}

export function getChapterContent(chapterID: string): Promise<ChapterContent> {
  return request<ChapterContent>(`/chapters/${chapterID}`)
}

export function getAudioURL(chapterID: string): Promise<AudioURLResponse> {
  return request<AudioURLResponse>(`/chapters/${chapterID}/audio`)
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

export function completeProgress(chapterID: string): Promise<ListeningProgress> {
  return request<ListeningProgress>(`/me/progress/${chapterID}/complete`, { method: 'POST' })
}

export function listFavorites(): Promise<FavoriteListResponse> {
  return request<FavoriteListResponse>('/me/favorites')
}

export function addFavorite(storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, { method: 'POST' })
}

export function removeFavorite(storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, { method: 'DELETE' })
}

export function getListenerLibrary(): Promise<ListenerLibrary> {
  return request<ListenerLibrary>('/me/library')
}

export function listAuditEvents(filters: AuditFilters = {}): Promise<AuditListResponse> {
  const qs = new URLSearchParams()
  for (const [key, value] of Object.entries(filters)) {
    if (value !== undefined && value !== '') qs.set(key, String(value))
  }
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return request<AuditListResponse>(`/admin/audit${suffix}`)
}

export function getAuditEvent(id: string): Promise<AuditEvent> {
  return request<AuditEvent>(`/admin/audit/${id}`)
}

export function listAttention(params?: { type?: string; status?: string }): Promise<AttentionListResponse> {
  const qs = new URLSearchParams()
  if (params?.type) qs.set('type', params.type)
  if (params?.status) qs.set('status', params.status)
  const suffix = qs.toString() ? `?${qs.toString()}` : ''
  return request<AttentionListResponse>(`/admin/attention${suffix}`)
}

export function listUsage(): Promise<UsageListResponse> {
  return request<UsageListResponse>('/admin/usage')
}

export function getThreadInactivity(): Promise<ThreadInactivityResponse> {
  return request<ThreadInactivityResponse>('/admin/threads/inactive')
}

export function listCreativeDecisions(storyID: string): Promise<CreativeDecisionListResponse> {
  return request<CreativeDecisionListResponse>(`/admin/stories/${storyID}/creative-decisions`)
}

export function getChapterReview(chapterID: string): Promise<ChapterReview> {
  return request<ChapterReview>(`/admin/chapters/${chapterID}/review`)
}

export function listChapterReviews(storyID: string): Promise<ChapterReviewListResponse> {
  return request<ChapterReviewListResponse>(`/admin/stories/${storyID}/reviews`)
}

export function listContentRevisions(chapterID: string): Promise<ContentRevisionListResponse> {
  return request<ContentRevisionListResponse>(`/admin/chapters/${chapterID}/content-revisions`)
}

export function getContentRevision(revisionID: string): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/content-revisions/${revisionID}`)
}

export function approveContentRevision(revisionID: string): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/content-revisions/${revisionID}/approve`, { method: 'POST' })
}

export function rejectContentRevision(revisionID: string): Promise<ContentRevision> {
  return request<ContentRevision>(`/admin/content-revisions/${revisionID}/reject`, { method: 'POST' })
}

export function chooseCreativeDecision(decisionID: string, optionID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/admin/creative-decisions/${decisionID}/select`, {
    method: 'POST',
    body: JSON.stringify({ option_id: optionID }),
  })
}

export function ignoreCreativeDecision(decisionID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/admin/creative-decisions/${decisionID}/ignore`, { method: 'POST' })
}

export function resolveArcCompletion(arcID: string): Promise<ArcCompletionResult> {
  return request<ArcCompletionResult>(`/admin/arcs/${arcID}/complete`, { method: 'POST' })
}

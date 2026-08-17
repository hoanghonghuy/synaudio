import type {
  ApiError,
  ArcCompletionResult,
  AttentionListResponse,
  AudioURLResponse,
  ChapterContent,
  ChapterListResponse,
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

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })

  if (!res.ok) {
    let body: ApiError | null = null
    try {
      body = (await res.json()) as ApiError
    } catch {
      // ignore non-JSON error bodies
    }
    throw new Error(body?.error?.message ?? `Request failed (${res.status})`)
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

export function createStory(input: CreateStoryInput): Promise<Story> {
  return request<Story>('/admin/stories', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export function listPublishedChapters(storyID: string): Promise<ChapterListResponse> {
  return request<ChapterListResponse>(`/stories/${storyID}/chapters`)
}

export function getChapterContent(chapterID: string): Promise<ChapterContent> {
  return request<ChapterContent>(`/chapters/${chapterID}/content`)
}

export function getAudioURL(chapterID: string): Promise<AudioURLResponse> {
  return request<AudioURLResponse>(`/chapters/${chapterID}/audio-url`)
}

export function listFavorites(userID: string): Promise<FavoriteListResponse> {
  return request<FavoriteListResponse>('/me/favorites', {
    headers: { 'X-User-ID': userID },
  })
}

export function addFavorite(userID: string, storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, {
    method: 'PUT',
    headers: { 'X-User-ID': userID },
  })
}

export function removeFavorite(userID: string, storyID: string): Promise<{ status: string }> {
  return request<{ status: string }>(`/me/favorites/${storyID}`, {
    method: 'DELETE',
    headers: { 'X-User-ID': userID },
  })
}

export function getProgress(userID: string, chapterID: string): Promise<ListeningProgress> {
  return request<ListeningProgress>(`/me/progress/${chapterID}`, {
    headers: { 'X-User-ID': userID },
  })
}

export function saveProgress(
  userID: string,
  chapterID: string,
  input: { position_ms: number; audio_asset_id: string; playback_session_id: string },
): Promise<ListeningProgress> {
  return request<ListeningProgress>(`/me/progress/${chapterID}`, {
    method: 'PUT',
    headers: { 'X-User-ID': userID },
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

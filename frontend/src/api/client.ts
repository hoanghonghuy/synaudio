import type { ApiError, CreateStoryInput, GenreListResponse, Story, StoryListResponse } from './types'

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

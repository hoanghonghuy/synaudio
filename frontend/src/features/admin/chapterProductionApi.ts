import type { ContentRevision } from '../../api/types'
import { refreshSession } from '../../api/client'

const BASE = '/api/v1'

export type GenerationRun = {
  ID: string
  RunType: string
  StoryID: string
  ChapterID: string
  Status: string
  WaitingReason: string
  WorkflowVersion: string
  Priority: number
  BaseCanonVersionID: string
  ContextSnapshotID: string
  RequestedBy: string
  IdempotencyKey: string
}

async function parseError(response: Response): Promise<Error> {
  try {
    const body = (await response.json()) as { error?: { message?: string } }
    return new Error(body.error?.message || `Request failed (${response.status})`)
  } catch {
    return new Error(`Request failed (${response.status})`)
  }
}

async function authorizedFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const session = await refreshSession()
  return fetch(`${BASE}${path}`, {
    ...init,
    credentials: 'same-origin',
    headers: {
      ...(init.body ? { 'Content-Type': 'application/json' } : {}),
      ...init.headers,
      Authorization: `Bearer ${session.access_token}`,
    },
  })
}

export async function startChapterGeneration(storyID: string, chapterID: string): Promise<GenerationRun> {
  const response = await authorizedFetch(`/admin/stories/${storyID}/batch-generate`, {
    method: 'POST',
    body: JSON.stringify({ chapter_ids: [chapterID] }),
  })
  if (!response.ok) throw await parseError(response)
  return (await response.json()) as GenerationRun
}

export async function getGenerationRun(runID: string): Promise<GenerationRun> {
  const response = await authorizedFetch(`/admin/runs/${runID}`)
  if (!response.ok) throw await parseError(response)
  return (await response.json()) as GenerationRun
}

export async function rewriteContent(
  chapterID: string,
  basedOnRevisionID: string,
  feedback: string,
): Promise<ContentRevision> {
  const response = await authorizedFetch(`/admin/chapters/${chapterID}/rewrite`, {
    method: 'POST',
    body: JSON.stringify({
      based_on_revision_id: basedOnRevisionID,
      feedback,
    }),
  })

  if (!response.ok) throw await parseError(response)
  return (await response.json()) as ContentRevision
}

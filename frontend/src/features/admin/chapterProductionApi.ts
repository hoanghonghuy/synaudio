import type { ContentRevision } from '../../api/types'
import { refreshSession } from '../../api/client'

const BASE = '/api/v1'

async function parseError(response: Response): Promise<Error> {
  try {
    const body = (await response.json()) as { error?: { message?: string } }
    return new Error(body.error?.message || `Request failed (${response.status})`)
  } catch {
    return new Error(`Request failed (${response.status})`)
  }
}

export async function rewriteContent(
  chapterID: string,
  basedOnRevisionID: string,
  feedback: string,
): Promise<ContentRevision> {
  const session = await refreshSession()
  const response = await fetch(`${BASE}/admin/chapters/${chapterID}/rewrite`, {
    method: 'POST',
    credentials: 'same-origin',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${session.access_token}`,
    },
    body: JSON.stringify({
      based_on_revision_id: basedOnRevisionID,
      feedback,
    }),
  })

  if (!response.ok) throw await parseError(response)
  return (await response.json()) as ContentRevision
}

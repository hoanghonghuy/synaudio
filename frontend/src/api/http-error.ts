export type ProgressConflictPayload = {
  UserID: string
  ChapterID: string
  PositionMs: number
  CompletedAt: string
  LastAudioAssetID: string
  LastPlaybackSessionID: string
  Version: number
  RelistenStatus?: 'NO_RELISTEN_NEEDED' | 'RELISTEN_RECOMMENDED' | 'RELISTEN_REQUIRED'
}

export class ApiRequestError extends Error {
  readonly status: number
  readonly code?: string
  readonly progress?: ProgressConflictPayload

  constructor(status: number, message: string, code?: string, progress?: ProgressConflictPayload) {
    super(message)
    this.name = 'ApiRequestError'
    this.status = status
    this.code = code
    this.progress = progress
  }
}

export function isExplicitNotFound(error: unknown): boolean {
  return error instanceof ApiRequestError && error.status === 404
}

export function isProgressVersionConflict(error: unknown): error is ApiRequestError & { progress: ProgressConflictPayload } {
  return (
    error instanceof ApiRequestError &&
    error.status === 409 &&
    error.code === 'PROGRESS_VERSION_CONFLICT' &&
    error.progress != null
  )
}

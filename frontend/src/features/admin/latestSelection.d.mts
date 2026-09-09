export function isChapterSelectionBlockingAction(action: string): boolean

export function canSelectChapter(input: { action: string }): boolean

export function createLatestSelectionGuard(): {
  begin(key: string): () => boolean
}

export function generationRunFromContentResponse<T>(response: {
  generation_run?: T | null
}): T | null

export function generationJobFromContentResponse<T>(response: {
  generation_job?: T | null
}): T | null

export function canRetryGenerationJob(input: {
  generationJob: { Retryable?: boolean; Observation?: string } | null
  selectionLoading: boolean
  actionInProgress: boolean
}): boolean

export function formatGenerationJobStatus(
  generationJob: { Observation?: string; Status?: string; AttemptCount?: number; MaxAttempts?: number; LastErrorClass?: string; LastErrorCode?: string } | null,
): string | null

export function canStartChapterGeneration(input: {
  hasPlanRevision: boolean
  selectionLoading: boolean
  actionInProgress: boolean
  hasGenerationRun: boolean
  hasGenerationRunProvenance: boolean
}): boolean

export function canCreateNarration(input: {
  approvedRevision: { ID?: string; ContentText?: string } | null
  selectionLoading: boolean
  actionInProgress: boolean
  voiceID: string
}): boolean

export function canSynthesizeNarration(input: {
  hasApprovedContent: boolean
  hasNarration: boolean
  narrationBelongsToChapter: boolean
  selectionLoading: boolean
  actionInProgress: boolean
}): boolean

export function canActivateAudio(input: {
  hasReadyAsset: boolean
  readyAssetBelongsToChapter: boolean
  readyAssetIsInactive: boolean
  readyAssetMatchesLatestNarration: boolean
  selectionLoading: boolean
  actionInProgress: boolean
}): boolean

export function canMarkChapterReady(input: {
  chapterStatus?: string
  publishReadiness: { ready?: boolean } | null
  selectionLoading: boolean
  actionInProgress: boolean
}): boolean

export function canPublishChapter(input: {
  chapterStatus?: string
  publishReadiness: { ready?: boolean } | null
  selectionLoading: boolean
  actionInProgress: boolean
}): boolean

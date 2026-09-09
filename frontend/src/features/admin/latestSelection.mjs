const CHAPTER_SELECTION_BLOCKING_ACTIONS = new Set([
  'start-generation',
  'retry-generation',
  'create-narration',
  'synthesize',
  'activate',
  'mark-ready',
  'publish',
])

export function isChapterSelectionBlockingAction(action) {
  return CHAPTER_SELECTION_BLOCKING_ACTIONS.has(action)
}

export function canSelectChapter({ action }) {
  return !isChapterSelectionBlockingAction(action)
}

export function createLatestSelectionGuard() {
  let generation = 0
  return {
    begin(key) {
      generation += 1
      const requestGeneration = generation
      return () => requestGeneration === generation && key != null
    },
  }
}

export function generationRunFromContentResponse(response) {
  return response?.generation_run ?? null
}

export function generationJobFromContentResponse(response) {
  return response?.generation_job ?? null
}

export function canRetryGenerationJob({
  generationJob,
  selectionLoading,
  actionInProgress,
}) {
  return Boolean(
    generationJob?.Retryable === true
    && generationJob?.Observation === 'retryable'
    && !selectionLoading
    && !actionInProgress,
  )
}

export function formatGenerationJobStatus(generationJob) {
  if (!generationJob) return null
  const attempt = `${generationJob.AttemptCount}/${generationJob.MaxAttempts}`
  switch (generationJob.Observation) {
    case 'queued':
      return `QUEUED · attempt ${attempt}`
    case 'running':
      return `RUNNING · attempt ${attempt}`
    case 'succeeded':
      return `SUCCEEDED · attempt ${attempt}`
    case 'retryable':
      return `RETRYABLE · ${generationJob.LastErrorClass || 'TRANSIENT'} · attempt ${attempt}`
    case 'exhausted':
      return `EXHAUSTED · ${generationJob.LastErrorCode || generationJob.LastErrorClass || 'MAX_ATTEMPTS'} · attempt ${attempt}`
    case 'failed':
      return `FAILED · ${generationJob.LastErrorClass || 'PERMANENT'} · attempt ${attempt}`
    default:
      return `${generationJob.Status} · attempt ${attempt}`
  }
}

export function canStartChapterGeneration({
  hasPlanRevision,
  selectionLoading,
  actionInProgress,
  hasGenerationRun,
  hasGenerationRunProvenance,
}) {
  return Boolean(
    hasPlanRevision
    && !selectionLoading
    && !actionInProgress
    && !hasGenerationRun
    && !hasGenerationRunProvenance,
  )
}

export function canCreateNarration({
  approvedRevision,
  selectionLoading,
  actionInProgress,
  voiceID,
}) {
  return Boolean(
    approvedRevision?.ID
    && approvedRevision.ContentText?.trim()
    && voiceID
    && !selectionLoading
    && !actionInProgress,
  )
}

export function canSynthesizeNarration({
  hasApprovedContent,
  hasNarration,
  narrationBelongsToChapter,
  selectionLoading,
  actionInProgress,
}) {
  return Boolean(
    hasApprovedContent
    && hasNarration
    && narrationBelongsToChapter
    && !selectionLoading
    && !actionInProgress,
  )
}

export function canActivateAudio({
  hasReadyAsset,
  readyAssetBelongsToChapter,
  readyAssetIsInactive,
  readyAssetMatchesLatestNarration,
  selectionLoading,
  actionInProgress,
}) {
  return Boolean(
    hasReadyAsset
    && readyAssetBelongsToChapter
    && readyAssetIsInactive
    && readyAssetMatchesLatestNarration
    && !selectionLoading
    && !actionInProgress,
  )
}

export function canMarkChapterReady({
  chapterStatus,
  publishReadiness,
  selectionLoading,
  actionInProgress,
}) {
  return Boolean(
    chapterStatus
    && chapterStatus !== 'READY'
    && chapterStatus !== 'PUBLISHED'
    && publishReadiness?.ready === true
    && !selectionLoading
    && !actionInProgress,
  )
}

export function canPublishChapter({
  chapterStatus,
  publishReadiness,
  selectionLoading,
  actionInProgress,
}) {
  return Boolean(
    chapterStatus === 'READY'
    && publishReadiness?.ready === true
    && !selectionLoading
    && !actionInProgress,
  )
}

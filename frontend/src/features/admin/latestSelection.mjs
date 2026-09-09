const CHAPTER_SELECTION_BLOCKING_ACTIONS = new Set([
  'start-generation',
  'create-narration',
  'synthesize',
  'activate',
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

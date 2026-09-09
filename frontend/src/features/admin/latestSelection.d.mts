export function createLatestSelectionGuard(): {
  begin(key: string): () => boolean
}

export function generationRunFromContentResponse<T>(response: {
  generation_run?: T | null
}): T | null

export function canStartChapterGeneration(input: {
  hasPlanRevision: boolean
  selectionLoading: boolean
  actionInProgress: boolean
  hasGenerationRun: boolean
  hasGenerationRunProvenance: boolean
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

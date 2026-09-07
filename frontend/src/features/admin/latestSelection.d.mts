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

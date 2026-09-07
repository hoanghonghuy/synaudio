export function createLatestSelectionGuard(): {
  begin(key: string): () => boolean
}

export function canStartChapterGeneration(input: {
  hasPlanRevision: boolean
  selectionLoading: boolean
  actionInProgress: boolean
  hasGenerationRun: boolean
  hasGenerationRunProvenance: boolean
}): boolean

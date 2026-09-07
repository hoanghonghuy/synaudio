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

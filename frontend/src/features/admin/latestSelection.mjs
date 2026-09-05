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

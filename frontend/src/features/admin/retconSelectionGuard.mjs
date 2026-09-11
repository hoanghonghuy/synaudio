export function createRetconSelectionGuard() {
  let generation = 0
  let selectedID = ''

  return {
    select(id) {
      selectedID = id
      generation += 1
      return { id, generation }
    },
    clear() {
      selectedID = ''
      generation += 1
    },
    isCurrent(request) {
      return Boolean(request) && request.id === selectedID && request.generation === generation
    },
  }
}

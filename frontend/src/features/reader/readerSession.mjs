export function createLatestChapterSelectionGuard() {
  let generation = 0

  return {
    begin(chapterID) {
      generation += 1
      const requestGeneration = generation
      return () => requestGeneration === generation && Boolean(chapterID)
    },
  }
}

export function normalizePlaybackRate(value) {
  const rate = Number(value)
  const supported = [0.75, 1, 1.25, 1.5, 1.75, 2]
  return supported.includes(rate) ? rate : 1
}

export function formatPlaybackTime(seconds) {
  const safeSeconds = Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 0
  const minutes = Math.floor(safeSeconds / 60)
  const remainder = safeSeconds % 60
  return `${minutes}:${String(remainder).padStart(2, '0')}`
}

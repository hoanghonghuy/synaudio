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

export function normalizeVolume(value, fallback = 0.9) {
  const fallbackVolume = Number(fallback)
  const safeFallback = Number.isFinite(fallbackVolume)
    ? Math.min(1, Math.max(0, fallbackVolume))
    : 0.9
  const volume = Number(value)
  return Number.isFinite(volume) ? Math.min(1, Math.max(0, volume)) : safeFallback
}

export function toggleMuteState(isMuted, volume, lastAudibleVolume = 0.9) {
  const currentVolume = normalizeVolume(volume)
  const rememberedVolume = normalizeVolume(lastAudibleVolume)

  if (isMuted || currentVolume === 0) {
    return {
      muted: false,
      volume: rememberedVolume,
      lastAudibleVolume: rememberedVolume,
    }
  }

  return {
    muted: true,
    volume: 0,
    lastAudibleVolume: currentVolume,
  }
}

export function formatPlaybackTime(seconds) {
  const safeSeconds = Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 0
  const hours = Math.floor(safeSeconds / 3600)
  const minutes = Math.floor(safeSeconds / 60) % 60
  const remainder = safeSeconds % 60
  if (hours > 0) return `${hours}:${String(minutes).padStart(2, '0')}:${String(remainder).padStart(2, '0')}`
  return `${minutes}:${String(remainder).padStart(2, '0')}`
}

export function formatTimelineTime(seconds) {
  const safeSeconds = Number.isFinite(seconds) && seconds > 0 ? Math.floor(seconds) : 0
  const hours = Math.floor(safeSeconds / 3600)
  const minutes = Math.floor(safeSeconds / 60) % 60
  const remainder = safeSeconds % 60
  if (hours > 0) return `${hours}:${String(minutes).padStart(2, '0')}:${String(remainder).padStart(2, '0')}`
  return `${String(minutes).padStart(2, '0')}:${String(remainder).padStart(2, '0')}`
}

export { formatChapterTitle } from '../admin/reviewPresentation.mjs'


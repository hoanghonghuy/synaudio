export function createLatestChapterSelectionGuard(): {
  begin(chapterID: string): () => boolean
}

export function normalizePlaybackRate(value: unknown): number
export function normalizeVolume(value: unknown, fallback?: number): number
export function toggleMuteState(
  isMuted: boolean,
  volume: number,
  lastAudibleVolume?: number,
): { muted: boolean; volume: number; lastAudibleVolume: number }
export function formatPlaybackTime(seconds: number): string

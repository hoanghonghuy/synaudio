export function createLatestChapterSelectionGuard(): {
  begin(chapterID: string): () => boolean
}

export function normalizePlaybackRate(value: unknown): number
export function formatPlaybackTime(seconds: number): string

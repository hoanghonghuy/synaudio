export function createLatestSelectionGuard(): {
  begin(key: string): () => boolean
}

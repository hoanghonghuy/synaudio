export type RetconSelectionRequest = { id: string; generation: number }

export type RetconSelectionGuard = {
  select(id: string): RetconSelectionRequest
  clear(): void
  isCurrent(request: RetconSelectionRequest): boolean
}

export function createRetconSelectionGuard(): RetconSelectionGuard

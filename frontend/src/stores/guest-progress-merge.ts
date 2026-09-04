import { isExplicitNotFound } from '../api/http-error.ts'

export type GuestMergeOutcome = 'server-wins' | 'imported' | 'deferred'

export async function mergeGuestProgressIfAbsent(
  loadServerProgress: () => Promise<unknown>,
  importGuestProgress: () => Promise<unknown>,
): Promise<GuestMergeOutcome> {
  try {
    await loadServerProgress()
    return 'server-wins'
  } catch (error) {
    if (!isExplicitNotFound(error)) {
      return 'deferred'
    }
  }

  await importGuestProgress()
  return 'imported'
}

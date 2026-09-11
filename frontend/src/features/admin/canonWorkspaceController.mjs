import { createCanonAuthorityState } from './canonAuthorityState.mjs'
import { deriveCanonReadiness } from './canonReadiness.mjs'

function messageFrom(error, fallback) {
  return error instanceof Error && error.message ? error.message : fallback
}

/**
 * Orchestrates the Chapter Production Canon/Memory authority lifecycle.
 *
 * The API object is the authenticated Canon boundary from canonApiBoundary.mjs.
 * This controller deliberately owns no auth/role truth; it only coordinates
 * exact-selection loading, commit idempotency and authoritative refresh.
 */
export function createCanonWorkspaceController(api) {
  if (!api || typeof api.getActiveOfficialBranch !== 'function' || typeof api.listVersions !== 'function' || typeof api.commitApprovedRevision !== 'function') {
    throw new Error('Canon API boundary is required')
  }

  const authority = createCanonAuthorityState()
  let selection = null

  function snapshot() {
    const state = authority.snapshot()
    return {
      ...state,
      readiness: deriveCanonReadiness({
        selectionLoading: state.loading,
        approvedRevision: selection?.approvedRevision ?? null,
        branch: state.branch,
        versions: state.versions,
        authorityError: state.error,
        actionInProgress: state.committing,
      }),
    }
  }

  async function load(nextSelection) {
    selection = nextSelection
      ? {
          storyID: String(nextSelection.storyID ?? '').trim(),
          chapterID: String(nextSelection.chapterID ?? '').trim(),
          approvedRevision: nextSelection.approvedRevision ?? null,
        }
      : null

    if (!selection?.storyID || !selection?.chapterID) {
      authority.reset()
      return snapshot()
    }

    const request = authority.beginSelection({
      storyID: selection.storyID,
      chapterID: selection.chapterID,
      approvedRevisionID: selection.approvedRevision?.ID ?? '',
    })

    try {
      const branch = await api.getActiveOfficialBranch(selection.storyID)
      if (!request.isCurrent()) return snapshot()
      const versionResponse = await api.listVersions(branch.ID)
      const versions = Array.isArray(versionResponse) ? versionResponse : (versionResponse?.versions ?? [])
      request.succeed(branch, versions)
    } catch (error) {
      request.fail(messageFrom(error, 'Không thể tải Canon/Memory authority.'))
    }

    return snapshot()
  }

  async function commit() {
    if (!selection?.approvedRevision) return snapshot()
    const request = authority.beginCommit()
    if (!request) return snapshot()

    const branchID = authority.snapshot().branch?.ID ?? ''
    try {
      const version = await api.commitApprovedRevision({
        storyID: selection.storyID,
        branchID,
        chapterID: selection.chapterID,
        approvedRevisionID: selection.approvedRevision.ID,
      })
      if (!request.isCurrent()) return snapshot()
      request.succeed(version)
      // Do not trust the mutation response as final projection truth. Refresh the
      // branch/version authority and re-derive CURRENT from persisted provenance.
      return await load(selection)
    } catch (error) {
      request.fail(messageFrom(error, 'Không thể commit Canon/Memory.'))
      return snapshot()
    }
  }

  function reset() {
    selection = null
    authority.reset()
    return snapshot()
  }

  return { snapshot, load, commit, reset }
}

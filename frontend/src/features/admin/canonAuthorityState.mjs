import { canonCommitIdentity } from './canonReadiness.mjs'

export function createCanonAuthorityState() {
  let generation = 0
  let currentSelection = null
  let activeCommitIdentity = ''
  let state = emptySnapshot()

  function emptySnapshot() {
    return {
      loading: false,
      branch: null,
      versions: [],
      error: '',
      committing: false,
    }
  }

  function snapshot() {
    return {
      loading: state.loading,
      branch: state.branch,
      versions: [...state.versions],
      error: state.error,
      committing: state.committing,
    }
  }

  function beginSelection({ storyID, chapterID, approvedRevisionID = '' }) {
    generation += 1
    const requestGeneration = generation
    currentSelection = { storyID, chapterID, approvedRevisionID }
    activeCommitIdentity = ''
    state = { ...emptySnapshot(), loading: true }

    const isCurrent = () => requestGeneration === generation
      && currentSelection?.storyID === storyID
      && currentSelection?.chapterID === chapterID
      && currentSelection?.approvedRevisionID === approvedRevisionID

    return {
      isCurrent,
      succeed(branch, versions = []) {
        if (!isCurrent()) return snapshot()
        state = { ...state, loading: false, branch: branch ?? null, versions: [...versions], error: '' }
        return snapshot()
      },
      fail(message) {
        if (!isCurrent()) return snapshot()
        state = { ...emptySnapshot(), error: message || 'Không thể tải Canon/Memory authority.' }
        return snapshot()
      },
    }
  }

  function beginCommit() {
    if (!currentSelection || !state.branch || state.loading || state.committing) return null

    const identity = canonCommitIdentity({
      storyID: currentSelection.storyID,
      chapterID: currentSelection.chapterID,
      approvedRevisionID: currentSelection.approvedRevisionID,
      branchID: state.branch.ID,
    })
    if (!currentSelection.approvedRevisionID || activeCommitIdentity === identity) return null

    const requestGeneration = generation
    activeCommitIdentity = identity
    state = { ...state, committing: true, error: '' }

    const isCurrent = () => requestGeneration === generation
      && activeCommitIdentity === identity
      && currentSelection != null
      && canonCommitIdentity({
        storyID: currentSelection.storyID,
        chapterID: currentSelection.chapterID,
        approvedRevisionID: currentSelection.approvedRevisionID,
        branchID: state.branch?.ID ?? '',
      }) === identity

    return {
      identity,
      isCurrent,
      succeed(version) {
        if (!isCurrent()) return snapshot()
        const versions = state.versions.filter((item) => item?.ID !== version?.ID)
        if (version) versions.push(version)
        activeCommitIdentity = ''
        state = { ...state, versions, committing: false, error: '' }
        return snapshot()
      },
      fail(message) {
        if (!isCurrent()) return snapshot()
        activeCommitIdentity = ''
        state = { ...state, committing: false, error: message || 'Không thể commit Canon/Memory.' }
        return snapshot()
      },
    }
  }

  function reset() {
    generation += 1
    currentSelection = null
    activeCommitIdentity = ''
    state = emptySnapshot()
    return snapshot()
  }

  return { snapshot, beginSelection, beginCommit, reset }
}

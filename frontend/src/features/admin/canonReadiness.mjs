export function latestOfficialCanonVersion(versions = []) {
  return [...versions]
    .filter((version) => version?.Status === 'OFFICIAL')
    .sort((a, b) => Number(b?.SequenceNo ?? 0) - Number(a?.SequenceNo ?? 0))[0] ?? null
}

export function deriveCanonReadiness({
  selectionLoading = false,
  approvedRevision = null,
  branch = null,
  versions = [],
  authorityError = '',
  actionInProgress = false,
} = {}) {
  if (selectionLoading) {
    return { state: 'LOADING', label: 'Đang tải Canon/Memory authority…', canCommit: false, latestVersion: null }
  }

  if (!approvedRevision) {
    return { state: 'WAITING', label: 'WAITING — cần approved content trước khi commit Canon/Memory', canCommit: false, latestVersion: null }
  }

  if (authorityError || !branch) {
    return {
      state: 'BLOCKED',
      label: authorityError || 'BLOCKED — thiếu ACTIVE OFFICIAL CanonBranch authority',
      canCommit: false,
      latestVersion: null,
    }
  }

  const latestVersion = latestOfficialCanonVersion(versions)
  if (latestVersion?.SourceContentRevisionID === approvedRevision.ID) {
    return {
      state: 'CURRENT',
      label: `CURRENT — Canon/Memory đã commit từ approved revision ${approvedRevision.ID}`,
      canCommit: false,
      latestVersion,
    }
  }

  return {
    state: 'READY_TO_COMMIT',
    label: latestVersion
      ? `READY TO COMMIT — approved revision ${approvedRevision.ID} mới hơn Canon source ${latestVersion.SourceContentRevisionID || '—'}`
      : `READY TO COMMIT — chưa có OFFICIAL CanonVersion cho approved revision ${approvedRevision.ID}`,
    canCommit: !actionInProgress,
    latestVersion,
  }
}

export function canonCommitIdentity({ storyID = '', chapterID = '', approvedRevisionID = '', branchID = '' } = {}) {
  return `${storyID}:${chapterID}:${approvedRevisionID}:${branchID}`
}

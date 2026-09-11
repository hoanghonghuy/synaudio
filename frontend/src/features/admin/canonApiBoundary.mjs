function requiredID(value, label) {
  const normalized = String(value ?? '').trim()
  if (!normalized) throw new Error(`${label} is required`)
  return encodeURIComponent(normalized)
}

/**
 * Builds the Chapter Production Canon/Memory API boundary on top of the
 * application's existing authenticated request pipeline.
 *
 * `request` must have the same contract as frontend/src/api/client.ts request:
 *   request(path, init?) -> Promise<decoded JSON>
 *
 * Keeping transport/auth outside this module prevents Canon/Memory from
 * inventing a second token/refresh/security boundary.
 */
export function createCanonApiBoundary(request) {
  if (typeof request !== 'function') throw new Error('authenticated request function is required')

  return {
    getActiveOfficialBranch(storyID) {
      const encodedStoryID = requiredID(storyID, 'storyID')
      return request(`/admin/stories/${encodedStoryID}/canon-branches/active-official`)
    },

    listVersions(branchID) {
      const encodedBranchID = requiredID(branchID, 'branchID')
      return request(`/admin/canon-branches/${encodedBranchID}/versions`)
    },

    commitApprovedRevision({ storyID, branchID, chapterID, approvedRevisionID } = {}) {
      const encodedBranchID = requiredID(branchID, 'branchID')
      const normalizedStoryID = String(storyID ?? '').trim()
      const normalizedChapterID = String(chapterID ?? '').trim()
      const normalizedRevisionID = String(approvedRevisionID ?? '').trim()
      if (!normalizedStoryID) throw new Error('storyID is required')
      if (!normalizedChapterID) throw new Error('chapterID is required')
      if (!normalizedRevisionID) throw new Error('approvedRevisionID is required')

      return request(`/admin/canon-branches/${encodedBranchID}/commit`, {
        method: 'POST',
        body: JSON.stringify({
          story_id: normalizedStoryID,
          source_chapter_id: normalizedChapterID,
          content_revision_id: normalizedRevisionID,
        }),
      })
    },
  }
}

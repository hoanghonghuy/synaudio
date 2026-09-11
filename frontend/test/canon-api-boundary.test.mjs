import test from 'node:test'
import assert from 'node:assert/strict'
import { createCanonApiBoundary } from '../src/features/admin/canonApiBoundary.mjs'

function captureRequests() {
  const calls = []
  const request = async (path, init) => {
    calls.push({ path, init })
    return { ok: true }
  }
  return { calls, api: createCanonApiBoundary(request) }
}

test('Canon API boundary uses authoritative active OFFICIAL branch and version routes', async () => {
  const { calls, api } = captureRequests()

  await api.getActiveOfficialBranch('story-1')
  await api.listVersions('branch-1')

  assert.deepEqual(calls, [
    { path: '/admin/stories/story-1/canon-branches/active-official', init: undefined },
    { path: '/admin/canon-branches/branch-1/versions', init: undefined },
  ])
})

test('Canon commit binds exact story, chapter and approved revision and never sends actor authority', async () => {
  const { calls, api } = captureRequests()

  await api.commitApprovedRevision({
    storyID: 'story-1',
    branchID: 'branch-1',
    chapterID: 'chapter-9',
    approvedRevisionID: 'revision-42',
  })

  assert.equal(calls.length, 1)
  assert.equal(calls[0].path, '/admin/canon-branches/branch-1/commit')
  assert.equal(calls[0].init.method, 'POST')
  assert.deepEqual(JSON.parse(calls[0].init.body), {
    story_id: 'story-1',
    source_chapter_id: 'chapter-9',
    content_revision_id: 'revision-42',
  })
  assert.equal('committed_by' in JSON.parse(calls[0].init.body), false)
})

test('Canon API boundary fails before transport when authority identity is incomplete', async () => {
  const { calls, api } = captureRequests()

  assert.throws(() => api.getActiveOfficialBranch(''), /storyID is required/)
  assert.throws(() => api.listVersions(''), /branchID is required/)
  assert.throws(
    () => api.commitApprovedRevision({ storyID: 'story-1', branchID: 'branch-1', chapterID: '', approvedRevisionID: 'revision-1' }),
    /chapterID is required/,
  )
  assert.throws(
    () => api.commitApprovedRevision({ storyID: 'story-1', branchID: 'branch-1', chapterID: 'chapter-1', approvedRevisionID: '' }),
    /approvedRevisionID is required/,
  )

  assert.equal(calls.length, 0)
})

test('Canon API boundary safely encodes path identities', async () => {
  const { calls, api } = captureRequests()
  await api.getActiveOfficialBranch('story/with space')
  await api.listVersions('branch/with space')

  assert.equal(calls[0].path, '/admin/stories/story%2Fwith%20space/canon-branches/active-official')
  assert.equal(calls[1].path, '/admin/canon-branches/branch%2Fwith%20space/versions')
})

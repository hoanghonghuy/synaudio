import test from 'node:test'
import assert from 'node:assert/strict'
import { canonCommitIdentity, deriveCanonReadiness, latestOfficialCanonVersion } from '../src/features/admin/canonReadiness.mjs'

test('WAITING until an approved revision exists', () => {
  assert.deepEqual(deriveCanonReadiness({ branch: { ID: 'b1' }, versions: [] }), {
    state: 'WAITING',
    label: 'WAITING — cần approved content trước khi commit Canon/Memory',
    canCommit: false,
    latestVersion: null,
  })
})

test('BLOCKED without authoritative active official branch', () => {
  const state = deriveCanonReadiness({ approvedRevision: { ID: 'r1' }, authorityError: 'BLOCKED — canon authority ambiguous' })
  assert.equal(state.state, 'BLOCKED')
  assert.equal(state.canCommit, false)
  assert.match(state.label, /ambiguous/)
})

test('CURRENT only when latest OFFICIAL version provenance matches exact approved revision', () => {
  const versions = [
    { ID: 'v1', SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: 'r0' },
    { ID: 'draft', SequenceNo: 99, Status: 'DRAFT', SourceContentRevisionID: 'r1' },
    { ID: 'v2', SequenceNo: 2, Status: 'OFFICIAL', SourceContentRevisionID: 'r1' },
  ]
  assert.equal(latestOfficialCanonVersion(versions).ID, 'v2')
  const state = deriveCanonReadiness({ approvedRevision: { ID: 'r1' }, branch: { ID: 'b1' }, versions })
  assert.equal(state.state, 'CURRENT')
  assert.equal(state.canCommit, false)
})

test('READY_TO_COMMIT when exact approved revision differs from committed provenance', () => {
  const state = deriveCanonReadiness({
    approvedRevision: { ID: 'r2' },
    branch: { ID: 'b1' },
    versions: [{ ID: 'v1', SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: 'r1' }],
  })
  assert.equal(state.state, 'READY_TO_COMMIT')
  assert.equal(state.canCommit, true)
  assert.match(state.label, /r2/)
  assert.match(state.label, /r1/)
})

test('action in progress prevents duplicate commit eligibility', () => {
  const state = deriveCanonReadiness({
    approvedRevision: { ID: 'r2' },
    branch: { ID: 'b1' },
    versions: [],
    actionInProgress: true,
  })
  assert.equal(state.state, 'READY_TO_COMMIT')
  assert.equal(state.canCommit, false)
})

test('commit identity binds story chapter approved revision and branch', () => {
  assert.equal(canonCommitIdentity({ storyID: 's1', chapterID: 'c1', approvedRevisionID: 'r1', branchID: 'b1' }), 's1:c1:r1:b1')
  assert.notEqual(
    canonCommitIdentity({ storyID: 's1', chapterID: 'c1', approvedRevisionID: 'r1', branchID: 'b1' }),
    canonCommitIdentity({ storyID: 's1', chapterID: 'c1', approvedRevisionID: 'r2', branchID: 'b1' }),
  )
})

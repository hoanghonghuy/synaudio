import test from 'node:test'
import assert from 'node:assert/strict'
import { createCanonAuthorityState } from '../src/features/admin/canonAuthorityState.mjs'

const branchA = { ID: 'branch-a', StoryID: 'story-1', Type: 'OFFICIAL', Status: 'ACTIVE' }
const branchB = { ID: 'branch-b', StoryID: 'story-1', Type: 'OFFICIAL', Status: 'ACTIVE' }

test('ignores stale A response after A → B → A selection cycle', () => {
  const state = createCanonAuthorityState()
  const firstA = state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-a', approvedRevisionID: 'rev-a1' })
  const requestB = state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-b', approvedRevisionID: 'rev-b1' })
  const secondA = state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-a', approvedRevisionID: 'rev-a2' })

  assert.equal(firstA.isCurrent(), false)
  assert.equal(requestB.isCurrent(), false)
  assert.equal(secondA.isCurrent(), true)

  firstA.succeed(branchA, [{ ID: 'stale-a', Status: 'OFFICIAL', SourceContentRevisionID: 'rev-a1' }])
  assert.deepEqual(state.snapshot(), {
    loading: true,
    branch: null,
    versions: [],
    error: '',
    committing: false,
  })

  secondA.succeed(branchA, [{ ID: 'current-a', Status: 'OFFICIAL', SourceContentRevisionID: 'rev-a2' }])
  assert.equal(state.snapshot().versions[0].ID, 'current-a')
})

test('selection change invalidates an in-flight commit result', () => {
  const state = createCanonAuthorityState()
  state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-a', approvedRevisionID: 'rev-a1' }).succeed(branchA, [])
  const commit = state.beginCommit()
  assert.ok(commit)
  assert.equal(commit.identity, 'story-1:chapter-a:rev-a1:branch-a')

  state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-b', approvedRevisionID: 'rev-b1' }).succeed(branchB, [])
  assert.equal(commit.isCurrent(), false)
  commit.succeed({ ID: 'stale-version', Status: 'OFFICIAL', SourceContentRevisionID: 'rev-a1' })

  assert.deepEqual(state.snapshot().versions, [])
  assert.equal(state.snapshot().branch.ID, 'branch-b')
})

test('prevents duplicate commit for the same authority identity while in flight', () => {
  const state = createCanonAuthorityState()
  state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-a', approvedRevisionID: 'rev-a1' }).succeed(branchA, [])

  const first = state.beginCommit()
  assert.ok(first)
  assert.equal(state.snapshot().committing, true)
  assert.equal(state.beginCommit(), null)

  first.fail('provider unavailable')
  assert.equal(state.snapshot().committing, false)
  assert.equal(state.snapshot().error, 'provider unavailable')
  assert.ok(state.beginCommit(), 'retry is allowed only after the prior request settles')
})

test('successful commit updates versions only for exact selected authority', () => {
  const state = createCanonAuthorityState()
  state.beginSelection({ storyID: 'story-1', chapterID: 'chapter-a', approvedRevisionID: 'rev-a1' }).succeed(branchA, [
    { ID: 'v1', Status: 'OFFICIAL', SourceContentRevisionID: 'rev-old' },
  ])

  const commit = state.beginCommit()
  const snapshot = commit.succeed({ ID: 'v2', Status: 'OFFICIAL', SourceContentRevisionID: 'rev-a1' })
  assert.equal(snapshot.committing, false)
  assert.deepEqual(snapshot.versions.map((version) => version.ID), ['v1', 'v2'])
})

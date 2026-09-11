import test from 'node:test'
import assert from 'node:assert/strict'
import { createCanonWorkspaceController } from '../src/features/admin/canonWorkspaceController.mjs'

function deferred() {
  let resolve
  let reject
  const promise = new Promise((res, rej) => { resolve = res; reject = rej })
  return { promise, resolve, reject }
}

test('loads ACTIVE OFFICIAL authority and derives readiness for exact approved revision', async () => {
  const api = {
    async getActiveOfficialBranch(storyID) {
      assert.equal(storyID, 'story-1')
      return { ID: 'branch-1', Status: 'ACTIVE', Type: 'OFFICIAL' }
    },
    async listVersions(branchID) {
      assert.equal(branchID, 'branch-1')
      return { versions: [{ ID: 'v1', SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: 'rev-1' }] }
    },
    async commitApprovedRevision() { throw new Error('unexpected commit') },
  }

  const controller = createCanonWorkspaceController(api)
  const state = await controller.load({ storyID: 'story-1', chapterID: 'chapter-1', approvedRevision: { ID: 'rev-1' } })

  assert.equal(state.branch.ID, 'branch-1')
  assert.equal(state.readiness.state, 'CURRENT')
  assert.equal(state.readiness.latestVersion.ID, 'v1')
})

test('ignores stale A response after A → B → A selection cycle', async () => {
  const firstA = deferred()
  const secondA = deferred()
  let aCalls = 0
  const api = {
    getActiveOfficialBranch(storyID) {
      if (storyID === 'story-a') {
        aCalls += 1
        return aCalls === 1 ? firstA.promise : secondA.promise
      }
      return Promise.resolve({ ID: 'branch-b' })
    },
    async listVersions(branchID) { return { versions: [{ ID: `v-${branchID}`, SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: branchID === 'branch-a2' ? 'rev-a2' : 'rev-b' }] } },
    async commitApprovedRevision() { throw new Error('unexpected commit') },
  }

  const controller = createCanonWorkspaceController(api)
  const loadA1 = controller.load({ storyID: 'story-a', chapterID: 'chapter-a', approvedRevision: { ID: 'rev-a1' } })
  await controller.load({ storyID: 'story-b', chapterID: 'chapter-b', approvedRevision: { ID: 'rev-b' } })
  const loadA2 = controller.load({ storyID: 'story-a', chapterID: 'chapter-a', approvedRevision: { ID: 'rev-a2' } })

  secondA.resolve({ ID: 'branch-a2' })
  await loadA2
  firstA.resolve({ ID: 'branch-a1' })
  await loadA1

  const state = controller.snapshot()
  assert.equal(state.branch.ID, 'branch-a2')
  assert.equal(state.readiness.state, 'CURRENT')
  assert.equal(state.readiness.latestVersion.ID, 'v-branch-a2')
})

test('commit binds exact authority and refreshes persisted projection before becoming CURRENT', async () => {
  let versions = []
  const calls = []
  const api = {
    async getActiveOfficialBranch() { return { ID: 'branch-1' } },
    async listVersions() { return { versions } },
    async commitApprovedRevision(input) {
      calls.push(input)
      versions = [{ ID: 'v2', SequenceNo: 2, Status: 'OFFICIAL', SourceContentRevisionID: input.approvedRevisionID }]
      return versions[0]
    },
  }

  const controller = createCanonWorkspaceController(api)
  const before = await controller.load({ storyID: 'story-1', chapterID: 'chapter-9', approvedRevision: { ID: 'rev-42' } })
  assert.equal(before.readiness.state, 'READY_TO_COMMIT')

  const after = await controller.commit()
  assert.deepEqual(calls, [{ storyID: 'story-1', branchID: 'branch-1', chapterID: 'chapter-9', approvedRevisionID: 'rev-42' }])
  assert.equal(after.readiness.state, 'CURRENT')
  assert.equal(after.readiness.latestVersion.ID, 'v2')
})

test('commit uses the approved revision snapshot even if caller mutates its object after load', async () => {
  const approvedRevision = { ID: 'rev-original' }
  const calls = []
  let versions = []
  const api = {
    async getActiveOfficialBranch() { return { ID: 'branch-1' } },
    async listVersions() { return { versions } },
    async commitApprovedRevision(input) {
      calls.push(input)
      versions = [{ ID: 'v1', SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: input.approvedRevisionID }]
      return versions[0]
    },
  }

  const controller = createCanonWorkspaceController(api)
  await controller.load({ storyID: 'story-1', chapterID: 'chapter-1', approvedRevision })
  approvedRevision.ID = 'rev-mutated'

  const state = await controller.commit()

  assert.deepEqual(calls, [{ storyID: 'story-1', branchID: 'branch-1', chapterID: 'chapter-1', approvedRevisionID: 'rev-original' }])
  assert.equal(state.readiness.state, 'CURRENT')
  assert.equal(state.readiness.latestVersion.SourceContentRevisionID, 'rev-original')
})

test('duplicate commit is suppressed while the exact authority mutation is in flight', async () => {
  const pending = deferred()
  let commitCalls = 0
  const api = {
    async getActiveOfficialBranch() { return { ID: 'branch-1' } },
    async listVersions() { return { versions: [] } },
    async commitApprovedRevision() { commitCalls += 1; return pending.promise },
  }

  const controller = createCanonWorkspaceController(api)
  await controller.load({ storyID: 'story-1', chapterID: 'chapter-1', approvedRevision: { ID: 'rev-1' } })
  const first = controller.commit()
  const second = await controller.commit()

  assert.equal(commitCalls, 1)
  assert.equal(second.committing, true)
  pending.resolve({ ID: 'v1', SequenceNo: 1, Status: 'OFFICIAL', SourceContentRevisionID: 'rev-1' })
  await first
})

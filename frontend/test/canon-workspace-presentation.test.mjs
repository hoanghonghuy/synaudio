import test from 'node:test'
import assert from 'node:assert/strict'
import { presentCanonWorkspace } from '../src/features/admin/canonWorkspacePresentation.mjs'

test('LOADING is busy and exposes no unsafe action', () => {
  assert.deepEqual(presentCanonWorkspace({ readiness: { state: 'LOADING', label: 'loading' } }), {
    status: 'LOADING', badge: 'LOADING', summary: 'loading', detail: 'Đang xác minh ACTIVE OFFICIAL branch và CanonVersion authoritative.', canCommit: false, canRetry: false, busy: true,
  })
})

test('READY_TO_COMMIT exposes commit only when authoritative readiness allows it', () => {
  const ready = presentCanonWorkspace({ readiness: { state: 'READY_TO_COMMIT', label: 'ready', canCommit: true, latestVersion: { SequenceNo: 3, SourceContentRevisionID: 'rev-old' } }, committing: false })
  assert.equal(ready.badge, 'READY TO COMMIT')
  assert.equal(ready.canCommit, true)
  assert.equal(ready.canRetry, false)
  assert.match(ready.detail, /OFFICIAL v3/)
  assert.match(ready.detail, /rev-old/)

  const busy = presentCanonWorkspace({ readiness: { state: 'READY_TO_COMMIT', canCommit: true }, committing: true })
  assert.equal(busy.canCommit, false)
  assert.equal(busy.busy, true)
})

test('CURRENT reports the persisted OFFICIAL provenance and never allows recommit', () => {
  const current = presentCanonWorkspace({ readiness: { state: 'CURRENT', label: 'current', latestVersion: { SequenceNo: 4, SourceContentRevisionID: 'rev-4' } } })
  assert.equal(current.badge, 'CURRENT')
  assert.equal(current.canCommit, false)
  assert.equal(current.canRetry, false)
  assert.equal(current.detail, 'OFFICIAL v4 · source rev-4')
})

test('BLOCKED fails closed and retry is available only when authority is idle', () => {
  const blocked = presentCanonWorkspace({ error: 'CANON_BRANCH_NOT_FOUND', readiness: { state: 'BLOCKED', label: 'fallback' } })
  assert.equal(blocked.summary, 'CANON_BRANCH_NOT_FOUND')
  assert.equal(blocked.canCommit, false)
  assert.equal(blocked.canRetry, true)

  const loading = presentCanonWorkspace({ error: 'temporary', loading: true, readiness: { state: 'BLOCKED' } })
  assert.equal(loading.canRetry, false)
  assert.equal(loading.busy, true)
})

test('WAITING does not offer retry or commit before approved content exists', () => {
  const waiting = presentCanonWorkspace({ readiness: { state: 'WAITING', label: 'need approval' } })
  assert.equal(waiting.badge, 'WAITING')
  assert.equal(waiting.canCommit, false)
  assert.equal(waiting.canRetry, false)
  assert.equal(waiting.busy, false)
})

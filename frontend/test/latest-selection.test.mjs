import assert from 'node:assert/strict'
import test from 'node:test'

import { createLatestSelectionGuard } from '../src/features/admin/latestSelection.mjs'

function deferred() {
  let resolve
  const promise = new Promise((done) => { resolve = done })
  return { promise, resolve }
}

test('only the latest chapter selection may commit out-of-order async results', async () => {
  const guard = createLatestSelectionGuard()
  const slowA = deferred()
  const fastB = deferred()
  const committed = []

  const mayCommitA = guard.begin('chapter-a')
  const requestA = slowA.promise.then((value) => {
    if (mayCommitA()) committed.push(value)
  })

  const mayCommitB = guard.begin('chapter-b')
  const requestB = fastB.promise.then((value) => {
    if (mayCommitB()) committed.push(value)
  })

  fastB.resolve('chapter-b')
  await requestB
  slowA.resolve('chapter-a')
  await requestA

  assert.deepEqual(committed, ['chapter-b'])
})

test('a superseded request may not commit its error path', async () => {
  const guard = createLatestSelectionGuard()
  const mayCommitA = guard.begin('chapter-a')
  const mayCommitB = guard.begin('chapter-b')

  assert.equal(mayCommitA(), false)
  assert.equal(mayCommitB(), true)
})

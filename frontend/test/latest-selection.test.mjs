import assert from 'node:assert/strict'
import test from 'node:test'

import { createLatestSelectionGuard } from '../src/features/admin/latestSelection.mjs'

test('only latest chapter selection may commit', () => {
  const guard = createLatestSelectionGuard()
  const mayCommitA = guard.begin('chapter-a')
  const mayCommitB = guard.begin('chapter-b')
  assert.equal(mayCommitA(), false)
  assert.equal(mayCommitB(), true)
})

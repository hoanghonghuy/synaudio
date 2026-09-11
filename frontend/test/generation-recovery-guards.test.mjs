import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canRetryGenerationJob,
  createLatestSelectionGuard,
} from '../src/features/admin/latestSelection.mjs'

test('latest selection guard rejects stale A to B to A responses', () => {
  const guard = createLatestSelectionGuard()
  const firstA = guard.begin('chapter-a')
  const b = guard.begin('chapter-b')
  const secondA = guard.begin('chapter-a')

  assert.equal(firstA(), false)
  assert.equal(b(), false)
  assert.equal(secondA(), true)
})

test('retry authority fails closed while a retry action is already in progress', () => {
  const generationJob = { Observation: 'retryable', Retryable: true }

  assert.equal(canRetryGenerationJob({
    generationJob,
    selectionLoading: false,
    actionInProgress: false,
  }), true)

  assert.equal(canRetryGenerationJob({
    generationJob,
    selectionLoading: false,
    actionInProgress: true,
  }), false)

  assert.equal(canRetryGenerationJob({
    generationJob,
    selectionLoading: true,
    actionInProgress: false,
  }), false)
})

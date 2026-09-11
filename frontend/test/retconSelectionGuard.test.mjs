import assert from 'node:assert/strict'
import test from 'node:test'
import { createRetconSelectionGuard } from '../src/features/admin/retconSelectionGuard.mjs'

test('older detail response cannot replace the latest selected retcon', () => {
  const guard = createRetconSelectionGuard()
  const requestA = guard.select('retcon-a')
  const requestB = guard.select('retcon-b')

  assert.equal(guard.isCurrent(requestB), true)
  assert.equal(guard.isCurrent(requestA), false)
})

test('a repeated request for the same retcon invalidates the earlier generation', () => {
  const guard = createRetconSelectionGuard()
  const first = guard.select('retcon-a')
  const refreshed = guard.select('retcon-a')

  assert.equal(guard.isCurrent(first), false)
  assert.equal(guard.isCurrent(refreshed), true)
})

test('clearing the selection invalidates an in-flight detail response', () => {
  const guard = createRetconSelectionGuard()
  const request = guard.select('retcon-a')
  guard.clear()

  assert.equal(guard.isCurrent(request), false)
})

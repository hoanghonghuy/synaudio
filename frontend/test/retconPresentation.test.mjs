import assert from 'node:assert/strict'
import test from 'node:test'
import {
  retconActionIsDestructive,
  retconActions,
  retconStatusLabel,
} from '../src/features/admin/retconPresentation.mjs'

test('retcon lifecycle only exposes backend-valid actions', () => {
  assert.deepEqual(retconActions('DRAFT'), ['analyze', 'cancel'])
  assert.deepEqual(retconActions('ANALYZING'), ['approve', 'cancel'])
  assert.deepEqual(retconActions('APPROVED'), ['ready', 'cancel'])
  assert.deepEqual(retconActions('READY_TO_APPLY'), ['apply', 'cancel'])
  assert.deepEqual(retconActions('APPLIED'), [])
  assert.deepEqual(retconActions('CANCELLED'), [])
})

test('draft cannot be approved directly', () => {
  assert.equal(retconActions('DRAFT').includes('approve'), false)
})

test('destructive actions require explicit confirmation in the UI', () => {
  assert.equal(retconActionIsDestructive('apply'), true)
  assert.equal(retconActionIsDestructive('cancel'), true)
  assert.equal(retconActionIsDestructive('approve'), false)
})

test('all authoritative states have user-facing labels', () => {
  for (const status of ['DRAFT', 'ANALYZING', 'APPROVED', 'READY_TO_APPLY', 'APPLIED', 'CANCELLED']) {
    assert.notEqual(retconStatusLabel(status), status)
  }
})

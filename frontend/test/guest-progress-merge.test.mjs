import test from 'node:test'
import assert from 'node:assert/strict'

import { ApiRequestError } from '../src/api/http-error.ts'
import { mergeGuestProgressIfAbsent } from '../src/stores/guest-progress-merge.ts'

test('imports guest progress only when server explicitly reports not found', async () => {
  let imported = 0
  const outcome = await mergeGuestProgressIfAbsent(
    async () => {
      throw new ApiRequestError(404, 'progress not found', 'NOT_FOUND')
    },
    async () => {
      imported += 1
    },
  )

  assert.equal(outcome, 'imported')
  assert.equal(imported, 1)
})

test('transient server failure does not import or overwrite progress', async () => {
  let imported = 0
  const outcome = await mergeGuestProgressIfAbsent(
    async () => {
      throw new ApiRequestError(503, 'temporarily unavailable')
    },
    async () => {
      imported += 1
    },
  )

  assert.equal(outcome, 'deferred')
  assert.equal(imported, 0)
})

test('network uncertainty does not import or overwrite progress', async () => {
  let imported = 0
  const outcome = await mergeGuestProgressIfAbsent(
    async () => {
      throw new TypeError('network failed')
    },
    async () => {
      imported += 1
    },
  )

  assert.equal(outcome, 'deferred')
  assert.equal(imported, 0)
})

test('existing server progress always wins', async () => {
  let imported = 0
  const outcome = await mergeGuestProgressIfAbsent(
    async () => ({ positionMs: 9000 }),
    async () => {
      imported += 1
    },
  )

  assert.equal(outcome, 'server-wins')
  assert.equal(imported, 0)
})
